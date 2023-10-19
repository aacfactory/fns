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
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/json"
	"math"
	"strconv"
	"strings"
	"time"
)

type Token string

func (token Token) String() string {
	return string(token)
}

type FormatTokenParam struct {
	Id          string                         `json:"id"`
	UserId      services.RequestUserId         `json:"userId"`
	Attributes  services.RequestUserAttributes `json:"attributes"`
	Expirations time.Duration                  `json:"expirations"`
}

type ParsedToken struct {
	Valid      bool                           `json:"valid"`
	Id         string                         `json:"id"`
	UserId     services.RequestUserId         `json:"userId"`
	Attributes services.RequestUserAttributes `json:"attributes"`
	ExpireAT   time.Time                      `json:"expireAt"`
}

type Tokens interface {
	services.Component
	Format(ctx context.Context, param FormatTokenParam) (token Token, err errors.CodeError)
	Parse(ctx context.Context, token Token) (result ParsedToken, err errors.CodeError)
}

type defaultTokensConfig struct {
	Key string
}

func DefaultTokens() Tokens {
	return &defaultTokens{}
}

type defaultTokens struct {
	signer signatures.Signature
}

func (tokens *defaultTokens) Name() (name string) {
	return "default"
}

func (tokens *defaultTokens) Build(options services.ComponentOptions) (err error) {
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
	tokens.signer = signatures.HMAC([]byte(key))
	return
}

func (tokens *defaultTokens) Close() {
	return
}

func (tokens *defaultTokens) Format(ctx context.Context, param FormatTokenParam) (token Token, err errors.CodeError) {
	if param.Id == "" {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("id is required"))
		return
	}
	if !param.UserId.Exist() {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("user id is required"))
		return
	}
	expirations := param.Expirations
	if expirations < 1 {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("expirations is required"))
		return
	}
	id := param.Id
	if len(id) > 64 {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("id is too large"))
		return
	}
	userId := param.UserId.String()
	if len(userId) > 64 {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("use id is too large"))
		return
	}
	deadline := fmt.Sprintf(fmt.Sprintf("%d", time.Now().Add(expirations).Unix()))
	payload := ""
	if param.Attributes != nil {
		payload = bytex.ToString(param.Attributes)
	}
	if len(payload) > math.MaxInt64 {
		err = errors.Warning("authorizations: format token failed").WithCause(errors.Warning("payload is too large"))
		return
	}
	p := make([]byte, 16, 64)
	binary.BigEndian.PutUint32(p[0:4], uint32(len(id)))
	binary.BigEndian.PutUint32(p[4:8], uint32(len(userId)))
	binary.BigEndian.PutUint32(p[8:12], uint32(len(deadline)))
	binary.BigEndian.PutUint32(p[12:16], uint32(len(payload)))
	p = append(p, bytex.FromString(id)...)
	p = append(p, bytex.FromString(userId)...)
	p = append(p, bytex.FromString(deadline)...)
	p = append(p, bytex.FromString(payload)...)
	s := tokens.signer.Sign(p)
	p = append(p, s...)
	token = Token(fmt.Sprintf("Fns %s", base64.URLEncoding.EncodeToString(p)))
	return
}

func (tokens *defaultTokens) Parse(ctx context.Context, token Token) (result ParsedToken, err errors.CodeError) {
	if token == "" {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is required"))
		return
	}
	remains, cut := strings.CutPrefix(string(token), "Fns ")
	if !cut {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is required"))
		return
	}
	p, decodeErr := base64.URLEncoding.DecodeString(remains)
	if decodeErr != nil {
		err = errors.Warning("authorizations: parse token failed").WithCause(decodeErr)
		return
	}
	pLen := uint32(len(p))
	if pLen < 16 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	idLen := binary.BigEndian.Uint32(p[0:4])
	userIdLen := binary.BigEndian.Uint32(p[4:8])
	expireAtLen := binary.BigEndian.Uint32(p[8:12])
	payloadLen := binary.BigEndian.Uint32(p[12:16])
	if idLen == 0 || userIdLen == 0 || expireAtLen == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	ib := uint32(16)
	ie := ib + idLen
	ub := ie
	ue := ub + userIdLen
	eb := ue
	ee := eb + expireAtLen
	pb := ee
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
	idBytes := p[ib:ie]
	userIdBytes := p[ub:ue]
	if len(userIdBytes) == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	userId := services.RequestUserId(bytex.ToString(userIdBytes))
	expireAtBytes := p[eb:ee]
	if len(expireAtBytes) == 0 {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid"))
		return
	}
	expireAtSec, expireAtSecErr := strconv.ParseInt(bytex.ToString(expireAtBytes), 10, 64)
	if expireAtSecErr != nil {
		err = errors.Warning("authorizations: parse token failed").WithCause(errors.Warning("token is invalid").WithCause(expireAtSecErr))
		return
	}
	expireAt := time.Unix(expireAtSec, 0)
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
		Valid:      expireAt.After(time.Now()),
		Id:         bytex.ToString(idBytes),
		UserId:     userId,
		Attributes: attributes,
		ExpireAT:   expireAt,
	}
	return
}
