package clusters

import "github.com/aacfactory/json"

type Entry struct {
	Key string          `json:"key"`
	Val json.RawMessage `json:"val"`
}

type RequestBody struct {
	UserValues []Entry         `json:"userValues"`
	Argument   json.RawMessage `json:"argument"`
}

type ResponseBody struct {
	Succeed bool            `json:"succeed"`
	Data    json.RawMessage `json:"data"`
	Span    json.RawMessage `json:"span"`
}
