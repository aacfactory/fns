package clusters

import (
	"bytes"
	"context"
	"fmt"
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

func NewRegistration(id []byte, name []byte, version versions.Version, internal bool, document documents.Document, client transports.Client, signature signatures.Signature) (v *Registration) {
	v = &Registration{
		id:        id,
		name:      name,
		version:   version,
		internal:  internal,
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
	internal  bool
	document  documents.Document
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
	ok = registration.internal
	return
}

func (registration *Registration) Document() (document documents.Document) {
	document = registration.document
	return
}

func (registration *Registration) Dispatch(ctx services.Request) (v interface{}, err error) {
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if !hasTransportRequestHeader {
		err = errors.Warning("fns: registration dispatch request failed").WithCause(fmt.Errorf("can not load transport request from context"))
		return
	}
	transportResponseHeader, hasTransportResponseHeader := transports.TryLoadResponseHeader(ctx)
	if !hasTransportResponseHeader {
		err = errors.Warning("fns: registration dispatch request failed").WithCause(fmt.Errorf("can not load transport response from context"))
		return
	}
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
		for _, value := range values {
			header.Add(key, value)
		}
	})
	// content-type
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), dispatchContentType)
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
	var body []byte
	argument, argumentErr := ctx.Argument().MarshalJSON()
	if argumentErr != nil {
		err = errors.Warning("fns: encode request argument failed").WithCause(argumentErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	const (
		null = "null"
	)
	if !bytes.Equal(argument, bytex.FromString(null)) {
		body = argument
	}

	// do
	status, respHeader, respBody, doErr := registration.client.Do(ctx, methodPost, path, header, body)
	if doErr != nil {
		n := registration.errs.Incr()
		if n > 10 {
			registration.closed.Store(true)
		}
		err = errors.Warning("fns: registration dispatch request failed").WithCause(doErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}
	if status == 200 {
		if registration.errs.Value() > 0 {
			registration.errs.Decr()
		}
		respHeader.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				transportResponseHeader.Add(key, value)
			}
		})
		v = respBody
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

func (registration *Registration) Handle(ctx services.Request) (v interface{}, err error) {
	if !ctx.Header().Internal() {
		v, err = registration.Dispatch(ctx)
		return
	}
	// header >>>
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	// try copy transport request header
	transportRequestHeader, hasTransportRequestHeader := transports.TryLoadRequestHeader(ctx)
	if hasTransportRequestHeader {
		transportRequestHeader.Foreach(func(key []byte, values [][]byte) {
			ok := string(key) == transports.CookieHeaderName &&
				string(key) == transports.XForwardedForHeaderName &&
				string(key) == transports.OriginHeaderName &&
				bytes.Index(key, bytex.FromString(transports.UserHeaderNamePrefix)) == 0
			if ok {
				for _, value := range values {
					header.Add(key, value)
				}
			}
		})
	}
	// content-type
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), internalContentType)
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
	status, respHeader, respBody, doErr := registration.client.Do(ctx, methodPost, path, header, body)
	if doErr != nil {
		n := registration.errs.Incr()
		if n > 10 {
			registration.closed.Store(true)
		}
		err = errors.Warning("fns: internal endpoint handle failed").WithCause(doErr).WithMeta("service", string(service)).WithMeta("fn", string(fn))
		return
	}

	// try copy transport response header
	transportResponseHeader, hasTransportResponseHeader := transports.TryLoadResponseHeader(ctx)
	if hasTransportResponseHeader {
		respHeader.Foreach(func(key []byte, values [][]byte) {
			ok := string(key) == transports.CookieHeaderName &&
				bytes.Index(key, bytex.FromString(transports.UserHeaderNamePrefix)) == 0
			if ok {
				for _, value := range values {
					transportResponseHeader.Add(key, value)
				}
			}
		})
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

func (registration *Registration) Shutdown(_ context.Context) {
	registration.closed.Store(true)
	registration.client.Close()
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
	length int
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
	named.length = len(named.values)
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
	named.length = len(named.values)
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
