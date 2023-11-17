package development

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"sync"
)

var (
	discoveryHandlePathPrefix    = []byte("/development/discovery/")
	discoveryHandleEndpointsPath = []byte("/development/discovery/endpoints")
	discoveryHandleEndpointPath  = []byte("/development/discovery/endpoint")
)

func NewDiscovery(log logs.Logger, address string, dialer transports.Dialer, signature signatures.Signature) services.Discovery {
	return &Discovery{
		log:       log,
		address:   []byte(address),
		dialer:    dialer,
		signature: signature,
		client:    nil,
		once:      sync.Once{},
	}
}

type Discovery struct {
	log       logs.Logger
	address   []byte
	dialer    transports.Dialer
	signature signatures.Signature
	client    transports.Client
	once      sync.Once
}

func (discovery *Discovery) Endpoints(ctx context.Context) (infos []services.EndpointInfo) {
	body := []byte{'{', '}'}
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), discovery.signature.Sign(body))
	client := discovery.Client()
	status, _, responseBody, err := client.Do(ctx, methodPost, discoveryHandleEndpointsPath, header, body)
	if err != nil {
		if discovery.log.WarnEnabled() {
			discovery.log.Warn().Cause(errors.Warning("fns: fetch endpoints failed").WithCause(err)).Message("fns: fetch endpoints failed")
		}
		return
	}
	if status == 200 {
		infos = make([]services.EndpointInfo, 0, 1)
		decodeErr := json.Unmarshal(responseBody, &infos)
		if decodeErr != nil {
			if discovery.log.WarnEnabled() {
				discovery.log.Warn().Cause(errors.Warning("fns: fetch endpoints failed").WithCause(decodeErr)).Message("fns: fetch endpoints failed")
			}
			return
		}
		return
	}
	return
}

type DiscoveryGetParam struct {
	Name     []byte             `json:"name"`
	Id       []byte             `json:"id"`
	Versions versions.Intervals `json:"versions"`
}

type DiscoveryGetResult struct {
	Has      bool               `json:"has"`
	Internal bool               `json:"internal"`
	Document documents.Document `json:"document"`
}

func (discovery *Discovery) Get(ctx context.Context, name []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {
	opt := services.EndpointGetOptions{}
	for _, option := range options {
		option(&opt)
	}
	param := DiscoveryGetParam{
		Name:     name,
		Id:       opt.Id(),
		Versions: opt.Versions(),
	}
	body, _ := json.Marshal(param)
	header := transports.AcquireHeader()
	defer transports.ReleaseHeader(header)
	header.Set(bytex.FromString(transports.ContentTypeHeaderName), contentType)
	// signature
	header.Set(bytex.FromString(transports.SignatureHeaderName), discovery.signature.Sign(body))
	client := discovery.Client()
	status, _, responseBody, err := client.Do(ctx, methodPost, discoveryHandleEndpointPath, header, body)
	if err != nil {
		if discovery.log.WarnEnabled() {
			discovery.log.Warn().Cause(errors.Warning("fns: fetch endpoint failed").WithCause(err)).Message("fns: fetch endpoint failed")
		}
		return
	}
	if status == 200 {
		result := DiscoveryGetResult{}
		decodeErr := json.Unmarshal(responseBody, &result)
		if decodeErr != nil {
			if discovery.log.WarnEnabled() {
				discovery.log.Warn().Cause(errors.Warning("fns: fetch endpoints failed").WithCause(decodeErr)).Message("fns: fetch endpoints failed")
			}
			return
		}
		has = result.Has
		if has {
			endpoint = NewEndpoint(name, result.Internal, result.Document, client, discovery.signature)
		}
		return
	}
	return
}

func (discovery *Discovery) Client() transports.Client {
	discovery.once.Do(func() {
		client, clientErr := discovery.dialer.Dial(discovery.address)
		if clientErr != nil {
			panic(fmt.Sprintf("%+v", errors.Warning("fns: dial failed").WithCause(clientErr).WithMeta("cluster", "development")))
			return
		}
		discovery.client = client
	})
	return discovery.client
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewDiscoveryHandler(signature signatures.Signature, endpoints services.Endpoints) transports.Handler {
	return &DiscoveryHandler{
		signature: signature,
		endpoints: endpoints,
	}
}

type DiscoveryHandler struct {
	signature signatures.Signature
	endpoints services.Endpoints
}

func (handler *DiscoveryHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	path := r.Path()
	if bytes.Equal(path, discoveryHandleEndpointsPath) {
		infos := handler.discovery.Endpoints(r)
		w.Succeed(infos)
	} else if bytes.Equal(path, discoveryHandleEndpointPath) {
		body, bodyErr := r.Body()
		if bodyErr != nil {
			w.Failed(ErrInvalidBody.WithCause(bodyErr))
			return
		}
		param := DiscoveryGetParam{}
		decodeErr := json.Unmarshal(body, &param)
		if decodeErr != nil {
			w.Failed(ErrInvalidBody.WithCause(decodeErr))
			return
		}
		options := make([]services.EndpointGetOption, 0, 1)
		if len(param.Id) > 0 {
			options = append(options, services.EndpointId(param.Id))
		}
		if len(param.Versions) > 0 {
			options = append(options, services.EndpointVersions(param.Versions))
		}
		endpoint, has := handler.discovery.Get(r, param.Name, options...)
		result := DiscoveryGetResult{
			Has:      has,
			Internal: false,
		}
		if has {
			result.Internal = endpoint.Internal()
			result.Document = endpoint.Document()
		}
		w.Succeed(result)
	} else {
		w.Failed(ErrInvalidPath.WithMeta("path", string(path)))
	}
	return
}
