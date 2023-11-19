package clusters

import (
	"github.com/aacfactory/json"
)

type Config struct {
	Secret        string          `json:"secret"`
	HostRetriever string          `json:"hostRetriever"`
	Barrier       BarrierConfig   `json:"barrier"`
	Name          string          `json:"name"`
	Proxy         bool            `json:"proxy"`
	Option        json.RawMessage `json:"option"`
}
