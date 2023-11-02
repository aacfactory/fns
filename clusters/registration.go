package clusters

import (
	"bytes"
	"context"
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
	"sort"
	"sync/atomic"
	"time"
)

func NewRegistration(id []byte, name []byte, version versions.Version, document *documents.Document, client transports.Client, signature signatures.Signature) (v *Registration) {
	v = &Registration{
		id:        id,
		name:      name,
		version:   version,
		document:  document,
		client:    client,
		signature: signature,
		closed:    new(atomic.Bool),
		errs:      window.NewTimes(10 * time.Second),
	}
	return
}

type Registration struct {
	id        []byte
	name      []byte
	version   versions.Version
	document  *documents.Document
	client    transports.Client
	signature signatures.Signature
	closed    *atomic.Bool
	errs      *window.Times
}

func (registration *Registration) Name() (name string) {
	name = bytex.ToString(registration.name)
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

func (registration *Registration) Dispatch(ctx context.Context, method []byte, path []byte, header transports.Header, body []byte) (status int, responseHeader transports.Header, responseBody []byte, err error) {
	status, responseHeader, responseBody, err = registration.client.Do(ctx, method, path, header, body)
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
	token := ctx.Header().Token()
	if len(token) > 0 {
		header.Set(bytex.FromString(transports.AuthorizationHeaderName), token)
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
		if rsb.Span != nil && len(rsb.Span.Id) > 0 {
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

type NamedRegistration struct {
	name   []byte
	length uint64
	pos    uint64
	values SortedRegistrations
}

func (named *NamedRegistration) Add(registration *Registration) {
	_, exist := named.Get(registration.id)
	if exist {
		return
	}
	named.values = append(named.values, registration)
	sort.Sort(named.values)
	named.length = uint64(len(named.values))
}

func (named *NamedRegistration) Remove(id []byte) {
	n := -1
	for i, value := range named.values {
		if bytes.Equal(value.id, id) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	named.values = append(named.values[:n], named.values[n+1:]...)
	named.length = uint64(len(named.values))
}

func (named *NamedRegistration) Get(id []byte) (r *Registration, has bool) {
	if named.length == 0 {
		return
	}
	for _, registration := range named.values {
		if bytes.Equal(registration.id, id) {
			r = registration
			has = true
			break
		}
	}
	return
}

func (named *NamedRegistration) Range(interval versions.Interval) (v *Registration, has bool) {
	if named.length == 0 {
		return
	}
	targets := make([]*Registration, 0, 1)
	for _, registration := range named.values {
		if interval.Accept(registration.version) {
			targets = append(targets, registration)
		}
	}
	n := uint64(len(targets))
	pos := int(atomic.AddUint64(&named.pos, 1) % n)
	v = targets[pos]
	has = true
	return
}

func (named *NamedRegistration) MaxOne() (v *Registration, has bool) {
	if named.length == 0 {
		return
	}
	targets := make([]*Registration, 0, 1)
	maxed := named.values[named.length-1]
	targets = append(targets, maxed)
	for i := named.length - 2; i > -1; i-- {
		prev := named.values[i]
		if prev.version.Equals(maxed.version) {
			targets = append(targets, prev)
			continue
		}
		break
	}
	n := uint64(len(targets))
	pos := int(atomic.AddUint64(&named.pos, 1) % n)
	v = targets[pos]
	has = true
	return
}

type NamedRegistrations []NamedRegistration

func (names NamedRegistrations) Get(name []byte) (v NamedRegistration, has bool) {
	for _, named := range names {
		if named.length > 0 && bytes.Equal(named.name, name) {
			v = named
			has = true
			break
		}
	}
	return
}

func (names NamedRegistrations) Add(registration *Registration) NamedRegistrations {
	name := registration.name
	for i, named := range names {
		if named.length > 0 && bytes.Equal(named.name, name) {
			named.Add(registration)
			names[i] = named
			return names
		}
	}
	named := NamedRegistration{}
	named.Add(registration)
	return append(names, named)
}

func (names NamedRegistrations) Remove(name []byte, id []byte) NamedRegistrations {
	for i, named := range names {
		if named.length > 0 && bytes.Equal(named.name, name) {
			named.Remove(id)
			if named.length == 0 {
				return append(names[:i], names[i+1:]...)
			}
			names[i] = named
			return names
		}
	}
	return names
}
