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
	"golang.org/x/sync/singleflight"
	"strings"
	"sync"
	"time"
)

const (
	signaturesMiddlewareName               = "signatures"
	signaturesMiddlewareLoadKeyBarrierName = "loading"
)

var (
	signaturesExchangeKeyPath        = []byte("/signatures/exchange_key")
	signaturesConfirmExchangeKeyPath = []byte("/signatures/confirm_exchange_key")
)

// SignatureMiddleware
//
// 使用SM2进行共享密钥交换，交换成功后双方使用共享密钥进行对签名与验证
// 签名方式为使用HMAC+XXHASH对path+body签名，使用HEX对签名编码。最终将签名赋值于X-Fns-Signature头。
// 错误:
// * 458: 验证签名错误
// * 468: 共享密钥超时
// * 488: 签名丢失
// * 448: 因发起方的秘密哈希不匹配而不同意密钥交换
//
// 交换共享密钥:
// 发起方进行发起交换共享密钥请求
// curl -H "Content-Type: application/json" -H "X-Fns-Device-Id: client-uuid" -X POST -d '{"publicKey":"pem string", "keyLength": 20}' http://ip:port/signatures/exchange_key
// 其中`X-Fns-Device-Id`的值是证书的编号，即证书必须是由响应方签发的。
// 响应方收到请求后，创建共享密钥，如果成功，则返回响应方的公钥，共享密钥有效期，响应方的共享密钥hash，发起方的共享密钥hash。
// 成功结果结果: `{"id": "responder id", "publicKey":"pem string", "expireAT": "RFC3339", "responderExchangeKeyHash": []byte}`
// 发起方拿到响应方的公钥，进行创建共享密钥，然后比对响应方的密钥hash，如果成功，则发起确认请求。
// curl -H "Content-Type: application/json" -H "X-Fns-Device-Id: client-uuid" -X POST -d '{"initiatorExchangeKeyHash":[]byte}' http://ip:port/signatures/confirm_exchange_key
// 响应方收到确认请求后，比对发起方的密钥HASH，如果成功，则同意协商结果。返回{"ok": true}
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
		group:          new(singleflight.Group),
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
	// DisableConfirm 是否关闭密钥交换确认过程
	DisableConfirm bool `json:"disableConfirm"`
}

type signatureMiddleware struct {
	store                     shareds.Store
	sigs                      sync.Map
	certificates              Certificates
	group                     *singleflight.Group
	certificateId             string
	exchangeKeyTTL            time.Duration
	publicPEM                 []byte
	publicKey                 *sm2.PublicKey
	privateKey                *sm2.PrivateKey
	expireAT                  time.Time
	exchangeKeyConfirmEnabled bool
}

func (middleware *signatureMiddleware) Name() (name string) {
	name = signaturesMiddlewareName
	return
}

func (middleware *signatureMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.store = options.Runtime.Shared().Store()
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
	middleware.exchangeKeyConfirmEnabled = !config.DisableConfirm
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
		if middleware.exchangeKeyConfirmEnabled {
			if r.IsPost() && bytes.Compare(r.Path(), signaturesConfirmExchangeKeyPath) == 0 {
				middleware.handleConfirmExchangeKey(w, r)
				return
			}
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
			if !sess.Agreed {
				w.Failed(ErrSharedSecretKeyNotAgreed)
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
	Id                       string    `json:"id"`
	PublicKey                string    `json:"publicKey"`
	ExpireAT                 time.Time `json:"expireAT"`
	ResponderExchangeKeyHash []byte    `json:"responderExchangeKeyHash"`
}

func (middleware *signatureMiddleware) handleExchangeKey(w transports.ResponseWriter, r *transports.Request) {
	_, loadErr, _ := middleware.group.Do(signaturesMiddlewareLoadKeyBarrierName, func() (interface{}, error) {
		return nil, middleware.loadSecretKey(r.Context())
	})
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
	// save session
	sessionKey = bytex.FromString(base64.URLEncoding.EncodeToString(sessionKey))
	sess := session{
		Agreed:                   false,
		Key:                      sessionKey,
		ExpireAT:                 time.Now().Add(middleware.exchangeKeyTTL),
		InitiatorExchangeKeyHash: nil,
		sig:                      nil,
	}
	if middleware.exchangeKeyConfirmEnabled {
		sess.InitiatorExchangeKeyHash = initiatorExchangeKeyHash
	} else {
		sess.Agreed = true
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
		Id:                       middleware.certificateId,
		PublicKey:                bytex.ToString(middleware.publicPEM),
		ExpireAT:                 sess.ExpireAT,
		ResponderExchangeKeyHash: responderExchangeKeyHash,
	})

	return
}

type signatureConfirmExchangeKeyParam struct {
	InitiatorExchangeKeyHash []byte `json:"initiatorExchangeKeyHash"`
}

type signatureConfirmExchangeKeyResult struct {
	Ok bool `json:"ok"`
}

func (middleware *signatureMiddleware) handleConfirmExchangeKey(w transports.ResponseWriter, r *transports.Request) {
	param := signatureConfirmExchangeKeyParam{}
	decodeErr := json.Unmarshal(r.Body(), &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("fns: confirm exchange key failed").WithCause(errors.Warning("param is invalid").WithCause(decodeErr)))
		return
	}
	if param.InitiatorExchangeKeyHash == nil || len(param.InitiatorExchangeKeyHash) == 0 {
		w.Failed(errors.Warning("fns: confirm exchange key failed").WithCause(errors.Warning("param is invalid")))
		return
	}
	deviceId := r.Header().Get(httpDeviceIdHeader)
	sessBytes, existSess, getErr := middleware.store.Get(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
	if getErr != nil {
		w.Failed(errors.Warning("fns: get exchange key session failed").WithCause(getErr))
		return
	}
	if !existSess {
		w.Failed(errors.Warning("fns: exchange key session was not started"))
		return
	}
	sess := session{}
	decodeSessionErr := json.Unmarshal(sessBytes, &sess)
	if decodeSessionErr != nil {
		_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
		w.Failed(errors.Warning("fns: decode exchange key session failed").WithCause(decodeSessionErr))
		return
	}
	if bytes.Compare(sess.InitiatorExchangeKeyHash, param.InitiatorExchangeKeyHash) != 0 {
		w.Succeed(&signatureConfirmExchangeKeyResult{
			Ok: false,
		})
		return
	}
	exchangeKeyTTL := sess.ExpireAT.Sub(time.Now())
	if exchangeKeyTTL < 1 {
		_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
		w.Failed(errors.Warning("fns: confirm exchange key failed").WithCause(errors.Warning("expired")))
		return
	}
	sess.Agreed = true
	sessBytes, _ = json.Marshal(&sess)
	setErr := middleware.store.SetWithTTL(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)), sessBytes, exchangeKeyTTL)
	if setErr != nil {
		_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
		w.Failed(errors.Warning("fns: confirm exchange key failed").WithCause(errors.Warning("update session failed").WithCause(setErr)))
		return
	}
	w.Succeed(&signatureConfirmExchangeKeyResult{
		Ok: true,
	})
	return
}

func (middleware *signatureMiddleware) Close() (err error) {
	return
}

type session struct {
	Agreed                   bool      `json:"agreed"`
	Key                      []byte    `json:"key"`
	ExpireAT                 time.Time `json:"expireAT"`
	InitiatorExchangeKeyHash []byte    `json:"initiatorExchangeKeyHash"`
	sig                      signatures.Signer
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
