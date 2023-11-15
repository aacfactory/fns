package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
)

type encodeFn struct {
	encoder TokenEncoder
}

func (fn *encodeFn) Name() string {
	return encodeFnName
}

func (fn *encodeFn) Internal() bool {
	return true
}

func (fn *encodeFn) Readonly() bool {
	return false
}

func (fn *encodeFn) Handle(r services.Request) (v interface{}, err error) {
	param := Authorization{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.BadRequest("authorizations: invalid param")
		return
	}
	token, encodeErr := fn.encoder.Encode(r, param)
	if encodeErr != nil {
		err = errors.ServiceError("authorizations: encode authorization failed").WithCause(encodeErr)
		return
	}
	v = token
	return
}

type decodeFn struct {
	encoder TokenEncoder
}

func (fn *decodeFn) Name() string {
	return decodeFnName
}

func (fn *decodeFn) Internal() bool {
	return true
}

func (fn *decodeFn) Readonly() bool {
	return false
}

func (fn *decodeFn) Handle(r services.Request) (v interface{}, err error) {
	param := Token{}
	paramErr := r.Param().Scan(&param)
	if paramErr != nil {
		err = errors.BadRequest("authorizations: invalid param")
		return
	}
	authorization, decodeErr := fn.encoder.Decode(r, param)
	if decodeErr != nil {
		err = errors.ServiceError("authorizations: decode token failed").WithCause(decodeErr)
		return
	}
	v = authorization
	return
}

func ServiceWithEncoder(encoder TokenEncoder) services.Service {
	return &service{
		Abstract: services.NewAbstract(endpointName, true, encoder),
	}
}

func Service() services.Service {
	return ServiceWithEncoder(DefaultTokenEncoder())
}

type service struct {
	services.Abstract
}

func (svc *service) Construct(options services.Options) (err error) {
	err = svc.Abstract.Construct(options)
	if err != nil {
		return
	}
	var encoder TokenEncoder
	has := false
	components := svc.Abstract.Components()
	for _, component := range components {
		encoder, has = component.(TokenEncoder)
		if has {
			break
		}
	}
	if encoder == nil {
		err = errors.Warning("authorizations: service need token encoder component")
		return
	}
	svc.AddFunction(&encodeFn{
		encoder: encoder,
	})
	svc.AddFunction(&decodeFn{
		encoder: encoder,
	})
	return
}

func (svc *service) Document() (v documents.Document) {
	v = documents.New(endpointName, "Authorizations service", true, svc.Abstract.Version())
	v.AddFn(documents.NewFn(encodeFnName).SetParam(documents.Any()).SetResult(documents.Any()))
	v.AddFn(documents.NewFn(decodeFnName).SetParam(documents.Any()).SetResult(documents.Any()))
	return
}
