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
	"encoding/pem"
	"fmt"
	"github.com/aacfactory/afssl"
	"github.com/aacfactory/configuares"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/internal/ssl"
	"io/ioutil"
	"strings"
	"time"
)

type TLSLoader func(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error)

var (
	tlsLoaders = map[string]TLSLoader{"SSC": sscTLSLoader, "DEFAULT": defaultTLSLoader}
)

func RegisterTLSLoader(kind string, loader TLSLoader) {
	if kind == "" || loader == nil {
		return
	}
	_, has := tlsLoaders[kind]
	if has {
		panic(fmt.Errorf("fns: regisger tls loader failed for existed"))
	}
	tlsLoaders[kind] = loader
}

type sscTLSOptions struct {
	CA    string `json:"ca"`
	CAKEY string `json:"caKey"`
}

func sscTLSLoader(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	caPEM := ssl.DefaultTestCaPEM
	caKeyPEM := ssl.DefaultTestCaKeyPEM
	opt := &sscTLSOptions{}
	optErr := options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: tls config load options failed").WithCause(fmt.Errorf("fns: options is undefined"))
		return
	}
	ca := strings.TrimSpace(opt.CA)
	if ca != "" {
		caKey := strings.TrimSpace(opt.CAKEY)
		if caKey == "" {
			err = errors.Warning("fns: tls config load ca failed").WithCause(fmt.Errorf("fns: caKey is undefined"))
			return
		}
		caKeyPEM, err = ioutil.ReadFile(caKey)
		if err != nil {
			err = errors.Warning("fns: tls config load caKey failed").WithCause(err)
			return
		}
		caPEM, err = ioutil.ReadFile(ca)
		if err != nil {
			err = errors.Warning("fns: tls config load ca failed").WithCause(err)
			return
		}
	}
	block, _ := pem.Decode(caPEM)
	ca0, parseCaErr := x509.ParseCertificate(block.Bytes)
	if parseCaErr != nil {
		err = errors.Warning("fns: tls config load ca failed").WithCause(parseCaErr)
		return
	}
	sscConfig := afssl.CertificateConfig{
		Country:            ca0.Subject.Country[0],
		Province:           ca0.Subject.Province[0],
		City:               ca0.Subject.Locality[0],
		Organization:       ca0.Subject.Organization[0],
		OrganizationalUnit: ca0.Subject.OrganizationalUnit[0],
		CommonName:         ca0.Subject.CommonName,
		IPs:                nil,
		Emails:             nil,
		DNSNames:           nil,
	}
	serverCert, serverKey, createServerErr := afssl.GenerateCertificate(sscConfig, afssl.WithParent(caPEM, caKeyPEM), afssl.WithExpirationDays(int(ca0.NotAfter.Sub(time.Now()).Hours())/24))
	if createServerErr != nil {
		err = errors.Warning("fns: tls config create server failed").WithCause(createServerErr)
		return
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		err = errors.Warning("fns: tls config create server failed").WithCause(fmt.Errorf("append client ca pool failed"))
		return
	}
	serverCertificate, serverCertificateErr := tls.X509KeyPair(serverCert, serverKey)
	if serverCertificateErr != nil {
		err = errors.Warning("fns: tls config create server failed").WithCause(serverCertificateErr)
		return
	}
	serverTLS = &tls.Config{
		ClientCAs:    clientCAs,
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	clientCert, clientKey, createClientErr := afssl.GenerateCertificate(sscConfig, afssl.WithParent(caPEM, caKeyPEM), afssl.WithExpirationDays(int(ca0.NotAfter.Sub(time.Now()).Hours())/24))
	if createClientErr != nil {
		err = errors.Warning("fns: tls config create client failed").WithCause(createClientErr)
		return
	}
	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caPEM) {
		err = errors.Warning("fns: tls config create client failed").WithCause(fmt.Errorf("append root ca pool failed"))
		return
	}
	clientCertificate, clientCertificateErr := tls.X509KeyPair(clientCert, clientKey)
	if clientCertificateErr != nil {
		err = errors.Warning("fns: tls config create client failed").WithCause(clientCertificateErr)
		return
	}
	clientTLS = &tls.Config{
		RootCAs:            rootCAs,
		Certificates:       []tls.Certificate{clientCertificate},
		InsecureSkipVerify: true,
	}
	return
}

type defaultTLSOptions struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

func defaultTLSLoader(options configuares.Config) (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	opt := &defaultTLSOptions{}
	optErr := options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: tls config load options failed").WithCause(fmt.Errorf("fns: options is undefined"))
		return
	}
	cert := strings.TrimSpace(opt.Cert)
	if cert == "" {
		err = errors.Warning("fns: tls config load cert failed").WithCause(fmt.Errorf("fns: cert is undefined"))
		return
	}
	certPEM, readCertErr := ioutil.ReadFile(cert)
	if readCertErr != nil {
		err = errors.Warning("fns: tls config load cert failed").WithCause(readCertErr)
		return
	}
	key := strings.TrimSpace(opt.Key)
	if cert == "" {
		err = errors.Warning("fns: tls config load key failed").WithCause(fmt.Errorf("fns: key is undefined"))
		return
	}
	keyPEM, readKeyErr := ioutil.ReadFile(key)
	if readKeyErr != nil {
		err = errors.Warning("fns: tls config load key failed").WithCause(readKeyErr)
		return
	}
	certificate, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
	if certificateErr != nil {
		err = errors.Warning("fns: tls config load key failed").WithCause(certificateErr)
		return
	}
	serverTLS = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.NoClientCert,
	}
	return
}
