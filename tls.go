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
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
)


// todo: move into app config

// ClientTLS
// Fn的客户端TLS信息
type ClientTLS struct {
	// Enable
	// 是否起效
	Enable bool `json:"enable"`
	// RootCA
	// 根证书的PEM内容
	RootCA []byte `json:"rootCa"`
	// Cert
	// Cert的PEM内容
	Cert []byte `json:"cert"`
	// Key
	// Key的PEM内容
	Key []byte `json:"key"`
	// Insecure
	// 是否跳过验证
	Insecure bool `json:"insecure"`
}

// Config
// 构建*tls.Config
func (c *ClientTLS) Config() (config *tls.Config, err error) {
	if !c.Enable {
		err = fmt.Errorf("generate client tls config failed, tls not enabled")
		return
	}
	if c.Cert == nil || len(c.Cert) == 0 || c.Key == nil || len(c.Key) == 0 {
		err = fmt.Errorf("generate client tls config failed, key is empty")
		return
	}

	certificate, certificateErr := tls.X509KeyPair(c.Cert, c.Key)
	if certificateErr != nil {
		err = fmt.Errorf("generate client tls config failed, %v", certificateErr)
		return
	}

	config = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		InsecureSkipVerify: c.Insecure,
	}

	if c.RootCA != nil && len(c.RootCA) > 0 {
		cas := x509.NewCertPool()
		ok := cas.AppendCertsFromPEM(c.RootCA)
		if !ok {
			err = fmt.Errorf("generate client tls config failed, append root ca failed")
			return
		}
		config.RootCAs = cas
	}

	return
}

type ServerTLS struct {
	Enable                     bool      `json:"enable"`
	RequireAndVerifyClientAuth bool      `json:"requireAndVerifyClientAuth"`
	CA                         []byte    `json:"ca"`
	Cert                       []byte    `json:"cert"`
	Key                        []byte    `json:"key"`
	Client                     ClientTLS `json:"client,omitempty"`
}

func (s *ServerTLS) Config() (config *tls.Config, err error) {
	if !s.Enable {
		err = fmt.Errorf("generate server tls config failed, tls not enabled")
		return
	}
	if s.Cert == nil || len(s.Cert) == 0 || s.Key == nil || len(s.Key) == 0 {
		err = fmt.Errorf("generate server tls config failed, key is empty")
		return
	}

	certificate, certificateErr := tls.X509KeyPair(s.Cert, s.Key)
	if certificateErr != nil {
		err = fmt.Errorf("generate server tls config failed, %v", certificateErr)
		return
	}

	config = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
		ClientAuth:   tls.NoClientCert,
	}
	if s.RequireAndVerifyClientAuth {
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if s.CA != nil && len(s.CA) > 0 {
		cas := x509.NewCertPool()
		ok := cas.AppendCertsFromPEM(s.CA)
		if !ok {
			err = fmt.Errorf("generate server tls config failed, append ca failed")
			return
		}
		config.ClientCAs = cas
	}

	return
}
