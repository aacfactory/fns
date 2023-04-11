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

package certificates

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service"
	"time"
)

const (
	name              = "certificates"
	rootFn            = "root"
	saveRootFn        = "save_root"
	issueFn           = "issue"
	revokeFn          = "revoke"
	verifyFn          = "verify"
	saveExchangeKeyFn = "save_exchange_key"
	getExchangeKeyFn  = "get_exchage_key"
)

type RootParam struct {
	Kind string `json:"kind"`
}

type RootResult struct {
	Has            bool          `json:"has"`
	Kind           string        `json:"kind"`
	AppId          string        `json:"appId"`
	PublicPEM      []byte        `json:"publicPem"`
	PrivatePEM     []byte        `json:"privatePem"`
	Password       []byte        `json:"password"`
	ExchangeKeyTTL time.Duration `json:"exchangeKeyTtl"`
}

func Root(ctx context.Context, param RootParam) (result RootResult, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: get root failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, rootFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("certificates: get root failed").WithCause(scanErr)
		return
	}
	return
}

type SaveRootParam struct {
	Kind           string        `json:"kind"`
	AppId          string        `json:"appId"`
	PublicPEM      []byte        `json:"publicPem"`
	PrivatePEM     []byte        `json:"privatePem"`
	Password       []byte        `json:"password"`
	ExchangeKeyTTL time.Duration `json:"exchangeKeyTtl"`
}

func SaveRoot(ctx context.Context, param SaveRootParam) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: save root failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	_, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, saveRootFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	return
}

type IssueParam struct {
	AppId      string `json:"appId"`
	AppName    string `json:"appName"`
	ExpireDays int    `json:"expireDays"`
}

type IssueResult struct {
	PublicPEM  []byte `json:"publicPem"`
	PrivatePEM []byte `json:"privatePem"`
}

func Issue(ctx context.Context, param IssueParam) (result IssueResult, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: issue failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, issueFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("certificates: issue failed").WithCause(scanErr)
		return
	}
	return
}

type RevokeParam struct {
	AppId string `json:"appId"`
}

func Revoke(ctx context.Context, param RevokeParam) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: revoke failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	_, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, revokeFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	return
}

type VerifyParam struct {
	PublicPEM []byte `json:"publicPem"`
}

type VerifyResult struct {
	Ok bool `json:"ok"`
}

func Verify(ctx context.Context, param VerifyParam) (result VerifyResult, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: verify failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, verifyFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("certificates: verify failed").WithCause(scanErr)
		return
	}
	return
}

type SaveExchangeKeyParam struct {
	AppId       string        `json:"appId"`
	ExchangeKey []byte        `json:"exchangeKey"`
	TTL         time.Duration `json:"ttl"`
}

func SaveExchangeKey(ctx context.Context, param SaveExchangeKeyParam) (err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: save exchange key failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	_, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, saveExchangeKeyFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	return
}

type GetExchangeKeyParam struct {
	AppId string `json:"appId"`
}

type GetExchangeKeyResult struct {
	Has         bool   `json:"has"`
	ExchangeKey []byte `json:"exchangeKey"`
}

func GetExchangeKey(ctx context.Context, param GetExchangeKeyParam) (result GetExchangeKeyResult, err errors.CodeError) {
	endpoint, hasEndpoint := service.GetEndpoint(ctx, name)
	if !hasEndpoint {
		err = errors.Warning("certificates: get exchange key failed").WithCause(errors.Warning("certificates: service was not deployed"))
		return
	}
	future, requestErr := endpoint.RequestSync(ctx, service.NewRequest(ctx, name, getExchangeKeyFn, service.NewArgument(param), service.WithInternalRequest()))
	if requestErr != nil {
		err = requestErr
		return
	}
	scanErr := future.Scan(&result)
	if scanErr != nil {
		err = errors.Warning("certificates: get exchange key failed").WithCause(scanErr)
		return
	}
	return
}
