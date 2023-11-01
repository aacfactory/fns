package authorizations

import (
	sc "context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"time"
)

const (
	contextKey     = "@fns:context:authorizations"
	contextUserKey = "authorizations"
)

func With(ctx sc.Context, authorization Authorization) sc.Context {
	r, ok := ctx.(services.Request)
	if ok {
		r.SetUserValue(bytex.FromString(contextUserKey), authorization)
		return r
	}
	return sc.WithValue(ctx, bytex.FromString(contextKey), authorization)
}

func Load(ctx sc.Context) Authorization {
	fc, ok := ctx.(context.Context)
	if ok {
		v := fc.UserValue(bytex.FromString(contextUserKey))
		if v == nil {
			return Authorization{}
		}
		return v.(Authorization)
	}
	v := ctx.Value(contextKey)
	if v == nil {
		return Authorization{}
	}
	return v.(Authorization)
}

type Authorization struct {
	Id         Id         `json:"id"`
	Account    Id         `json:"account"`
	Attributes Attributes `json:"attributes"`
	ExpireAT   time.Time  `json:"expireAT"`
}

func (authorization Authorization) Exist() bool {
	return authorization.Id.Exist()
}

func (authorization Authorization) Validate() bool {
	return authorization.Exist() && authorization.ExpireAT.After(time.Now())
}

const (
	EndpointName = "authorizations"
	EncodeFnName = "encode"
	DecodeFnName = "decode"
)

func Encode(ctx sc.Context, authorization Authorization) (token Token, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(EndpointName), bytex.FromString(EncodeFnName),
		services.NewArgument(authorization),
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	token = make([]byte, 0, 1)
	scanErr := response.Scan(&token)
	if scanErr != nil {
		err = errors.Warning("fns: scan encode value failed").WithCause(scanErr)
		return
	}
	return
}

func Decode(ctx sc.Context, token Token) (authorization Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(EndpointName), bytex.FromString(DecodeFnName),
		services.NewArgument(token),
	)
	if handleErr != nil {
		err = handleErr
		return
	}
	scanErr := response.Scan(&authorization)
	if scanErr != nil {
		err = errors.Warning("fns: scan decode value failed").WithCause(scanErr)
		return
	}
	return
}

var ErrUnauthorized = errors.Unauthorized("fns: unauthorized")

func Validate(ctx sc.Context) (err error) {
	authorization := Load(ctx)
	if authorization.Exist() {
		if authorization.Validate() {
			return
		}
		err = ErrUnauthorized
		return
	}
	r := services.LoadRequest(ctx)
	token := r.Header().Token()
	if len(token) == 0 {
		err = ErrUnauthorized
		return
	}
	authorization, err = Decode(ctx, token)
	if err != nil {
		err = ErrUnauthorized.WithCause(err).WithMeta("token", string(token))
		return
	}
	if authorization.Validate() {
		With(ctx, authorization)
		return
	}
	err = ErrUnauthorized
	return
}
