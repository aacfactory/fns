package development

const (
	Server = Mode("server")
	Client = Mode("client")
)

type Mode string

type Config struct {
	Enabled bool   `json:"enabled"`
	Mode    Mode   `json:"mode"`
	Address string `json:"address"`
}
