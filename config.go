/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fns

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	B = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB

	activeSystemEnvKey = "FNS-ACTIVE"
)

func defaultConfigRetrieverOption() (option configuares.RetrieverOption) {
	path, pathErr := filepath.Abs("./config")
	if pathErr != nil {
		panic(fmt.Sprintf("fns create default config retriever failed, cant not get './config'"))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configuares.NewFileStore(path, "fns", '-')
	option = configuares.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

type Config struct {
	Log  logConfig  `json:"log,omitempty"`
	Http HttpConfig `json:"http,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type httpCorsConfig struct {
	AllowedOrigins   []string `json:"allowedOrigins,omitempty"`
	AllowedHeaders   []string `json:"allowedHeaders,omitempty"`
	ExposedHeaders   []string `json:"exposedHeaders,omitempty"`
	AllowCredentials bool     `json:"allowCredentials,omitempty"`
	MaxAge           int      `json:"maxAge,omitempty"`
}

type websocketConfig struct {
	HandshakeTimeoutSeconds int    `json:"handshakeTimeoutSeconds"`
	ReadBufferSize          string `json:"readBufferSize"`
	WriteBufferSize         string `json:"writeBufferSize"`
}

type HttpConfig struct {
	Port    int             `json:"port"`
	TLS     *TLSConfig      `json:"tls"`
	Options json.RawMessage `json:"options"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type ServerTLSConfig struct {
	Cert           string   `json:"cert"`
	Key            string   `json:"key"`
	ClientCAs      []string `json:"clientCAs"`
	ClientAuthType string   `json:"clientAuthType"`
}

func (config *ServerTLSConfig) Config() (v *tls.Config, err error) {
	clientAuthType := tls.NoClientCert
	config.ClientAuthType = strings.ToLower(strings.TrimSpace(config.ClientAuthType))
	switch config.ClientAuthType {
	case "no":
		clientAuthType = tls.NoClientCert
	case "request":
		clientAuthType = tls.RequestClientCert
	case "require":
		clientAuthType = tls.RequireAnyClientCert
	case "verify":
		clientAuthType = tls.VerifyClientCertIfGiven
	case "require+verify":
		clientAuthType = tls.RequireAndVerifyClientCert
	default:
		err = fmt.Errorf("fns: server clientAuthType is invalid")
		return
	}
	cert := strings.TrimSpace(config.Cert)
	if cert == "" {
		err = fmt.Errorf("fns: server cert is empty")
		return
	}
	certPEM, certErr := decodeTlsSource(cert)
	if certErr != nil {
		err = fmt.Errorf("fns: read server cert failed, %v", certErr)
		return
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		err = fmt.Errorf("fns: server key is empty")
		return
	}
	keyPEM, keyErr := decodeTlsSource(key)
	if keyErr != nil {
		err = fmt.Errorf("fns: read server key failed, %v", keyErr)
		return
	}
	certificate, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
	if certificateErr != nil {
		err = fmt.Errorf("fns: create server x509 keypair failed, %v", certificateErr)
		return
	}
	var clientCAs *x509.CertPool
	if config.ClientCAs != nil && len(config.ClientCAs) > 0 {
		clientCAs = x509.NewCertPool()
		for _, ca := range config.ClientCAs {
			ca = strings.TrimSpace(ca)
			if ca == "" {
				err = fmt.Errorf("fns: one of client ca is empty")
				return
			}
			caPEM, caErr := decodeTlsSource(ca)
			if caErr != nil {
				err = fmt.Errorf("fns: read client ca failed, %v", caErr)
				return
			}
			if ok := clientCAs.AppendCertsFromPEM(caPEM); !ok {
				err = fmt.Errorf("fns: append client ca cert failed")
				return
			}
		}
	}
	v = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    clientCAs,
		ClientAuth:   clientAuthType,
	}
	return
}

type ClientTLSConfig struct {
	Trusted            bool     `json:"trusted"`
	Cert               string   `json:"cert"`
	Key                string   `json:"key"`
	RootCAs            []string `json:"rootCAs"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
}

func (config *ClientTLSConfig) Load() (err error) {
	if config.Trusted {
		return
	}
	if config.RootCAs != nil && len(config.RootCAs) > 0 {
		encodedCAs := make([]string, 0, 1)
		for _, ca := range config.RootCAs {
			ca = strings.TrimSpace(ca)
			if ca == "" {
				err = fmt.Errorf("fns: one of root ca is empty")
				return
			}
			caPEM, caErr := decodeTlsSource(ca)
			if caErr != nil {
				err = fmt.Errorf("fns: read root ca failed, %v", caErr)
				return
			}
			block, _ := pem.Decode(caPEM)
			if block == nil {
				err = fmt.Errorf("fns: root ca pem is invalid")
				return
			}
			if block.Type != "CERTIFICATE" {
				err = fmt.Errorf("fns: root ca pem is not CERTIFICATE")
				return
			}
			_, parseErr := x509.ParseCertificate(block.Bytes)
			if parseErr != nil {
				err = fmt.Errorf("fns: parse root ca failed, %v", parseErr)
				return
			}
			encodedCAs = append(encodedCAs, encodeTlsSource(caPEM))
		}
		config.RootCAs = encodedCAs
	}
	cert := strings.TrimSpace(config.Cert)
	if cert == "" {
		err = fmt.Errorf("fns: client cert is empty")
		return
	}
	certPEM, certErr := decodeTlsSource(cert)
	if certErr != nil {
		err = fmt.Errorf("fns: read client cert failed, %v", certErr)
		return
	}
	key := strings.TrimSpace(config.Key)
	if key == "" {
		err = fmt.Errorf("fns: client key is empty")
		return
	}
	keyPEM, keyErr := decodeTlsSource(key)
	if keyErr != nil {
		err = fmt.Errorf("fns: read client key failed, %v", keyErr)
		return
	}
	_, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
	if certificateErr != nil {
		err = fmt.Errorf("fns: create client x509 keypair failed, %v", certificateErr)
		return
	}
	config.Cert = encodeTlsSource(certPEM)
	config.Key = encodeTlsSource(keyPEM)
	return
}

var defaultCertificateRequestConfig = &CertificateRequestConfig{
	KeyBits:            2048,
	Country:            "CN",
	Province:           "Shanghai",
	City:               "Shanghai",
	Organization:       "AACFACTORY",
	OrganizationalUnit: "Tech",
	CommonName:         "FNS",
	IPs:                nil,
	DelegationEnabled:  true,
	ExpirationMonths:   12,
}

type CertificateRequestConfig struct {
	KeyBits            int      `json:"keyBits"`
	Country            string   `json:"country"`
	Province           string   `json:"province"`
	City               string   `json:"city"`
	Organization       string   `json:"organization"`
	OrganizationalUnit string   `json:"organizationalUnit"`
	CommonName         string   `json:"commonName"`
	IPs                []string `json:"ips"`
	DelegationEnabled  bool     `json:"delegationEnabled"`
	ExpirationMonths   int      `json:"expirationMonths"`
}

type TLSConfig struct {
	// Kind
	// ACME
	// ASSC(AUTO-SELF-SIGN-CERT)
	// DEFAULT
	Kind    string           `json:"kind"`
	Server  *ServerTLSConfig `json:"server"`
	Client  *ClientTLSConfig `json:"client"`
	Options json.RawMessage  `json:"options"`
}

func (config *TLSConfig) Config() (srvTLS *tls.Config, err error) {
	config.Kind = strings.ToUpper(strings.TrimSpace(config.Kind))
	switch config.Kind {
	case "ACME":
		// TODO
		err = errors.Warning("fns: acme is unavailable")
	case "ASSC", "AUTO-SELF-SIGN-CERT":
		var csr *CertificateRequestConfig = nil
		options := config.Options
		if options == nil {
			csr = defaultCertificateRequestConfig
		} else {
			csr = &CertificateRequestConfig{}
			decodeErr := json.Unmarshal(options, csr)
			if decodeErr != nil {
				err = errors.Warning("fns: load tls config failed").WithCause(decodeErr)
				return
			}
		}
		// todo auto generate ssl, and set server and client

	default:
		if config.Server == nil {
			err = errors.Warning("fns: load tls config failed").WithCause(fmt.Errorf("no server in tls config"))
			return
		}
		srvTLS, err = config.Server.Config()
		if err != nil {
			err = errors.Warning("fns: load tls config failed").WithCause(err)
			return
		}
		if config.Client == nil {
			config.Client = &ClientTLSConfig{
				Trusted:            true,
				Cert:               "",
				Key:                "",
				RootCAs:            nil,
				InsecureSkipVerify: false,
			}
		}
		err = config.Client.Load()
	}
	return
}

func encodeTlsSource(p []byte) (s string) {
	s = fmt.Sprintf("base64:%s", base64.StdEncoding.EncodeToString(p))
	return
}

func decodeTlsSource(s string) (p []byte, err error) {
	if strings.Index(s, "base64:") == 0 {
		s = strings.TrimSpace(s[7:])
		p, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			err = fmt.Errorf("read source from %s failed, %v", s, err)
			return
		}
	} else if strings.Index(s, "http:") == 0 || strings.Index(s, "https:") == 0 {
		status, body, getErr := fasthttp.GetTimeout(make([]byte, 0, 1), s, 30*time.Second)
		if getErr != nil {
			err = fmt.Errorf("read source from %s failed, %v", s, getErr)
			return
		}
		if status != 200 {
			err = fmt.Errorf("read source from %s failed, %v", s, status)
			return
		}
		p = body
	} else {
		p, err = ioutil.ReadFile(s)
		if err != nil {
			err = fmt.Errorf("read source from %s failed, %v", s, err)
			return
		}
	}
	return
}

func getRegistrationClientTLS(env Environments) (v RegistrationClientTLS) {
	config, hasConfig := env.Config("http")
	if !hasConfig {
		return
	}
	tlsConfigRaw, hasTLS := config.Node("tls")
	if !hasTLS {
		return
	}
	tlsConfig := &TLSConfig{}
	decodeErr := tlsConfigRaw.As(tlsConfig)
	if decodeErr != nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: get client tls failed").WithCause(decodeErr)))
	}
	_, configErr := tlsConfig.Config()
	if decodeErr != nil {
		panic(fmt.Sprintf("%+v", errors.Warning("fns: get client tls failed").WithCause(configErr)))
	}
	v = RegistrationClientTLS{
		Enable:             true,
		Trusted:            tlsConfig.Client.Trusted,
		Cert:               tlsConfig.Client.Cert,
		Key:                tlsConfig.Client.Key,
		RootCAs:            tlsConfig.Client.RootCAs,
		InsecureSkipVerify: tlsConfig.Client.InsecureSkipVerify,
	}
	return
}
