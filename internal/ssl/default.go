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

package ssl

import (
	"crypto/tls"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"io/ioutil"
	"strings"
)

type DefaultLoaderOptions struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

func DefaultLoader(options configures.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	opt := &DefaultLoaderOptions{}
	optErr := options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(optErr)
		return
	}
	cert := strings.TrimSpace(opt.Cert)
	if cert == "" {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(fmt.Errorf("cert is undefined"))
		return
	}
	certPEM, readCertErr := ioutil.ReadFile(cert)
	if readCertErr != nil {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(readCertErr)
		return
	}
	key := strings.TrimSpace(opt.Key)
	if cert == "" {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(fmt.Errorf("key is undefined"))
		return
	}
	keyPEM, readKeyErr := ioutil.ReadFile(key)
	if readKeyErr != nil {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(readKeyErr)
		return
	}
	certificate, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
	if certificateErr != nil {
		err = errors.Warning("fns: load default kind tls config failed").WithCause(certificateErr)
		return
	}
	serverTLS = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.NoClientCert,
	}
	return
}
