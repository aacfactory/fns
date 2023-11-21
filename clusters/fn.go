package clusters

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/tracings"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
	"sync/atomic"
)

type Fn struct {
	endpointName string
	name         string
	internal     bool
	readonly     bool
	path         []byte
	signature    signatures.Signature
	errs         *window.Times
	health       atomic.Bool
	client       transports.Client
}

func (fn *Fn) Enable() bool {
	return fn.health.Load()
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
	if !ctx.Header().Internal() {
		err = errors.Warning("fns: request must be internal")
		return
	}
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	// try copy transport request header
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if hasTransportRequestHeader {
		transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
			ok := bytes.Equal(key, transports.CookieHeaderName) ||
				bytes.Equal(key, transports.XForwardedForHeaderName) ||
				bytes.Equal(key, transports.OriginHeaderName) ||
				bytes.Index(key, transports.UserHeaderNamePrefix) == 0
			if ok {
				for _, value := range values {
					header.Add(key, value)
				}
			}
		})
	}
	// content-type
	header.Set(transports.ContentTypeHeaderName, internalContentType)
	// endpoint id
	endpointId := ctx.Header().EndpointId()
	if len(endpointId) > 0 {
		header.Set(transports.EndpointIdHeaderName, endpointId)
	}
	// device id
	deviceId := ctx.Header().DeviceId()
	if len(deviceId) > 0 {
		header.Set(transports.DeviceIdHeaderName, deviceId)
	}
	// device ip
	deviceIp := ctx.Header().DeviceIp()
	if len(deviceIp) > 0 {
		header.Set(transports.DeviceIpHeaderName, deviceIp)
	}
	// request id
	requestId := ctx.Header().RequestId()
	if len(requestId) > 0 {
		header.Set(transports.RequestIdHeaderName, requestId)
	}
	// request version
	requestVersion := ctx.Header().AcceptedVersions()
	if len(requestVersion) > 0 {
		header.Set(transports.RequestVersionsHeaderName, requestVersion.Bytes())
	}
	// authorization
	token := ctx.Header().Token()
	if len(token) > 0 {
		header.Set(transports.AuthorizationHeaderName, token)
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
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	rb := RequestBody{
		ContextUserValues: userValues,
		Params:            argument,
	}
	body, bodyErr := json.Marshal(rb)
	if bodyErr != nil {
		err = errors.Warning("fns: encode body failed").WithCause(bodyErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}
	// sign
	signature := fn.signature.Sign(body)
	header.Set(transports.SignatureHeaderName, signature)

	// do
	status, respHeader, respBody, doErr := fn.client.Do(ctx, transports.MethodPost, fn.path, header, body)
	if doErr != nil {
		n := fn.errs.Incr()
		if n > 10 {
			fn.health.Store(false)
		}
		err = errors.Warning("fns: internal endpoint handle failed").WithCause(doErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
		return
	}

	// try copy transport response header
	transportResponseHeader, hasTransportResponseHeader := transports.TryLoadResponseHeader(ctx)
	if hasTransportResponseHeader {
		respHeader.Foreach(func(key []byte, values [][]byte) {
			ok := bytes.Equal(key, transports.CookieHeaderName) ||
				bytes.Index(key, transports.UserHeaderNamePrefix) == 0
			if ok {
				for _, value := range values {
					transportResponseHeader.Add(key, value)
				}
			}
		})
	}

	if status == 200 {
		if fn.errs.Value() > 0 {
			fn.errs.Decr()
		}
		rsb := ResponseBody{}
		decodeErr := json.Unmarshal(respBody, &rsb)
		if decodeErr != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(decodeErr).WithMeta("endpoint", fn.endpointName).WithMeta("fn", fn.name)
			return
		}
		trace, hasTrace := tracings.Load(ctx)
		if hasTrace {
			spanAttachment, hasSpanAttachment := rsb.Attachments.Get("span")
			if hasSpanAttachment {
				span := tracings.Span{}
				spanErr := spanAttachment.Scan(&span)
				if spanErr == nil {
					trace.Mount(&span)
				}
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
		fn.health.Store(false)
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
