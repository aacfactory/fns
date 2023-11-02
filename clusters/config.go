package clusters

import (
	"github.com/aacfactory/json"
)

type Config struct {
	Secret        string          `json:"secret"`
	HostRetriever string          `json:"hostRetriever"`
	Kind          string          `json:"kind"`
	Option        json.RawMessage `json:"option"`
}
