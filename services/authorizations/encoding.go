/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package authorizations

import (
	"bytes"
	"encoding/base64"
	"github.com/aacfactory/avro"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"strings"
)

type TokenEncoder interface {
	services.Component
	Encode(ctx context.Context, param Authorization) (token Token, err error)
	Decode(ctx context.Context, token Token) (result Authorization, err error)
}

type hmacTokenEncoderConfig struct {
	Key string `json:"key" yaml:"key"`
}

func HmacTokenEncoder() TokenEncoder {
	return &hmacTokenEncoder{}
}

type hmacTokenEncoder struct {
	signature signatures.Signature
}

func (encoder *hmacTokenEncoder) Name() (name string) {
	return "encoder"
}

func (encoder *hmacTokenEncoder) Construct(options services.Options) (err error) {
	config := hmacTokenEncoderConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("authorizations: build default token encoder failed").WithMeta("encoder", "hmac").WithCause(configErr)
		return
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		err = errors.Warning("authorizations: build default token encoder failed").WithMeta("encoder", "hmac").WithCause(errors.Warning("key is require"))
		return
	}
	encoder.signature = signatures.HMAC([]byte(key))
	return
}

func (encoder *hmacTokenEncoder) Shutdown(_ context.Context) {
	return
}

func (encoder *hmacTokenEncoder) Encode(_ context.Context, param Authorization) (token Token, err error) {
	p, encodeErr := avro.Marshal(param)
	if encodeErr != nil {
		err = errors.Warning("authorizations: encode token failed").WithMeta("encoder", "hmac").WithCause(encodeErr)
		return
	}
	pb := make([]byte, base64.URLEncoding.EncodedLen(len(p)))
	base64.URLEncoding.Encode(pb, p)
	pbl := len(pb)

	s := encoder.signature.Sign(p)
	sb := make([]byte, base64.URLEncoding.EncodedLen(len(s)))
	base64.URLEncoding.Encode(sb, s)
	sbl := len(sb)

	token = make(Token, 4+pbl+1+sbl)
	token[0] = 'F'
	token[1] = 'n'
	token[2] = 's'
	token[3] = ' '

	copy(token[4:4+pbl], pb)
	token[4+pbl] = '.'
	copy(token[4+pbl+1:], sb)

	return
}

func (encoder *hmacTokenEncoder) Decode(_ context.Context, token Token) (result Authorization, err error) {
	if len(token) < 4 {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(errors.Warning("token is invalid"))
		return
	}
	after, found := bytes.CutPrefix(token, []byte{'F', 'n', 's', ' '})
	if !found {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(errors.Warning("token is invalid"))
		return
	}
	pos := bytes.IndexByte(after, '.')
	if pos < 1 {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(errors.Warning("token is invalid"))
		return
	}
	p, pErr := base64.URLEncoding.DecodeString(string(after[0:pos]))
	if pErr != nil {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(pErr)
		return
	}
	s, sErr := base64.URLEncoding.DecodeString(string(after[pos+1:]))
	if sErr != nil {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(sErr)
		return
	}
	if !encoder.signature.Verify(p, s) {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(errors.Warning("token is invalid"))
		return
	}
	decodeErr := avro.Unmarshal(p, &result)
	if decodeErr != nil {
		err = errors.Warning("authorizations: decode token failed").WithMeta("encoder", "hmac").WithCause(decodeErr)
		return
	}
	return
}
