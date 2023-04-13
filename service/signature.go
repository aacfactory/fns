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
	"bytes"
	"context"
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
	signaturesMiddlewareName = "signatures"
)

var (
	signaturesExchangeKeyPath = []byte("/signatures/exchange_key")
)

// SignatureMiddleware
//
// 使用SM2进行共享密钥交换，交换成功后双方使用共享密钥进行对签名与验证
// 签名方式为使用HMAC+XXHASH对path+body签名，使用HEX对签名编码。最终将签名赋值于X-Fns-Signature头。
// 错误:
// * 458: 验证签名错误
// * 468: 共享密钥超时
// * 488: 签名丢失
//
// 交换共享密钥:
// 发起方进行发起交换共享密钥请求
// curl -H "Content-Type: application/json" -H "X-Fns-Device-Id: client-uuid" -X POST -d '{"publicKey":"pem string", "keyLength": 20}' http://ip:port/signatures/exchange_key
// 其中`X-Fns-Device-Id`的值是证书的编号，即证书必须是由响应方签发的。
// 响应方收到请求后，创建共享密钥，如果成功，则返回响应方的公钥，共享密钥有效期，响应方的共享密钥hash，发起方的共享密钥hash。
// 成功结果结果: `{"publicKey":"pem string", "expireAT": "RFC3339", "responderExchangeKeyHash": []byte, "initiatorExchangeKeyHash": []byte}`
// 发起方拿到响应方的公钥，进行创建共享密钥，然后比对hash。
//
// certificates : 证书仓库
func SignatureMiddleware(certificates Certificates) TransportMiddleware {
	if certificates == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: signatures middleware require certificates")))
		return nil
	}
	return &signatureMiddleware{
		store:          nil,
		sigs:           sync.Map{},
		certificates:   certificates,
		locker:         new(sync.Mutex),
		certificateId:  "",
		exchangeKeyTTL: 0,
		publicPEM:      nil,
		publicKey:      nil,
		privateKey:     nil,
	}
}

type signatureMiddlewareConfig struct {
	// CertificateId 响应方的证书编号，其证书类型必须是SM2
	CertificateId string `json:"certificateId"`
	// ExchangeKeyTTL 共享密钥的有效时长
	ExchangeKeyTTL string `json:"exchangeKeyTTL"`
}

type signatureMiddleware struct {
	store          shareds.Store
	sigs           sync.Map
	certificates   Certificates
	locker         sync.Locker
	certificateId  string
	exchangeKeyTTL time.Duration
	publicPEM      []byte
	publicKey      *sm2.PublicKey
	privateKey     *sm2.PrivateKey
	expireAT       time.Time
}

func (middleware *signatureMiddleware) Name() (name string) {
	name = signaturesMiddlewareName
	return
}

func (middleware *signatureMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.store = options.Shared.Store()
	config := signatureMiddlewareConfig{}
	err = options.Config.As(&config)
	if err != nil {
		err = errors.Warning("fns: build signatures middleware failed").WithCause(err)
		return
	}
	middleware.certificateId = strings.TrimSpace(config.CertificateId)
	if middleware.certificateId == "" {
		err = errors.Warning("fns: build signatures middleware failed").WithCause(errors.Warning("certificateId is required"))
		return
	}
	if config.ExchangeKeyTTL != "" {
		middleware.exchangeKeyTTL, err = time.ParseDuration(strings.TrimSpace(config.ExchangeKeyTTL))
		if err != nil {
			err = errors.Warning("fns: build signatures middleware failed").WithCause(errors.Warning("exchangeKeyTTL must be time.Duration format"))
			return
		}
	}
	if middleware.exchangeKeyTTL < 1 {
		middleware.exchangeKeyTTL = 24 * time.Hour
	}
	return
}

