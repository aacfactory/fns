package proxies

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/mmhash"
	"github.com/aacfactory/fns/commons/scanner"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/sync/singleflight"
	"net/http"
	"strconv"
)

var (
	slashBytes = []byte{'/'}
)

func NewProxyHandler(manager clusters.ClusterEndpointsManager, dialer transports.Dialer) transports.MuxHandler {
	return &proxyHandler{
		manager: manager,
		dialer:  dialer,
		group:   singleflight.Group{},
	}
}

type proxyHandler struct {
	manager clusters.ClusterEndpointsManager
	dialer  transports.Dialer
	group   singleflight.Group
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
	groupKeyBuf := bytebufferpool.Get()

	// path
	path := r.Path()
	pathItems := bytes.Split(path, slashBytes)
	if len(pathItems) != 3 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrInvalidPath.WithMeta("path", bytex.ToString(path)))
		return
	}
	service := pathItems[1]
	fn := pathItems[2]
	_, _ = groupKeyBuf.Write(path)
	// device id
	deviceId := r.Header().Get(transports.DeviceIdHeaderName)
	if len(deviceId) == 0 {
		bytebufferpool.Put(groupKeyBuf)
		w.Failed(ErrDeviceId.WithMeta("path", bytex.ToString(path)))
		return
	}
	_, _ = groupKeyBuf.Write(deviceId)

	// discovery
	endpointGetOptions := make([]services.EndpointGetOption, 0, 1)
	var intervals versions.Intervals
	acceptedVersions := r.Header().Get(transports.RequestVersionsHeaderName)
	if len(acceptedVersions) > 0 {
		var intervalsErr error
		intervals, intervalsErr = versions.ParseIntervals(acceptedVersions)
		if intervalsErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(ErrInvalidRequestVersions.WithMeta("path", bytex.ToString(path)).WithMeta("versions", bytex.ToString(acceptedVersions)).WithCause(intervalsErr))
			return
		}
		endpointGetOptions = append(endpointGetOptions, services.EndpointVersions(intervals))
		_, _ = groupKeyBuf.Write(acceptedVersions)
	}

	var queryParams transports.Params
	var body []byte
	method := r.Method()
	if bytes.Equal(method, transports.MethodGet) {
		queryParams = r.Params()
		queryParamsBytes := queryParams.Encode()
		path = append(path, '?')
		path = append(path, queryParamsBytes...)
		_, _ = groupKeyBuf.Write(queryParamsBytes)
	} else {
		var bodyErr error
		body, bodyErr = r.Body()
		if bodyErr != nil {
			bytebufferpool.Put(groupKeyBuf)
			w.Failed(errors.Warning("fns: read request body failed").WithCause(bodyErr).
				WithMeta("service", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn)))
			return
		}
		_, _ = groupKeyBuf.Write(body)
	}

	groupKey := strconv.FormatUint(mmhash.Sum64(groupKeyBuf.Bytes()), 16)
	bytebufferpool.Put(groupKeyBuf)
	v, err, _ := handler.group.Do(groupKey, func() (v interface{}, err error) {
		address, has := handler.manager.PublicFnAddress(r, service, fn, endpointGetOptions...)
		if !has {
			err = errors.NotFound("fns: endpoint was not found").
				WithMeta("service", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		if address == handler.manager.Address() {
			var param scanner.Scanner
			if bytes.Equal(method, transports.MethodGet) {
				param = transports.ParamsScanner(queryParams)
			} else {
				param = json.RawMessage(body)
			}
			response, handleErr := handler.manager.Request(r, service, fn, param)
			if handleErr != nil {
				err = handleErr
				return
			}
			v = Response{
				Header: nil,
				Value:  response,
			}
			return
		}
		client, clientErr := handler.dialer.Dial(bytex.FromString(address))
		if clientErr != nil {
			err = errors.Warning("fns: dial endpoint failed").WithCause(clientErr).
				WithMeta("service", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		status, respHeader, respBody, doErr := client.Do(r, method, path, r.Header(), body)
		if doErr != nil {
			err = errors.Warning("fns: send request to endpoint failed").WithCause(doErr).
				WithMeta("service", bytex.ToString(service)).
				WithMeta("fn", bytex.ToString(fn))
			return
		}
		if status == http.StatusOK {
			respHeader.Foreach(func(key []byte, values [][]byte) {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			})
			v = Response{
				Header: respHeader,
				Value:  respBody,
			}
			return
		}
		err = errors.Decode(respBody)
		return
	})

	if err != nil {
		w.Failed(err)
		return
	}

	response := v.(Response)
	if response.Header.Len() > 0 {
		response.Header.Foreach(func(key []byte, values [][]byte) {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		})
	}
	w.Succeed(response.Value)
}

type Response struct {
	Header transports.Header
	Value  interface{}
}
