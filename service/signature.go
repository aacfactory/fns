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
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	signatureMiddlewareName = "signature"
)

var (
	ErrSignatureLost            = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified      = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
	ErrSharedSecretKeyOutOfDate = errors.New(468, "***SHARED SECRET KEY OUT OF DATE***", "need to recreate shared secret key")
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
// 响应方收到请求后，创建共享密钥，如果成功，则返回响应方的公钥，共享密钥有效期，响应方的共享密钥hash，发起方的共享密钥hash。
// 成功结果结果: `{"publicPem":"pem string", "deadline": "RFC3339", "responderExchangeKeyHash": []byte, "initiatorExchangeKeyHash": []byte}`
// 发起方拿到响应方的公钥，进行创建共享密钥，然后比对hash。
//
// certificates : 证书仓库，签名用的类型为`signatures`
func SignatureMiddleware(certificates Certificates) TransportMiddleware {
	return &signatureMiddleware{
		certificates: certificates,
	}
}

type signatureMiddleware struct {
	store        shareds.Store
	sigs         sync.Map
	certificates Certificates
}

func (middleware *signatureMiddleware) Name() (name string) {
	name = signatureMiddlewareName
	return
}

func (middleware *signatureMiddleware) Build(options TransportMiddlewareOptions) (err error) {
	middleware.store = options.Shared.Store()
	middleware.sigs = sync.Map{}
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
			if sess.Deadline.Before(time.Now()) || sess.Key == nil || len(sess.Key) == 0 {
				_ = middleware.store.Remove(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)))
				w.Failed(ErrSharedSecretKeyOutOfDate)
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
	PublicPEM                string    `json:"publicPem"`
	Deadline                 time.Time `json:"deadline"`
	ResponderExchangeKeyHash []byte    `json:"responderExchangeKeyHash"`
	InitiatorExchangeKeyHash []byte    `json:"initiatorExchangeKeyHash"`
}

func (middleware *signatureMiddleware) handleExchangeKey(w transports.ResponseWriter, r *transports.Request) {
	// device
	param := signatureExchangeKeyParam{}
	decodeErr := json.Unmarshal(r.Body(), &param)
	if decodeErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("param is invalid").WithCause(decodeErr)))
		return
	}
	keyLength := param.KeyLength
	if keyLength < 20 || keyLength > 64 {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("key length is invalid, must greater than 19 and less than 65")))
		return
	}
	devPub, parseDevPubErr := sm2.ParsePublicKey(bytex.FromString(strings.TrimSpace(param.PublicKey)))
	if parseDevPubErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("parse device public key failed").WithCause(parseDevPubErr)))
		return
	}
	deviceId := r.Header().Get(httpDeviceIdHeader)
	// get root
	appId, pubPEM, priPEM, passwd, exTTL, has, getErr := middleware.certificates.Root(r.Context(), signatureCertificateKind)
	if getErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("get root certificate failed")).WithCause(getErr))
		return
	}
	if !has {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("root certificate was not exists")))
		return
	}
	if appId == "" {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("app id of root certificate was not exists")))
		return
	}
	if exTTL < 1 {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("exchange key ttl of root certificate was not exists")))
		return
	}
	var parsePriEr error
	var pri *sm2.PrivateKey
	if passwd != nil && len(passwd) > 0 {
		pri, parsePriEr = sm2.ParsePrivateKeyWithPassword(priPEM, passwd)
	} else {
		pri, parsePriEr = sm2.ParsePrivateKey(priPEM)
	}
	if parsePriEr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("parse private key of root certificate failed").WithCause(parsePriEr)))
		return
	}

	// exchange
	sessionKey, responderExchangeKeyHash, initiatorExchangeKeyHash, exchangeErr := sm2.KeyExchangeResponder(keyLength, bytex.FromString(deviceId), bytex.FromString(appId), pri, devPub, pri, devPub)
	if exchangeErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("exchange key failed").WithCause(exchangeErr)))
		return
	}
	sessionKey = bytex.FromString(base64.URLEncoding.EncodeToString(sessionKey))
	sess := session{
		Key:      sessionKey,
		Deadline: time.Now().Add(exTTL),
		sig:      nil,
	}
	sessBytes, _ := json.Marshal(&sess)

	setErr := middleware.store.SetWithTTL(r.Context(), bytex.FromString(fmt.Sprintf("fns/signatures/sessions/%s", deviceId)), sessBytes, exTTL)
	if setErr != nil {
		w.Failed(errors.Warning("fns: exchange key failed").WithCause(errors.Warning("save session failed").WithCause(setErr)))
		return
	}
	sess.sig = signatures.HMAC(sess.Key)
	middleware.sigs.Store(deviceId, &sess)

	w.Succeed(&signatureExchangeKeyResult{
		PublicPEM:                bytex.ToString(pubPEM),
		Deadline:                 sess.Deadline,
		ResponderExchangeKeyHash: responderExchangeKeyHash,
		InitiatorExchangeKeyHash: initiatorExchangeKeyHash,
	})

	return
}

