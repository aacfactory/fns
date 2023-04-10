/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"encoding/base64"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/cryptos/sm2"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/service/shareds"
	"github.com/aacfactory/fns/service/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"strings"
	"sync"
	"time"
)

const (
	signatureMiddlewareName = "signature"
)

var (
	ErrSignatureLost       = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
	ErrSessionOutOfDate    = errors.New(468, "***SESSION OUT OF DATE***", "need to create session")
)

type signatureMiddlewareConfig struct {
	Issuer             string `json:"issuer"`
	PublicKey          string `json:"publicKey"`
	PrivateKey         string `json:"privateKey"`
	PrivateKeyPassword string `json:"privateKeyPassword"`
	SessionKeyTTL      string `json:"sessionKeyTTL"`
}

func SignatureMiddleware() TransportMiddleware {
	return &signatureMiddleware{}
}

// signatureMiddleware
// 使用SM2进行共享密钥交换，交换成功后客户都使用hmac来签名，服务端来验证
// todo 提供service，处理client app的公钥合法性，包含签发和验证。
// errors:
// * 458: 签名错误，session key 不是共享的
// * 468: session 超时
// * 488: 签名丢失
type signatureMiddleware struct {
	store         shareds.Store
	sigs          sync.Map
	pub           *sm2.PublicKey
	pri           *sm2.PrivateKey
	issuer        string
	sessionKeyTTL time.Duration
}

func (middleware *signatureMiddleware) Name() (name string) {
	name = signatureMiddlewareName
	return
}

func (middleware *signatureMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.store = options.Shared.Store()
	middleware.sigs = sync.Map{}
	config := signatureMiddlewareConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: signature middleware build failed").WithCause(configErr)
		return
	}
	middleware.issuer = strings.TrimSpace(config.Issuer)
	if middleware.issuer == "" {
		middleware.issuer = "FNS"
	}
	middleware.pub, err = sm2.ParsePublicKey([]byte(strings.TrimSpace(config.PublicKey)))
	if err != nil {
		err = errors.Warning("fns: signature middleware build failed").WithCause(errors.Warning("parse public key failed")).WithCause(err)
		return
	}
	if config.PrivateKeyPassword == "" {
		middleware.pri, err = sm2.ParsePrivateKey([]byte(strings.TrimSpace(config.PrivateKey)))
	} else {
		middleware.pri, err = sm2.ParsePrivateKeyWithPassword([]byte(strings.TrimSpace(config.PrivateKey)), []byte(strings.TrimSpace(config.PrivateKeyPassword)))
	}
	if err != nil {
		err = errors.Warning("fns: signature middleware build failed").WithCause(errors.Warning("parse private key failed")).WithCause(err)
		return
	}
	if config.SessionKeyTTL != "" {
		middleware.sessionKeyTTL, err = time.ParseDuration(strings.TrimSpace(config.SessionKeyTTL))
		if err != nil {
			err = errors.Warning("fns: signature middleware build failed").WithCause(errors.Warning("sessionKeyTTL must be time.Duration format")).WithCause(err)
			return
		}
	}
	if middleware.sessionKeyTTL < 1 {
		middleware.sessionKeyTTL = 24 * time.Hour
	}
	return
}

