package transports

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/service/ssl"
	"github.com/aacfactory/json"
	"strings"
)

type TLSConfig struct {
	// Kind
	// ACME
	// SSC(SELF-SIGN-CERT)
	// DEFAULT
	Kind    string          `json:"kind" yaml:"kind,omitempty"`
	Options json.RawMessage `json:"options" yaml:"options,omitempty"`
}

func (config *TLSConfig) Config() (conf ssl.Config, err error) {
	kind := strings.TrimSpace(config.Kind)
	hasConf := false
	conf, hasConf = ssl.GetConfig(kind)
	if !hasConf {
		err = errors.Warning(fmt.Sprintf("fns: can not get %s tls config", kind))
		return
	}
	confOptions, confOptionsErr := configures.NewJsonConfig(config.Options)
	if confOptionsErr != nil {
		err = errors.Warning(fmt.Sprintf("fns: can not get options of %s tls config", kind)).WithCause(confOptionsErr)
		return
	}
	err = conf.Build(confOptions)
	return
}

type Config struct {
	Port_        int             `json:"port" yaml:"port,omitempty"`
	TLS_         *TLSConfig      `json:"tls" yaml:"tls,omitempty"`
	Options_     json.RawMessage `json:"options" yaml:"options,omitempty"`
	Middlewares_ json.RawMessage `json:"middlewares" yaml:"middlewares,omitempty"`
	Handlers_    json.RawMessage `json:"handlers" yaml:"handlers,omitempty"`
}

func (config *Config) Port() (port int, err error) {
	port = config.Port_
	if port == 0 {
		if config.TLS_ == nil {
			port = 80
		} else {
			port = 443
		}
	}
	if port < 1 || port > 65535 {
		err = errors.Warning("port is invalid, port must great than 1024 or less than 65536")
		return
	}
	return
}

func (config *Config) TLS() (tls ssl.Config, err error) {
	if config.TLS_ == nil {
		return
	}
	tls, err = config.TLS_.Config()
	if err != nil {
		err = errors.Warning("tls is invalid").WithCause(err)
		return
	}
	return
}

func (config *Config) Options() (options configures.Config, err error) {
	options, err = configures.NewJsonConfig(config.Options_)
	if err != nil {
		err = errors.Warning("options is invalid").WithCause(err)
		return
	}
	return
}

func (config *Config) Middleware(name string) (middleware configures.Config, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.Warning("middleware is invalid").WithCause(fmt.Errorf("name is nil")).WithMeta("middleware", name)
		return
	}
	middlewares, middlewaresErr := configures.NewJsonConfig(config.Middlewares_)
	if middlewaresErr != nil {
		err = errors.Warning("middleware is invalid").WithCause(middlewaresErr).WithMeta("middleware", name)
		return
	}
	has := false
	middleware, has = middlewares.Node(name)
	if !has {
		middleware, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	return
}

func (config *Config) Handler(name string) (handler configures.Config, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		err = errors.Warning("middleware is invalid").WithCause(fmt.Errorf("name is nil")).WithMeta("middleware", name)
		return
	}
	handlers, handlersErr := configures.NewJsonConfig(config.Handlers_)
	if handlersErr != nil {
		err = errors.Warning("middleware is invalid").WithCause(handlersErr).WithMeta("middleware", name)
		return
	}
	has := false
	handler, has = handlers.Node(name)
	if !has {
		handler, _ = configures.NewJsonConfig([]byte{'{', '}'})
	}
	return
}