func (middleware *signatureMiddleware) Close() (err error) {
	return
}

func (middleware *signatureMiddleware) Services() (v []Service) {
	if middleware.certificates == nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: certificates of signature middleware is required")))
		return
	}
	if reflect.TypeOf(middleware.certificates).Kind() != reflect.Pointer {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: certificates of signature middleware must be pointer")))
		return
	}
	v = []Service{
		&certificatesService{
			Abstract:     NewAbstract(certificatesServiceName, true, middleware.certificates),
			certificates: middleware.certificates,
		},
	}
	return
}

type session struct {
	Key      []byte    `json:"key"`
	Deadline time.Time `json:"deadline"`
	sig      signatures.Signer
}

const (
	certificatesServiceName  = "certificates"
	signatureCertificateKind = "signatures"
)

type Certificates interface {
	Component
	Root(ctx context.Context, kind string) (appId string, publicPEM []byte, privatePEM []byte, password []byte, exchangeKeyTTL time.Duration, has bool, err errors.CodeError)
	SaveRoot(ctx context.Context, kind string, appId string, publicPEM []byte, privatePEM []byte, password []byte, exchangeKeyTTL time.Duration) (err errors.CodeError)
	Issue(ctx context.Context, appId string, appName string, expirations time.Duration) (publicPEM []byte, privatePEM []byte, err errors.CodeError)
	Revoke(ctx context.Context, appId string) (err errors.CodeError)
	Verify(ctx context.Context, publicPEM []byte) (ok bool, err errors.CodeError)
	SaveExchangeKey(ctx context.Context, appId string, exchangeKey []byte, ttl time.Duration) (err errors.CodeError)
	GetExchangeKey(ctx context.Context, appId string) (exchangeKey []byte, has bool, err errors.CodeError)
}

type certificatesRootParam struct {
	Kind string `json:"kind"`
}

type certificatesRootResult struct {
	Has            bool          `json:"has"`
	Kind           string        `json:"kind"`
	AppId          string        `json:"appId"`
	PublicPEM      []byte        `json:"publicPem"`
	PrivatePEM     []byte        `json:"privatePem"`
	Password       []byte        `json:"password"`
	ExchangeKeyTTL time.Duration `json:"exchangeKeyTtl"`
}

type certificatesSaveRootParam struct {
	Kind           string        `json:"kind"`
	AppId          string        `json:"appId"`
	PublicPEM      []byte        `json:"publicPem"`
	PrivatePEM     []byte        `json:"privatePem"`
	Password       []byte        `json:"password"`
	ExchangeKeyTTL time.Duration `json:"exchangeKeyTtl"`
}

type certificatesIssueParam struct {
	AppId      string `json:"appId"`
	AppName    string `json:"appName"`
	ExpireDays int    `json:"expireDays"`
}