func (middleware *signatureMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r *transports.Request) {
		// discard not service fn request
		if !r.IsPost() {
			next.Handle(w, r)
			return
		}
		// discard internal request
		if r.Header().Get(httpRequestInternalSignatureHeader) != "" {
			next.Handle(w, r)
			return
		}
		// discard dev request
		if r.Header().Get(httpDevModeHeader) != "" {
			next.Handle(w, r)
			return
		}
		// discard upgrade request
		if r.Header().Get(httpDevModeHeader) != "" {
			next.Handle(w, r)
			return
		}
		if r.IsPost() && bytex.ToString(r.Path()) == "/signatures/session" {
			middleware.handleCreateSession(w, r)
			return
		}

		signature := r.Header().Get(httpSignatureHeader)
		if signature == "" {
			w.Failed(ErrSignatureLost)
			return
		}
		deviceId := r.Header().Get(httpDeviceIdHeader)
		x, loaded := middleware.sigs.Load(deviceId)
		if !loaded {
			sessionBytes, hasExchanged, getErr := middleware.store.Get(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
			if getErr != nil {
				w.Failed(errors.Warning("fns: get session key from shared store failed").WithCause(getErr))
				return
			}
			if !hasExchanged {
				w.Failed(ErrSessionOutOfDate)
				return
			}
			sess := session{}
			decodeErr := json.Unmarshal(sessionBytes, &sess)
			if decodeErr != nil {
				w.Failed(ErrSessionOutOfDate.WithCause(decodeErr))
				return
			}
			if sess.Deadline.Before(time.Now()) || sess.Key == nil || len(sess.Key) == 0 {
				_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
				w.Failed(ErrSessionOutOfDate)
				return
			}
			sess.sig = signatures.HMAC(sess.Key)
			middleware.sigs.Store(deviceId, &sess)
			x = &sess
		}
		sess := x.(*session)
		if sess.Deadline.Before(time.Now()) {
			_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
			middleware.sigs.Delete(deviceId)
			w.Failed(ErrSessionOutOfDate)
			return
		}
		buf := bytebufferpool.Get()
		_, _ = buf.Write(r.Path())
		if r.Body() != nil && len(r.Body()) > 0 {
			_, _ = buf.Write(r.Body())
		}
		source := buf.Bytes()
		bytebufferpool.Put(buf)
		if !sess.sig.Verify(source, bytex.FromString(signature)) {
			w.Failed(ErrSignatureUnverified)
			return
		}
		next.Handle(w, r)
		w.Header().Set(httpSignatureHeader, bytex.ToString(sess.sig.Sign(w.Body())))
	})
}

type signatureSessionCreateParam struct {
	PublicKey string `json:"publicKey"`
}

// todo 返回app的公钥，
type signatureSessionCreateResult struct {
	SessionKey string    `json:"sessionKey"`
	Deadline   time.Time `json:"deadline"`
}

func (middleware *signatureMiddleware) handleCreateSession(w transports.ResponseWriter, r *transports.Request) {
	param := signatureSessionCreateParam{}
	decodeErr := json.Unmarshal(r.Body(), &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("fns: create session failed").WithCause(errors.Warning("param is invalid").WithCause(decodeErr)))
		return
	}
	pub, pubErr := sm2.ParsePublicKey(bytex.FromString(strings.TrimSpace(param.PublicKey)))
	if pubErr != nil {
		w.Failed(errors.Warning("fns: create session failed").WithCause(errors.Warning("publicKey is invalid").WithCause(pubErr)))
		return
	}
	deviceId := r.Header().Get(httpDeviceIdHeader)

	sessionKey, _, _, exchangeErr := sm2.KeyExchangeB(20, bytex.FromString(deviceId), bytex.FromString(middleware.issuer), middleware.pri, pub, middleware.pri, pub)
	if exchangeErr != nil {
		w.Failed(errors.Warning("fns: create session failed").WithCause(errors.Warning("exchange key failed").WithCause(exchangeErr)))
		return
	}
	sessionKey = bytex.FromString(base64.URLEncoding.EncodeToString(sessionKey))
	sess := session{
		Key:      sessionKey,
		Deadline: time.Now().Add(middleware.sessionKeyTTL),
		sig:      nil,
	}
	sessBytes, _ := json.Marshal(&sess)

	setErr := middleware.store.SetWithTTL(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)), sessBytes, middleware.sessionKeyTTL)
	if setErr != nil {
		w.Failed(errors.Warning("fns: create session failed").WithCause(errors.Warning("save session failed").WithCause(setErr)))
		return
	}
	sess.sig = signatures.HMAC(sess.Key)
	middleware.sigs.Store(deviceId, &sess)

	w.Succeed(&signatureSessionCreateResult{
		SessionKey: bytex.ToString(sess.Key),
		Deadline:   sess.Deadline,
	})

	return
}

func (middleware *signatureMiddleware) Close() (err error) {
	return
}

type session struct {
	Key      []byte    `json:"key"`
	Deadline time.Time `json:"deadline"`
	sig      signatures.Signer
}
