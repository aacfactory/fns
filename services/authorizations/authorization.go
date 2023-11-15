package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"time"
)

var (
	contextUserKey = []byte("authorizations")
)

func With(ctx context.Context, authorization Authorization) {
	ctx.SetUserValue(contextUserKey, authorization)
}

func Load(ctx context.Context) Authorization {
	authorization := Authorization{}
	_, _ = ctx.ScanUserValue(contextUserKey, &authorization)
	return authorization
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

func Encode(ctx context.Context, authorization Authorization) (token Token, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(EndpointName), bytex.FromString(EncodeFnName),
		authorization,
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

func Decode(ctx context.Context, token Token) (authorization Authorization, err error) {
	rt := runtime.Load(ctx)
	response, handleErr := rt.Endpoints().Request(
		ctx,
		bytex.FromString(EndpointName), bytex.FromString(DecodeFnName),
		token,
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

func Validate(ctx context.Context) (err error) {
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
