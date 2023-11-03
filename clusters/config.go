package clusters

import (
	"github.com/aacfactory/json"
	"time"
)

type BarrierConfig struct {
	TTL      time.Duration `json:"ttl"`
	Interval time.Duration `json:"interval"`
}

type Config struct {
	Secret        string          `json:"secret"`
	HostRetriever string          `json:"hostRetriever"`
	Barrier       *BarrierConfig  `json:"barrier"`
	Kind          string          `json:"kind"`
	Option        json.RawMessage `json:"option"`
}
