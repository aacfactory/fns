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

package authorizations

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/uid"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"github.com/cespare/xxhash/v2"
	"strconv"
	"strings"
	"time"
)

type Token string

type CreateTokenParam struct {
	UserId     service.RequestUserId `json:"userId"`
	Attributes *json.Object          `json:"attributes"`
}

type ParsedToken struct {
	Valid      bool                  `json:"valid"`
	Id         string                `json:"id"`
	UserId     service.RequestUserId `json:"userId"`
	Attributes *json.Object          `json:"attributes"`
}

type Tokens interface {
	service.Component
	Create(ctx context.Context, param CreateTokenParam) (token Token, err errors.CodeError)
	Parse(ctx context.Context, token Token) (result ParsedToken, err errors.CodeError)
}

type defaultTokensConfig struct {
	Key    string
	Expire string
}

func DefaultTokens() Tokens {
	return &defaultTokens{}
}

type defaultTokens struct {
	signer *secret.Signer
	expire time.Duration
}

func (tokens *defaultTokens) Name() (name string) {
	return "default"
}

func (tokens *defaultTokens) Build(options service.ComponentOptions) (err error) {
	config := defaultTokensConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("authorizations: build default tokens failed").WithCause(configErr)
		return
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		err = errors.Warning("authorizations: build default tokens failed").WithCause(errors.Warning("key is require"))
		return
	}
	tokens.signer = secret.NewSigner([]byte(key))
	expire := 13 * 24 * time.Hour
	if config.Expire != "" {
		expire, err = time.ParseDuration(strings.TrimSpace(config.Expire))
		if err != nil {
			err = errors.Warning("authorizations: build default tokens failed").WithCause(errors.Warning("expire must be time.Duration format").WithCause(err))
			return
		}
	}
	tokens.expire = expire
	return
}

func (tokens *defaultTokens) Close() {
	return
}

func (tokens *defaultTokens) Create(ctx context.Context, param CreateTokenParam) (token Token, err errors.CodeError) {
	if !param.UserId.Exist() {
		err = errors.Warning("authorizations: create token failed").WithCause(errors.Warning("user id is not exist"))
		return
	}
	id := xxhash.Sum64(bytex.FromString(uid.UID()))
	userId := param.UserId.String()
	deadline := fmt.Sprintf(fmt.Sprintf("%d", time.Now().Add(tokens.expire).Unix()))
	payload := ""
	if param.Attributes != nil {
		payload = bytex.ToString(param.Attributes.Raw())
	}
	p := make([]byte, 16, 64)
	binary.BigEndian.PutUint64(p[0:8], id)
	binary.BigEndian.PutUint16(p[8:10], uint16(len(userId)))
	binary.BigEndian.PutUint16(p[10:12], uint16(len(deadline)))
	binary.BigEndian.PutUint32(p[12:16], uint32(len(payload)))
	p = append(p, bytex.FromString(userId)...)
	p = append(p, bytex.FromString(deadline)...)
	p = append(p, bytex.FromString(payload)...)
	s := tokens.signer.Sign(p)
	p = append(p, s...)
	token = Token(base64.URLEncoding.EncodeToString(p))
	return
}

func (tokens *defaultTokens) Parse(ctx context.Context, token Token) (result ParsedToken, err errors.CodeError) {
	if token == "" {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is required"))
		return
	}
	p, decodeErr := base64.URLEncoding.DecodeString(string(token))
	if decodeErr != nil {
		err = errors.Warning("authorizations: parse token failed").WithCause(decodeErr)
		return
	}
	pLen := uint32(len(p))
	if pLen < 16 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	id := binary.BigEndian.Uint64(p[0:8])
	if id == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userIdLen := uint32(binary.BigEndian.Uint16(p[8:10]))
	deadlineLen := uint32(binary.BigEndian.Uint16(p[10:12]))
	payloadLen := binary.BigEndian.Uint32(p[12:16])
	if userIdLen == 0 || deadlineLen == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	ub := 16
	ue := 16 + userIdLen
	db := ue
	de := db + deadlineLen
	pb := de
	pe := pb + payloadLen
	if pLen <= pe {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	signature := p[pe:]
	if !tokens.signer.Verify(p[0:pe], signature) {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userIdBytes := p[ub:ue]
	if len(userIdBytes) == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userId := service.RequestUserId(bytex.ToString(userIdBytes))
	deadlineBytes := p[db:de]
	if len(deadlineBytes) == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	deadlineSec, deadlineSecErr := strconv.ParseInt(bytex.ToString(deadlineBytes), 10, 64)
	if deadlineSecErr != nil {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid").WithCause(deadlineSecErr))
		return
	}
	deadline := time.Unix(deadlineSec, 0)
	attributes := json.NewObject()
	if payloadLen > 0 {
		payload := p[pb:pe]
		if !json.Validate(payload) {
			err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
			return
		}
		attributesErr := attributes.UnmarshalJSON(payload)
		if attributesErr != nil {
			err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid").WithCause(attributesErr))
			return
		}
	}
	result = ParsedToken{
		Valid:      deadline.After(time.Now()),
		Id:         fmt.Sprintf("%d", id),
		UserId:     userId,
		Attributes: attributes,
	}
	return
}