func (middleware *signatureMiddleware) loadSecretKey(ctx context.Context) (err error) {
	if middleware.publicPEM != nil {
		if !middleware.expireAT.IsZero() && middleware.expireAT.Before(time.Now()) {
			middleware.publicPEM = nil
			err = middleware.loadSecretKey(ctx)
			return
		}
		return
	}
	certificate, getErr := middleware.certificates.Get(ctx, middleware.certificateId)
	if getErr != nil {
		err = errors.Warning("fns: get certificate failed").WithCause(getErr).WithMeta("id", middleware.certificateId)
		return
	}
	if certificate == nil {
		err = errors.Warning("fns: get certificate failed").WithCause(errors.Warning("certificate was not found")).WithMeta("id", middleware.certificateId)
		return
	}
	if strings.ToUpper(certificate.Kind()) != "SM2" {
		err = errors.Warning("fns: get certificate failed").WithCause(errors.Warning("certificate kind must be SM2")).WithMeta("id", middleware.certificateId)
		return
	}
	expireAT := certificate.ExpireAT()
	if !expireAT.IsZero() && expireAT.Before(time.Now()) {
		err = errors.Warning("fns: get certificate failed").WithCause(errors.Warning("out of date")).WithMeta("id", middleware.certificateId)
		return
	}
	pubKey, parsePubErr := sm2.ParsePublicKey(certificate.Key())
	if parsePubErr != nil {
		err = errors.Warning("fns: get certificate failed").WithCause(errors.Warning("parse public key failed").WithCause(parsePubErr)).WithMeta("id", middleware.certificateId)
		return
	}
	var parsePriEr error
	var priKey *sm2.PrivateKey
	if certificate.Password() != nil && len(certificate.Password()) > 0 {
		priKey, parsePriEr = sm2.ParsePrivateKeyWithPassword(certificate.SecretKey(), certificate.Password())
	} else {
		priKey, parsePriEr = sm2.ParsePrivateKey(certificate.SecretKey())
	}
	if parsePriEr != nil {
		err = errors.Warning("fns: get certificate failed").WithCause(errors.Warning("parse private key failed").WithCause(parsePriEr)).WithMeta("id", middleware.certificateId)
		return
	}
	middleware.publicPEM = certificate.Key()
	middleware.publicKey = pubKey
	middleware.privateKey = priKey
	middleware.expireAT = expireAT
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
		if r.IsPost() && bytes.Compare(r.Path(), signaturesExchangeKeyPath) == 0 {
			middleware.handleExchangeKey(w, r)
			return
		}

		signature := r.Header().Get(httpSignatureHeader)
		if signature == "" {
			w.Failed(ErrSignatureLost)
			return
		}
		// get signer
		deviceId := r.Header().Get(httpDeviceIdHeader)
		x, loaded := middleware.sigs.Load(deviceId)
		if !loaded {
			sessionBytes, hasExchanged, getErr := middleware.store.Get(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
			if getErr != nil {
				w.Failed(errors.Warning("fns: get session key from shared store failed").WithCause(getErr))
				return
			}
			if !hasExchanged {
				w.Failed(ErrSharedSecretKeyOutOfDate)
				return
			}
			sess := session{}
			decodeErr := json.Unmarshal(sessionBytes, &sess)
			if decodeErr != nil {
				w.Failed(ErrSharedSecretKeyOutOfDate.WithCause(decodeErr))
				return
			}
			if sess.ExpireAT.Before(time.Now()) || sess.Key == nil || len(sess.Key) == 0 {
				_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
				w.Failed(ErrSharedSecretKeyOutOfDate)
				return
			}
			sess.sig = signatures.HMAC(sess.Key)
			middleware.sigs.Store(deviceId, &sess)
			x = &sess
		}
		sess := x.(*session)
		if sess.ExpireAT.Before(time.Now()) {
			_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
			middleware.sigs.Delete(deviceId)
			w.Failed(ErrSharedSecretKeyOutOfDate)
			return
		}
		// verify
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
		// next
		next.Handle(w, r)
		// write signature
		buf = bytebufferpool.Get()
		_, _ = buf.Write(r.Path())
		_, _ = buf.Write(w.Body())
		respSignature := sess.sig.Sign(buf.Bytes())
		bytebufferpool.Put(buf)
		w.Header().Set(httpSignatureHeader, bytex.ToString(respSignature))
	})
}

