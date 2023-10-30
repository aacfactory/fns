package clusters

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/services/tracing"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
	"sync/atomic"
)

// Registration
// implement endpoint
// 直接调client，不再workers
// 如果是internal 则签名，如果不是，可能是proxy的
type Registration struct {
	id        []byte
	address   []byte
	name      string
	version   versions.Version
	document  *documents.Document
	client    transports.Client
	signature signatures.Signature
	closed    *atomic.Bool
	errs      *window.Times
}

func (registration *Registration) Name() (name string) {
	name = registration.name
	return
}

func (registration *Registration) Internal() (ok bool) {
	ok = true
	return
}

func (registration *Registration) Document() (document *documents.Document) {
	document = registration.document
	return
}

func (registration *Registration) Handle(ctx services.Request) (v interface{}, err error) {
	// header >>>
	header := transports.NewHeader()
	// content-type
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// internal
	header.Set(bytex.FromString(transports.RequestInternalHeaderName), []byte{'1'})
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
	authorization := ctx.Header().Authorization()
	if len(authorization) > 0 {
		header.Set(bytex.FromString(transports.AuthorizationHeaderName), authorization)
	}
	// header <<<

	// path
	service, fn := ctx.Fn()
	sln := len(service)
	fln := len(fn)
	path := make([]byte, sln+fln+2)
	path[0] = '/'
	path[sln+1] = '/'
	copy(path[1:], service)
	copy(path[sln+2:], fn)

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
	argument, argumentErr := ctx.Argument().MarshalJSON()
	if argumentErr != nil {
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	rb := RequestBody{
		UserValues: userValues,
		Argument:   argument,
	}
	body, bodyErr := json.Marshal(rb)
	if bodyErr != nil {
		err = errors.Warning("fns: encode body failed").WithCause(bodyErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	// sign
	signature := registration.signature.Sign(body)
	header.Set(bytex.FromString(transports.SignatureHeaderName), signature)

	// do
	status, _, respBody, doErr := registration.client.Do(ctx, methodPost, path, header, body)
	if doErr != nil {
		n := registration.errs.Incr()
		if n > 10 {
			registration.closed.Store(true)
		}
		err = errors.Warning("fns: internal endpoint handle failed").WithCause(doErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	if status == 200 {
		if registration.errs.Value() > 0 {
			registration.errs.Decr()
		}
		rsb := ResponseBody{}
		decodeErr := json.Unmarshal(respBody, &rsb)
		if decodeErr != nil {
			err = errors.Warning("fns: internal endpoint handle failed").WithCause(decodeErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
			return
		}
		if rsb.Span != nil {
			tracing.MountSpan(ctx, rsb.Span)
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
		registration.closed.Store(true)
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

func (registration *Registration) Enabled() bool {
	return registration.closed.Load()
}

func (registration *Registration) Close() {
	registration.closed.Store(true)
}

type SortedRegistrations []*Registration

func (list SortedRegistrations) Len() int {
	return len(list)
}

func (list SortedRegistrations) Less(i, j int) bool {
	return list[i].version.LessThan(list[j].version)
}

func (list SortedRegistrations) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
	return
}

func (list SortedRegistrations) Get(version versions.Version) (r *Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	for _, registration := range list {
		if version.Equals(registration.version) {
			r = registration
			break
		}
	}
	return
}

func (list SortedRegistrations) Range(interval versions.Interval) (r []*Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = make([]*Registration, 0, 1)
	for _, registration := range list {
		if interval.Accept(registration.version) {
			r = append(r, registration)
		}
	}
	return
}

func (list SortedRegistrations) MaxVersion() (r []*Registration) {
	size := len(list)
	if size == 0 {
		return
	}
	r = make([]*Registration, 0, 1)
	maxed := list[size-1]
	r = append(r, maxed)
	for i := size - 2; i < 0; i-- {
		prev := list[i]
		if prev.version.Equals(maxed.version) {
			r = append(r, prev)
			continue
		}
		break
	}
	return
}