type certificatesIssueResult struct {
	PublicPEM  []byte `json:"publicPem"`
	PrivatePEM []byte `json:"privatePem"`
}

type certificatesRevokeParam struct {
	AppId string `json:"appId"`
}

type certificatesVerifyParam struct {
	PublicPEM []byte `json:"publicPem"`
}

type certificatesVerifyResult struct {
	Ok bool `json:"ok"`
}

type certificatesSaveExchangeKeyParam struct {
	AppId       string        `json:"appId"`
	ExchangeKey []byte        `json:"exchangeKey"`
	TTL         time.Duration `json:"ttl"`
}

type certificatesGetExchangeKeyParam struct {
	AppId string `json:"appId"`
}

type certificatesGetExchangeKeyResult struct {
	Has         bool   `json:"has"`
	ExchangeKey []byte `json:"exchangeKey"`
}

type certificatesService struct {
	Abstract
	certificates Certificates
}

func (svc *certificatesService) Handle(ctx context.Context, fn string, arg Argument) (v interface{}, err errors.CodeError) {
	switch fn {
	case "root":
		param := certificatesRootParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode get root param failed").WithCause(paramErr)
			return
		}
		appId, publicPEM, privatePEM, password, exchangeKeyTTL, has, rootErr := svc.certificates.Root(ctx, param.Kind)
		if rootErr != nil {
			err = rootErr
			return
		}
		v = certificatesRootResult{
			Has:            has,
			Kind:           param.Kind,
			AppId:          appId,
			PublicPEM:      publicPEM,
			PrivatePEM:     privatePEM,
			Password:       password,
			ExchangeKeyTTL: exchangeKeyTTL,
		}
		break
	case "save_root":
		param := certificatesSaveRootParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode save root param failed").WithCause(paramErr)
			return
		}
		err = svc.certificates.SaveRoot(ctx, param.Kind, param.AppId, param.PublicPEM, param.PrivatePEM, param.Password, param.ExchangeKeyTTL)
		break
	case "issue":
		param := certificatesIssueParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode issue param failed").WithCause(paramErr)
			return
		}
		pub, pri, issueErr := svc.certificates.Issue(ctx, param.AppId, param.AppName, time.Duration(param.ExpireDays*24)*time.Hour)
		if issueErr != nil {
			err = issueErr
			return
		}
		v = certificatesIssueResult{
			PublicPEM:  pub,
			PrivatePEM: pri,
		}
		break
	case "revoke":
		param := certificatesRevokeParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode revoke param failed").WithCause(paramErr)
			return
		}
		err = svc.certificates.Revoke(ctx, param.AppId)
		break
	case "verify":
		param := certificatesVerifyParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode verify param failed").WithCause(paramErr)
			return
		}
		ok, verifyErr := svc.certificates.Verify(ctx, param.PublicPEM)
		if verifyErr != nil {
			err = verifyErr
			return
		}
		v = certificatesVerifyResult{
			Ok: ok,
		}
		break
	case "save_exchange_key":
		param := certificatesSaveExchangeKeyParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode save exchange key param failed").WithCause(paramErr)
			return
		}
		err = svc.certificates.SaveExchangeKey(ctx, param.AppId, param.ExchangeKey, param.TTL)
		break
	case "get_exchange_key":
		param := certificatesGetExchangeKeyParam{}
		paramErr := arg.As(&param)
		if paramErr != nil {
			err = errors.Warning("fns: decode get exchange key param failed").WithCause(paramErr)
			return
		}
		key, has, getErr := svc.certificates.GetExchangeKey(ctx, param.AppId)
		if getErr != nil {
			err = getErr
			return
		}
		v = certificatesGetExchangeKeyResult{
			Has:         has,
			ExchangeKey: key,
		}
		break
	default:
		err = errors.NotFound("fns: not found")
		break
	}
	return
}
