package fns

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/logs"
	"github.com/rs/xid"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	HttpMaxRequestBodySize = int64(32 * MB)
)

type HttpServiceConfig struct {
	Name       string      `json:"name,omitempty"`
	Host       string      `json:"host,omitempty"`
	Port       int         `json:"port,omitempty"`
	PublicHost string      `json:"publicHost,omitempty"`
	PublicPort int         `json:"publicPort,omitempty"`
	TLS        ServiceTLS  `json:"tls,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
	Meta       ServiceMeta `json:"meta,omitempty"`
}

func NewHttpServiceFnRegister() *HttpServiceFnRegister {
	return &HttpServiceFnRegister{
		lock:       sync.Mutex{},
		fnProxyMap: make(map[string]FnProxy),
	}
}

type HttpServiceFnRegister struct {
	lock       sync.Mutex
	fnProxyMap map[string]FnProxy
}

func (r *HttpServiceFnRegister) Add(fnAddr string, proxy FnProxy) (err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	fnAddr = strings.TrimSpace(fnAddr)
	if fnAddr == "" {
		err = fmt.Errorf("fns register fn failed, fn addr is empty")
		return
	}
	if proxy == nil {
		err = fmt.Errorf("fns register fn failed, proxy is nil")
		return
	}
	_, has := r.fnProxyMap[fnAddr]
	if has {
		err = fmt.Errorf("fns register fn failed, %s proxy exists", fnAddr)
		return
	}
	r.fnProxyMap[fnAddr] = proxy
	return
}

func NewHttpService(register *HttpServiceFnRegister) (service Service) {

	if register == nil || len(register.fnProxyMap) == 0 {
		panic(fmt.Errorf("fns create http service failed, no fn registered"))
	}

	service = &httpService{
		fnProxyMap: register.fnProxyMap,
	}

	return
}

type httpService struct {
	name         string
	port         int
	ln           net.Listener
	fnProxyMap   map[string]FnProxy
	log          Logs
	eventbus     Eventbus
	clusterMode  bool
	cluster      Cluster
	registration Registration
}

func (service *httpService) Start(context Context, env Environment) (err error) {
	config := &HttpServiceConfig{}
	configErr := env.Config().Get("http", config)
	if configErr != nil {
		err = fmt.Errorf("fns start http service failed, read http config failed, %v", configErr)
		return
	}
	name := strings.TrimSpace(config.Name)
	if name == "" {
		err = fmt.Errorf("fns start http service failed, no name in http config")
		return
	}

	publicHost := strings.TrimSpace(config.PublicHost)
	host := strings.TrimSpace(config.Host)
	if publicHost == "" {
		if host == "" {
			hostIp, hostIpErr := IpFromHostname(false)
			if hostIpErr != nil || hostIp == "" {
				err = fmt.Errorf("fns create http failed, can not get ip from hostname, please set public host in config, %v", hostIpErr)
				return
			}
			publicHost = hostIp
		}
	}
	if host == "" {
		host = "0.0.0.0"
	}
	port := config.Port
	if port < 1 || port > 65535 {
		err = fmt.Errorf("fns create http failed, port is invalied")
		return
	}
	service.port = port
	publicPort := config.PublicPort
	if publicPort == 0 {
		publicPort = port
	}
	if publicPort < 1 || publicPort > 65535 {
		err = fmt.Errorf("fns create http failed, public port is invalied")
		return
	}
	protocol := "http"
	var serverTLSConfig *tls.Config
	serviceTLS := config.TLS
	if serviceTLS.Enable() {
		serverTLSConfig0, tlsErr := serviceTLS.ToServerTLSConfig()
		if tlsErr != nil {
			err = fmt.Errorf("fns create http failed, tls is enabled, but build tls config failed, %v", tlsErr)
			return
		}
		serverTLSConfig = serverTLSConfig0
		protocol = "https"
	}

	if serverTLSConfig != nil {
		ln, lnErr := tls.Listen("tcp", fmt.Sprintf("%s:%d", host, port), serverTLSConfig)
		if lnErr != nil {
			err = fmt.Errorf("fns create http failed, tls is enabled, but listen failed, %v", lnErr)
			return
		}
		service.ln = ln
	} else {
		ln, lnErr := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
		if lnErr != nil {
			err = fmt.Errorf("fns create http failed, listen failed, %v", lnErr)
			return
		}
		service.ln = ln
	}

	tags := config.Tags
	if tags == nil {
		tags = make([]string, 0, 1)
	}
	meta := config.Meta
	if meta == nil {
		meta = NewServiceMeta()
	}

	httpLog, withErr := logs.With(context.Log(), logs.F("http", service.name))
	if withErr != nil {
		err = fmt.Errorf("fns create http service failed, make http log failed")
		return
	}
	service.log = httpLog

	service.serve(context)
	time.Sleep(1 * time.Second)
	if env.ClusterMode() {
		registration, publishErr := env.Discovery().Publish("http", service.name, protocol, fmt.Sprintf("%s:%d", publicHost, publicPort), tags, meta, serviceTLS)
		if publishErr != nil {
			err = fmt.Errorf("fns create http service failed, publish http service failed, %v", publishErr)
			_ = service.ln.Close()
			return
		}
		service.registration = registration
	}

	return
}

func (service *httpService) Stop(context Context) (err error) {
	service.log.Debugf("fns http service begin to stop")
	if service.clusterMode {
		service.log.Debugf("fns http service begin to un publish service")
		unPublishErr := service.cluster.UnPublish(service.registration)
		if unPublishErr != nil {
			service.log.Warnf("fns http service un publish service failed, %v", unPublishErr)
		}
		service.log.Debugf("fns http service un publish service is finished")
	}
	_ = service.ln.Close()
	service.log.Debugf("fns http service stopped")
	return
}

func (service *httpService) serve(context Context) {
	mux := service.httpServiceRegisterFnHandle(context)
	go func(ln net.Listener, mux *http.ServeMux) {
		service.log.Debugf("fns http service listen at %d", service.port)
		err := http.Serve(ln, mux)
		if err != nil {
			service.log.Errorf("fns serve http failed, %v", err)
		}
	}(service.ln, mux)
}

func (service *httpService) httpServiceRegisterFnHandle(context Context) (mux *http.ServeMux) {

	mux = http.NewServeMux()

	for fnAddr, proxy := range service.fnProxyMap {
		handler := newFnHttpHandler(context, fnAddr, proxy)
		mux.Handle(fmt.Sprintf("_fn_/%s", fnAddr), handler)
	}

	return
}

func newFnHttpHandler(context Context, fnAddr string, fnProxy FnProxy) (h *fnHttpHandler) {
	h = &fnHttpHandler{
		fnAddr:  fnAddr,
		context: context,
		fnProxy: fnProxy,
	}
	return
}

type fnHttpHandler struct {
	context     Context
	clusterMode bool
	fnAddr      string
	fnProxy     FnProxy
}

func (h *fnHttpHandler) tags(r *http.Request) (tags []string) {
	tags = r.Header.Values("X-Fns-Tags")
	if tags == nil {
		tags = make([]string, 0, 1)
	}
	return
}

func (h *fnHttpHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		rw.WriteHeader(405)
		_, _ = rw.Write(JsonEncode(map[string]string{"Error": "http method is not supported, please use POST!"}))
		return
	}

	rid := xid.New().String()

	fc := newFnsFnContext(h.fnAddr, rid, h.context, h.clusterMode)

	arguments := newFnHttpRequestArguments(r)

	tags := h.tags(r)

	result, proxyErr := h.fnProxy(fc, arguments, tags...)
	if proxyErr != nil {
		codeErr := errors.Map(proxyErr)
		rw.WriteHeader(codeErr.FailureCode())
		_, _ = rw.Write(codeErr.ToJson())
		return
	}

	if result == nil {
		return
	}

	data, encodeErr := JsonAPI().Marshal(result)
	if encodeErr != nil {
		codeErr := errors.Map(encodeErr)
		rw.WriteHeader(codeErr.FailureCode())
		_, _ = rw.Write(codeErr.ToJson())
		return
	}

	rw.WriteHeader(200)
	_, _ = rw.Write(data)
	return
}
