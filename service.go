package fns

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
)

type Service interface {
	Start(context Context, env Environment) (err error)
	Stop(context Context) (err error)
}

// +-------------------------------------------------------------------------------------------------------------------+

func NewServiceMeta() ServiceMeta {
	return make(map[string]string)
}

type ServiceMeta map[string]string

func (meta ServiceMeta) Put(key string, value string) {
	meta[key] = value
}

func (meta ServiceMeta) Get(key string) (string, bool) {
	if meta == nil {
		return "", false
	}
	v, has := meta[key]
	return v, has
}

func (meta ServiceMeta) Rem(key string) {
	delete(meta, key)
}

func (meta ServiceMeta) Keys() []string {
	if meta.Empty() {
		return nil
	}
	keys := make([]string, 0, 1)
	for key := range meta {
		keys = append(keys, key)
	}
	return keys
}

func (meta ServiceMeta) Empty() bool {
	return meta == nil || len(meta) == 0
}

func (meta ServiceMeta) Merge(o ...Meta) {
	if o == nil || len(o) == 0 {
		return
	}
	for _, other := range o {
		keys := other.Keys()
		for _, k := range keys {
			v, has := other.Get(k)
			if has {
				meta.Put(k, v)
			}
		}
	}
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServiceTLS struct {
	Enable_     bool   `json:"enable,omitempty"`
	VerifySSL_  bool   `json:"verifySsl,omitempty"`
	CA_         string `json:"ca,omitempty"`
	ServerCert_ string `json:"serverCert,omitempty"`
	ServerKey_  string `json:"serverKey,omitempty"`
	ClientCert_ string `json:"clientCert,omitempty"`
	ClientKey_  string `json:"clientKey,omitempty"`
}

func (s ServiceTLS) Enable() bool {
	return s.Enable_
}

func (s ServiceTLS) VerifySSL() bool {
	return s.VerifySSL_
}

func (s ServiceTLS) CA() string {
	return s.CA_
}

func (s ServiceTLS) ServerCert() string {
	return s.ServerCert_
}

func (s ServiceTLS) ServerKey() string {
	return s.ServerKey_
}

func (s ServiceTLS) ClientCert() string {
	return s.ClientCert_
}

func (s ServiceTLS) ClientKey() string {
	return s.ClientKey_
}

func (s ServiceTLS) ToServerTLSConfig() (config *tls.Config, err error) {
	if !s.Enable() {
		err = fmt.Errorf("generate endpint server tls config failed, tls not enabled")
		return
	}
	if s.ServerCert() == "" || s.ServerKey() == "" {
		err = fmt.Errorf("generate endpint server tls config failed, key is empty")
		return
	}

	certificate, certificateErr := tls.X509KeyPair([]byte(s.ServerCert()), []byte(s.ServerKey()))
	if certificateErr != nil {
		err = fmt.Errorf("generate endpint server tls config failed, %v", certificateErr)
		return
	}

	config = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		Rand:               rand.Reader,
		InsecureSkipVerify: !s.VerifySSL(),
		ClientAuth:         tls.RequireAndVerifyClientCert,
	}

	if s.CA() != "" {
		cas := x509.NewCertPool()
		ok := cas.AppendCertsFromPEM([]byte(s.CA()))
		if !ok {
			err = fmt.Errorf("generate endpint server tls config failed, append ca failed")
			return
		}
		config.ClientCAs = cas
	}

	return
}

func (s ServiceTLS) ToClientTLSConfig() (config *tls.Config, err error) {
	if !s.Enable() {
		err = fmt.Errorf("generate endpint client tls config failed, tls not enabled")
		return
	}
	if s.ClientCert() == "" || s.ClientKey() == "" {
		err = fmt.Errorf("generate endpint client tls config failed, key is empty")
		return
	}

	certificate, certificateErr := tls.X509KeyPair([]byte(s.ClientCert()), []byte(s.ClientKey()))
	if certificateErr != nil {
		err = fmt.Errorf("generate endpint client tls config failed, %v", certificateErr)
		return
	}

	config = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: !s.VerifySSL(),
	}

	if s.CA() != "" {
		cas := x509.NewCertPool()
		ok := cas.AppendCertsFromPEM([]byte(s.CA()))
		if !ok {
			err = fmt.Errorf("generate endpint client tls config failed, append ca failed")
			return
		}
		config.RootCAs = cas
	}

	return
}
