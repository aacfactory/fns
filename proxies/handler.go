package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"net/http"
)

var (
	slashBytes = []byte{'/'}
)

func NewProxyHandler(manager clusters.ClusterEndpointsManager, dialer transports.Dialer) transports.MuxHandler {
	return &proxyHandler{
		manager: manager,
		dialer:  dialer,
	}
}

type proxyHandler struct {
	manager clusters.ClusterEndpointsManager
	dialer  transports.Dialer
}

func (handler *proxyHandler) Name() string {
	return "proxy"
}

func (handler *proxyHandler) Construct(_ transports.MuxHandlerOptions) error {
	return nil
}

func (handler *proxyHandler) Match(_ context.Context, method []byte, path []byte, header transports.Header) bool {
	if len(header.Get(transports.UpgradeHeaderName)) > 0 {
		return false
	}
	if bytes.Equal(method, transports.MethodPost) {
		return len(bytes.Split(path, slashBytes)) == 3 && bytes.Equal(header.Get(transports.ContentTypeHeaderName), transports.ContentTypeJsonHeaderValue)
	}
	if bytes.Equal(method, transports.MethodGet) {
		return len(bytes.Split(path, slashBytes)) == 3
	}
	return false
}

func (handler *proxyHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	if len(deviceId) == 0 {
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	// discovery
	endpointGetOptions := make([]services.EndpointGetOption, 0, 1)
	var intervals versions.Intervals
	acceptedVersions := r.Header().Get(transports.RequestVersionsHeaderName)
	if len(acceptedVersions) > 0 {
		var intervalsErr error
		intervals, intervalsErr = versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		endpointGetOptions = append(endpointGetOptions, services.EndpointVersions(intervals))
	}

	address, has := handler.manager.PublicFnAddress(r, service, fn, endpointGetOptions...)
	if !has {
		w.Failed(errors.NotFound("fns: endpoint was not found").
			WithMeta("service", bytex.ToString(service)).
			WithMeta("fn", bytex.ToString(fn)))
		return
	}

	client, clientErr := handler.dialer.Dial(bytex.FromString(address))
	if clientErr != nil {
		w.Failed(errors.Warning("fns: dial endpoint failed").WithCause(clientErr).
			WithMeta("service", bytex.ToString(service)).
			WithMeta("fn", bytex.ToString(fn)))
		return
	}
	method := r.Method()
	var body []byte
	if bytes.Equal(method, transports.MethodGet) {
		params := r.Params().Encode()
		path = append(path, '?')
		path = append(path, params...)
	} else {
		var bodyErr error
		body, bodyErr = r.Body()
		if bodyErr != nil {
			w.Failed(errors.Warning("fns: read request body failed").WithCause(bodyErr).
				WithMeta("service", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn)))
			return
		}
	}

	status, respHeader, respBody, doErr := client.Do(r, method, path, r.Header(), body)
	if doErr != nil {
		w.Failed(errors.Warning("fns: send request to endpoint failed").WithCause(doErr).
			WithMeta("service", bytex.ToString(service)).
			WithMeta("fn", bytex.ToString(fn)))
		return
	}
	respHeader.Foreach(func(key []byte, values [][]byte) {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	})
	if status == http.StatusOK {
		w.Succeed(json.RawMessage(respBody))
		return
	}
	respErr := errors.Decode(respBody)
	w.Failed(respErr)
}
