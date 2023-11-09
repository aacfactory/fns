package cachecontrol

type Config struct {
	Enable bool `json:"enable"`
	MaxAge int  `json:"maxAge"`
}
