package development

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
)

type Fn struct {
	endpointName string
	name         string
	internal     bool
	readonly     bool
	path         []byte
	signature    signatures.Signature
	client       transports.Client
}

func (fn *Fn) Name() string {
	return fn.name
}

func (fn *Fn) Internal() bool {
	return fn.internal
}

func (fn *Fn) Readonly() bool {
	return fn.readonly
}

func (fn *Fn) Handle(ctx services.Request) (v interface{}, err error) {
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	// try copy transport request header
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if hasTransportRequestHeader {
		transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				header.Add(key, value)
			}
		})
	}
	// content-type
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// endpoint id
	endpointId := ctx.Header().EndpointId()
	if len(endpointId) > 0 {
		header.Set(bytex.FromString(transports.EndpointIdHeaderName), endpointId)
	}
	// device id
	deviceId := ctx.Header().DeviceId()
	if len(deviceId) > 0 {
		header.Set(bytex.FromString(transports.DeviceIdHeaderName), deviceId)
	}
	// device ip
	deviceIp := ctx.Header().DeviceIp()
	if len(deviceIp) > 0 {
		header.Set(bytex.FromString(transports.DeviceIpHeaderName), deviceIp)
	}
	// request id
	requestId := ctx.Header().RequestId()
	if len(requestId) > 0 {
		header.Set(bytex.FromString(transports.RequestIdHeaderName), requestId)
	}
	// request version
	requestVersion := ctx.Header().AcceptedVersions()
	if len(requestVersion) > 0 {
		header.Set(bytex.FromString(transports.RequestVersionsHeaderName), requestVersion.Bytes())
	}
	// authorization
	token := ctx.Header().Token()
	if len(token) > 0 {
		header.Set(bytex.FromString(transports.AuthorizationHeaderName), token)
	}
	// header <<<

	// body
	userValues := make([]Entry, 0, 1)
	ctx.UserValues(func(key []byte, val any) {
		p, encodeErr := json.Marshal(val)
		if encodeErr != nil {
			return
		}
		userValues = append(userValues, Entry{
			Key: key,
			Val: p,
		})
	})
	argument, argumentErr := ctx.Param().MarshalJSON()
	if argumentErr != nil {
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("service", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	rb := RequestBody{
		UserValues: userValues,
		Argument:   argument,
	}
	body, bodyErr := json.Marshal(rb)
	if bodyErr != nil {
		err = errors.Warning("fns: encode body failed").WithCause(bodyErr).WithMeta("service", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	// sign
	signature := fn.signature.Sign(body)
	header.Set(bytex.FromString(transports.SignatureHeaderName), signature)
	// do
	status, _, respBody, doErr := fn.client.Do(ctx, transports.MethodPost, fn.path, header, body)
	if doErr != nil {
		err = errors.Warning("fns: development endpoint handle failed").WithCause(doErr).WithMeta("service", fn.endpointName).WithMeta("fn", fn.name)
		return
	}

	if status == 200 {
		rsb := ResponseBody{}
		decodeErr := json.Unmarshal(respBody, &rsb)
		if decodeErr != nil {
			err = errors.Warning("fns: development endpoint handle failed").WithCause(decodeErr).WithMeta("service", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		if rsb.Span != nil && len(rsb.Span.Id) > 0 {
			trace, hasTrace := tracings.Load(ctx)
			if hasTrace {
				trace.Mount(rsb.Span)
			}
		}
		if rsb.Succeed {
			v = rsb.Data
		} else {
			err = errors.Decode(rsb.Data)
		}
		return
	}
	switch status {
	case http.StatusServiceUnavailable:
		err = ErrUnavailable
		break
	case http.StatusTooManyRequests:
		err = ErrTooMayRequest
		break
	case http.StatusTooEarly:
		err = ErrTooEarly
		break
	}
	return
}
