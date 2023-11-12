package cachecontrol

type Config struct {
	Enable    bool `json:"enable"`
	ProxyMode bool `json:"proxyMode"`
	MaxAge    int  `json:"maxAge"`
}
