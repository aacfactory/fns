package configs

type Config struct {
	Runtime   *RuntimeConfig   `json:"runtime" yaml:"runtime,omitempty"`
	Log       *LogConfig       `json:"log" yaml:"log,omitempty"`
	Transport *TransportConfig `json:"transport" yaml:"transport,omitempty"`
	Cluster   *ClusterConfig   `json:"cluster" yaml:"cluster,omitempty"`
	Proxy     *ProxyConfig     `json:"proxy" yaml:"proxy,omitempty"`
	Services  json.RawMessage  `json:"services" yaml:"services,omitempty"`
}
