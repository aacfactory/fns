package clusters

import (
	"github.com/aacfactory/json"
)

type Config struct {
	DevMode       bool            `json:"devMode"` // 标记cluster开启代理访问功能，与dev cluster 互斥。
	Secret        string          `json:"secret"`
	HostRetriever string          `json:"hostRetriever"`
	Barrier       BarrierConfig   `json:"barrier"`
	Name          string          `json:"name"`
	Option        json.RawMessage `json:"option"`
}
