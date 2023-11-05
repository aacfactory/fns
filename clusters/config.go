package clusters

import (
	"github.com/aacfactory/json"
)

type Config struct {
	Secret        string          `json:"secret"`
	HostRetriever string          `json:"hostRetriever"`
	Shared        json.RawMessage `json:"shared"`
	Barrier       json.RawMessage `json:"barrier"`
	Name          string          `json:"name"`
	Option        json.RawMessage `json:"option"`
}
