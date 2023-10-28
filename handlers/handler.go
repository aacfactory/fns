package handlers

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/oas"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"golang.org/x/sync/singleflight"
	"net/http"
	"sync"
)

var (
	methodGet  = bytex.FromString(http.MethodGet)
	methodPost = bytex.FromString(http.MethodPost)
)

var (
	slashBytes = []byte{'/'}
)

type Handler struct {
	rt      *runtime.Runtime
	doc     documents.Documents
	openapi oas.API
	once    sync.Once
	group   singleflight.Group
}

func (h *Handler) Handle(w transports.ResponseWriter, r transports.Request) {
	if bytex.Equal(r.Method(), methodGet) {
		if bytex.Equal(r.Path(), bytex.FromString(documents.ServicesDocumentsPath)) {
			h.handleDocuments(w, r)
		} else if bytex.Equal(r.Path(), bytex.FromString(documents.ServicesOpenapiPath)) {
			h.handleOpenapi(w, r)
		} else {
			w.Failed(ErrNotFound)
		}
		return
	}
	if bytex.Equal(r.Method(), methodPost) {

		internalHeader := r.Header().Get(bytex.FromString(transports.RequestInternalHeaderName))
		if len(internalHeader) == 0 {
			h.handleRequest(w, r)
		} else {
			h.handleInternalRequest(w, r)
		}
		return
	}
}

func (h *Handler) prepareDocuments() {
	h.once.Do(func() {
		h.doc = h.rt.Endpoints().Documents()
		h.openapi = h.doc.Openapi("", h.rt.AppId(), h.rt.AppName(), h.rt.AppVersion())
	})
}

func (h *Handler) handleDocuments(w transports.ResponseWriter, _ transports.Request) {
	h.prepareDocuments()
	w.Succeed(h.doc)
}

func (h *Handler) handleOpenapi(w transports.ResponseWriter, _ transports.Request) {
	v, err, _ := h.group.Do("documents", func() (interface{}, error) {
		h.prepareDocuments()
		p, err := h.openapi.Encode()
		if err != nil {
			return nil, errors.Warning("fns: encode openapi failed").WithCause(err)
		}
		return p, nil
	})
	if err != nil {
		w.Failed(errors.Map(err))
		return
	}
	w.Succeed(v)
}

func (h *Handler) handleRequest(w transports.ResponseWriter, r transports.Request) {
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}

	// header >>>
	options := make([]services.RequestOption, 0, 1)
	// device id
	deviceId := r.Header().Get(bytex.FromString(transports.DeviceIdHeaderName))
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	options = append(options, services.WithDeviceId(deviceId))
	// device ip
	deviceIp := r.Header().Get(bytex.FromString(transports.DeviceIpHeaderName))
	if len(deviceIp) > 0 {
		options = append(options, services.WithDeviceIp(deviceIp))
	}
	// request id
	requestId := r.Header().Get(bytex.FromString(transports.RequestIdHeaderName))
	if len(requestId) > 0 {
		options = append(options, services.WithRequestId(requestId))
	}
	// request version
	acceptedVersions := r.Header().Get(bytex.FromString(transports.RequestVersionsHeaderName))
	if len(acceptedVersions) > 0 {
		intervals, intervalsErr := versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		options = append(options, services.WithRequestVersions(intervals))
	}
	// authorization
	authorization := r.Header().Get(bytex.FromString(transports.AuthorizationHeaderName))
	if len(authorization) > 0 {
		options = append(options, services.WithAuthorization(authorization))
	}

	// header <<<

	// ctx
	ctx := transports.WithRequest(r, r)
	ctx = transports.WithResponse(ctx, w)

	// do
	future := h.rt.Endpoints().Request(
		ctx, service, fn,
		services.NewArgument(body),
		options...,
	)

	result, err := future.Get(ctx)
	if err != nil {
		w.Failed(err)
		return
	}
	if result.Exist() {
		w.Succeed(result)
	} else {
		w.Succeed(nil)
	}

}

func (h *Handler) handleInternalRequest(w transports.ResponseWriter, r transports.Request) {
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	// body
	body, bodyErr := r.Body()
	if bodyErr != nil {
		w.Failed(ErrInvalidBody.WithMeta("path", bytex.ToString(path)))
		return
	}

}
