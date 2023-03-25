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
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/service/internal/secret"
	"github.com/aacfactory/json"
	"strconv"
	"strings"
	"time"
)

type Token string

type CreateTokenParam struct {
	UserId  service.RequestUserId `json:"userId"`
	Options *json.Object          `json:"options"`
}

type VerifyResult struct {
	Succeed    bool
	UserId     service.RequestUserId `json:"userId"`
	Attributes *json.Object          `json:"attributes"`
}

type Tokens interface {
	service.Component
	Create(ctx context.Context, param CreateTokenParam) (token Token, err errors.CodeError)
	Verify(ctx context.Context, token Token) (result VerifyResult, err errors.CodeError)
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
	userId := param.UserId.String()
	deadline := fmt.Sprintf(fmt.Sprintf("%d", time.Now().Add(tokens.expire).Unix()))
	p := make([]byte, 4, 8)
	binary.BigEndian.PutUint16(p[0:2], uint16(len(userId)))
	binary.BigEndian.PutUint16(p[2:4], uint16(len(deadline)))
	p = append(p, bytex.FromString(userId)...)
	p = append(p, bytex.FromString(deadline)...)
	s := tokens.signer.Sign(p)
	p = append(p, s...)
	token = Token(bytex.ToString(p))
	return
}

func (tokens *defaultTokens) Verify(ctx context.Context, token Token) (result VerifyResult, err errors.CodeError) {
	if token == "" {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is required"))
		return
	}
	p := bytex.FromString(string(token))
	pLen := uint16(len(p))
	if pLen < 5 {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userIdLen := binary.BigEndian.Uint16(p[0:2])
	deadlineLen := binary.BigEndian.Uint16(p[2:4])
	if userIdLen == 0 || deadlineLen == 0 {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	ub := 4
	ue := 4 + userIdLen
	db := ue
	de := db + deadlineLen
	if pLen <= de {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	signature := p[de:]
	if !tokens.signer.Verify(p[0:de], signature) {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userIdBytes := p[ub:ue]
	if len(userIdBytes) == 0 {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userId := service.RequestUserId(bytex.ToString(userIdBytes))
	deadlineBytes := p[db:de]
	if len(deadlineBytes) == 0 {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	deadlineSec, deadlineSecErr := strconv.ParseInt(bytex.ToString(deadlineBytes), 10, 64)
	if deadlineSecErr != nil {
		err = errors.Warning("authorizations: verify token failed").WithCause(errors.Warning("token is invalid").WithCause(deadlineSecErr))
		return
	}
	deadline := time.Unix(deadlineSec, 0)
	result = VerifyResult{
		Succeed:    deadline.After(time.Now()),
		UserId:     userId,
		Attributes: nil,
	}
	return
}