type signatureExchangeKeyParam struct {
	PublicKey string `json:"publicKey"`
	KeyLength int    `json:"keyLength"`
}

type signatureExchangeKeyResult struct {
	PublicKey                string    `json:"publicKey"`
	ExpireAT                 time.Time `json:"expireAT"`
	ResponderExchangeKeyHash []byte    `json:"responderExchangeKeyHash"`
	InitiatorExchangeKeyHash []byte    `json:"initiatorExchangeKeyHash"`
}

func (middleware *signatureMiddleware) handleExchangeKey(w transports.ResponseWriter, r *transports.Request) {
	middleware.locker.Lock()
	loadErr := middleware.loadSecretKey(r.Context())
	middleware.locker.Unlock()
	if loadErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(loadErr))
		return
	}
	// device
	deviceId := r.Header().Get(httpDeviceIdHeader)
	certificate, getErr := middleware.certificates.Get(r.Context(), deviceId)
	if getErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("get device certificate failed").WithCause(getErr).WithMeta("id", deviceId)))
		return
	}
	if certificate != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("device certificate was not issued").WithMeta("id", deviceId)))
		return
	}
	param := signatureExchangeKeyParam{}
	decodeErr := json.Unmarshal(r.Body(), &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("param is invalid").WithCause(decodeErr)))
		return
	}
	if param.PublicKey != bytex.ToString(certificate.Key()) {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("device certificate is invalid").WithMeta("id", deviceId)))
		return
	}
	devPub, parseDevPubErr := sm2.ParsePublicKey(bytex.FromString(strings.TrimSpace(param.PublicKey)))
	if parseDevPubErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("parse device public key failed").WithCause(parseDevPubErr)))
		return
	}
	keyLength := param.KeyLength
	if keyLength < 20 || keyLength > 64 {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("key length is invalid, must greater than 19 and less than 65")))
		return
	}
	// exchange
	sessionKey, responderExchangeKeyHash, initiatorExchangeKeyHash, exchangeErr := sm2.KeyExchangeResponder(
		keyLength,
		bytex.FromString(deviceId), bytex.FromString(middleware.certificateId),
		middleware.privateKey, devPub, middleware.privateKey, devPub,
	)
	if exchangeErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("exchange key failed").WithCause(exchangeErr)))
		return
	}
	sessionKey = bytex.FromString(base64.URLEncoding.EncodeToString(sessionKey))
	sess := session{
		Key:      sessionKey,
		ExpireAT: time.Now().Add(middleware.exchangeKeyTTL),
		sig:      nil,
	}
	sessBytes, _ := json.Marshal(&sess)

	setErr := middleware.store.SetWithTTL(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)), sessBytes, middleware.exchangeKeyTTL)
	if setErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("save session failed").WithCause(setErr)))
		return
	}
	sess.sig = signatures.HMAC(sess.Key)
	middleware.sigs.Store(deviceId, &sess)

	w.Succeed(&signatureExchangeKeyResult{
		PublicKey:                bytex.ToString(middleware.publicPEM),
		ExpireAT:                 sess.ExpireAT,
		ResponderExchangeKeyHash: responderExchangeKeyHash,
		InitiatorExchangeKeyHash: initiatorExchangeKeyHash,
	})

	return
}

func (middleware *signatureMiddleware) Close() (err error) {
	return
}

type session struct {
	Key      []byte    `json:"key"`
	ExpireAT time.Time `json:"expireAT"`
	sig      signatures.Signer
}

type Certificate interface {
	Id() string
	Kind() string
	Key() []byte
	SecretKey() []byte
	Password() []byte
	ExpireAT() time.Time
}

type Certificates interface {
	Get(ctx context.Context, id string) (certificate Certificate, err errors.CodeError)
}
