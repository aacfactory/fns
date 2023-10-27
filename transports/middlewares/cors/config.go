package cors

type Config struct {
	AllowedOrigins      []string `json:"allowedOrigins"`
	AllowedHeaders      []string `json:"allowedHeaders"`
	ExposedHeaders      []string `json:"exposedHeaders"`
	AllowCredentials    bool     `json:"allowCredentials"`
	MaxAge              int      `json:"maxAge"`
	AllowPrivateNetwork bool     `json:"allowPrivateNetwork"`
}
