package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/service"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
	"time"
)

type Request struct {
	User     service.RequestUser `json:"user"`
	Argument json.RawMessage     `json:"argument"`
}

type Response struct {
	Succeed bool            `json:"succeed"`
	Span    *service.Span   `json:"span"`
	Body    json.RawMessage `json:"body"`
}

func newRegistrationTask(registration *Registration, handleTimeout time.Duration, hook func(task *registrationTask)) *registrationTask {
	return &registrationTask{
		registration: registration,
		r:            nil,
		result:       nil,
		timeout:      handleTimeout,
		hook:         hook,
	}
}

type registrationTask struct {
	registration *Registration
	r            service.Request
	result       service.Promise
	timeout      time.Duration
	hook         func(task *registrationTask)
}

func (task *registrationTask) begin(r service.Request, w service.Promise) {
	task.r = r
	task.result = w
}

func (task *registrationTask) end() {
	task.r = nil
	task.result = nil
}

func (task *registrationTask) executeInternal(ctx context.Context) {
	// internal is called by service function
	// so check cache first
	// when cache exists and was not out of date, then use cache
	// when cache not exist, then call with cache control disabled
	// when cache exist but was out of date, then call with if non match
	registration := task.registration
	r := task.r
	fr := task.result
	name, fn := r.Fn()
	trace, hasTracer := service.GetTracer(ctx)
	var span *service.Span
	if hasTracer {
		span = trace.StartSpan(name, fn)
		span.AddTag("kind", "remote")
	}

	ifNonMatch := ""
	var cachedBody []byte // is not internal response, cause cache was set in service, not in handler
	var cachedErr errors.CodeError
	if !r.Header().CacheControlDisabled() { // try cache control
		etag, status, _, _, deadline, body, exist := service.FetchCacheControl(ctx, name, fn, r.Argument())
		if exist {
			// cache exists
			if status != http.StatusOK {
				cachedErr = errors.Decode(body)
			} else {
				cachedBody = body
			}
			// check deadline
			if deadline.After(time.Now()) {
				// not out of date
				if span != nil {
					span.Finish()
					span.AddTag("cached", "hit")
					span.AddTag("etag", etag)
					if cachedErr == nil {
						span.AddTag("status", "OK")
						span.AddTag("handled", "succeed")
					} else {
						span.AddTag("status", cachedErr.Name())
						span.AddTag("handled", "failed")
					}
				}
				if cachedErr == nil {
					fr.Succeed(body)
				} else {
					fr.Failed(cachedErr)
				}
				return
			} else {
				// out of date
				ifNonMatch = etag
			}
		}
	}
	// make request
	// request timeout
	timeout := task.timeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	// request body
	argumentBytes, encodeArgumentErr := json.Marshal(r.Argument())
	if encodeArgumentErr != nil {
		err := errors.Warning("fns: registration request internal failed").WithCause(encodeArgumentErr)
		fr.Failed(err)
		return
	}
	ir := Request{
		User:     r.User(),
		Argument: argumentBytes,
	}
	requestBody, encodeErr := json.Marshal(&ir)
	if encodeErr != nil {
		// finish span
		err := errors.Warning("fns: registration request internal failed").WithCause(encodeErr)
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	// request path
	header := r.Header()
	path := bytex.FromString(fmt.Sprintf("/%s/%s", name, fn))
	// request
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, path)
	// request set header
	for name, vv := range header {
		for _, v := range vv {
			req.Header().Add(name, v)
		}
	}
	// internal sign header
	req.Header().Set(transports.RequestInternalHeaderName, bytex.ToString(registration.signer.Sign(requestBody)))
	// if non match header
	if ifNonMatch != "" {
		req.Header().Set(transports.CacheControlHeaderName, transports.CacheControlHeaderEnabled)
		req.Header().Set(transports.CacheControlHeaderIfNonMatch, ifNonMatch)
	}

	// request set body
	req.SetBody(requestBody)
	// do
	resp, postErr := registration.client.Do(ctx, req)
	if postErr != nil {
		task.registration.errs.Incr()
		// finish span
		err := errors.Warning("fns: registration request failed").WithCause(postErr)
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	// handle response
	if resp.Status != 200 && resp.Status != 304 && resp.Status >= 400 {
		// undefined error
		// remote endpoint was closed
		if resp.Header.Get(transports.ConnectionHeaderName) == transports.CloseHeaderValue {
			task.registration.closed.Store(false)
			if span != nil {
				span.Finish()
				span.AddTag("status", service.ErrUnavailable.Name())
				span.AddTag("handled", "failed")
			}
			fr.Failed(service.ErrUnavailable)
			return
		}
		if resp.Status == 404 {
			err := errors.NotFound("fns: not found").WithMeta("path", bytex.ToString(path))
			// finish span
			if span != nil {
				span.Finish()
				span.AddTag("status", err.Name())
				span.AddTag("handled", "failed")
			}
			fr.Failed(err)
			return
		}
		task.registration.errs.Incr()
		err := errors.Warning("fns: registration request failed").WithCause(errors.Warning(fmt.Sprintf("unknonw error, status is %d, %s", resp.Status, string(resp.Body))))
		// finish span
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}

	// check 304
	if resp.Status == http.StatusNotModified {
		// use cached
		if span != nil {
			span.Finish()
			span.AddTag("cached", "hit")
			span.AddTag("etag", ifNonMatch)
			if cachedErr == nil {
				span.AddTag("status", "OK")
				span.AddTag("handled", "succeed")
			} else {
				span.AddTag("status", cachedErr.Name())
				span.AddTag("handled", "failed")
			}
		}
		if cachedErr == nil {
			fr.Succeed(cachedBody)
		} else {
			fr.Failed(cachedErr)
		}
		return
	}
	// 200
	iresp := Response{}
	decodeErr := json.Unmarshal(resp.Body, &iresp)
	if decodeErr != nil {
		err := errors.Warning("fns: registration request failed").WithCause(errors.Warning("decode internal response failed")).WithCause(decodeErr)
		// finish span
		if span != nil {
			span.Finish()
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
		fr.Failed(err)
		return
	}
	var err errors.CodeError
	if !iresp.Succeed {
		err = errors.Decode(iresp.Body)
	}
	// finish span
	if span != nil {
		if iresp.Span != nil {
			span.AppendChild(iresp.Span)
		}
		span.Finish()
		if err == nil {
			span.AddTag("status", "OK")
			span.AddTag("handled", "succeed")
		} else {
			span.AddTag("status", err.Name())
			span.AddTag("handled", "failed")
		}
	}

	if err == nil {
		fr.Succeed(iresp.Body)
	} else {
		fr.Failed(err)
	}

	return
}

func (task *registrationTask) Execute(ctx context.Context) {
	defer task.hook(task)
	if task.r.Internal() {
		task.executeInternal(ctx)
		return
	}
	// non-internal is called by proxy
	// and its cache control was handled by middleware
	// also cache was update by remote endpoint
	// so just call
	registration := task.registration
	r := task.r
	fr := task.result

	// make request
	// request timeout
	timeout := task.timeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		timeout = deadline.Sub(time.Now())
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()
	// request body
	requestBody, encodeErr := json.Marshal(r.Argument())
	if encodeErr != nil {
		fr.Failed(errors.Warning("fns: registration request failed").WithCause(encodeErr))
		return
	}
	// request path
	header := r.Header()
	serviceName, fn := r.Fn()
	path := bytex.FromString(fmt.Sprintf("/%s/%s", serviceName, fn))
	// request
	req := transports.NewUnsafeRequest(ctx, transports.MethodPost, path)
	// request set header
	for name, vv := range header {
		for _, v := range vv {
			req.Header().Add(name, v)
		}
	}
	// request set body
	req.SetBody(requestBody)
	// do
	resp, postErr := registration.client.Do(ctx, req)
	if postErr != nil {
		task.registration.errs.Incr()
		fr.Failed(errors.Warning("fns: registration request failed").WithCause(postErr))
		return
	}
	// handle response
	// remote endpoint was closed
	if resp.Header.Get(transports.ConnectionHeaderName) == transports.CloseHeaderValue {
		task.registration.closed.Store(false)
		fr.Failed(service.ErrUnavailable)
		return
	}
	// failed response
	if resp.Status != http.StatusOK {
		var body errors.CodeError
		if resp.Body == nil || len(resp.Body) == 0 {
			body = errors.Warning("nil error")
		} else {
			body = errors.Decode(resp.Body)
		}
		task.registration.errs.Incr()
		fr.Failed(body)
		return
	}
	// succeed response
	if resp.Body == nil || len(resp.Body) == 0 {
		fr.Succeed(nil)
	} else {
		fr.Succeed(json.RawMessage(resp.Body))
	}
	return
}
