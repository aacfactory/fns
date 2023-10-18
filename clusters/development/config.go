package development

import "github.com/aacfactory/fns/transports"

type Config struct {
	ProxyTransportName string                `json:"proxyTransportName"`
	ProxyAddress       string                `json:"proxyAddress"`
	TLS                *transports.TLSConfig `json:"tls"`
}
