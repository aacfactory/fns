/*
 * Copyright 2023 Wang Min Xiang
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
 *
 */

package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/aacfactory/afssl"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"net"
	"os"
	"strings"
	"time"
)

type SSCConfigOptions struct {
	CA                 string `json:"ca"`
	CAKEY              string `json:"caKey"`
	ClientAuth         string `json:"clientAuth"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	ExpireDays         int    `json:"expireDays"`
}

func (opt *SSCConfigOptions) ClientAuthType() tls.ClientAuthType {
	switch opt.ClientAuth {
	case "NoClientCert":
		return tls.NoClientCert
	case "RequestClientCert":
		return tls.RequestClientCert
	case "RequireAnyClientCert":
		return tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		return tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.NoClientCert
	}
}

type SSCConfig struct {
	serverTLS *tls.Config
	clientTLS *tls.Config
}

func (config *SSCConfig) Build(options configures.Config) (err error) {
	caPEM := defaultTestSSCCaPEM
	caKeyPEM := defaultTestSSCCaKeyPEM
	opt := SSCConfigOptions{}
	optErr := options.As(&opt)
	if optErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(optErr)
		return
	}
	ca := strings.TrimSpace(opt.CA)
	if ca != "" {
		caKey := strings.TrimSpace(opt.CAKEY)
		if caKey == "" {
			err = errors.Warning("fns: load ssc kind tls config failed").WithCause(fmt.Errorf("caKey is undefined"))
			return
		}
		caKeyPEM, err = os.ReadFile(caKey)
		if err != nil {
			err = errors.Warning("fns: load ssc kind tls config failed").WithCause(err)
			return
		}
		caPEM, err = os.ReadFile(ca)
		if err != nil {
			err = errors.Warning("fns: load ssc kind tls config failed").WithCause(err)
			return
		}
	}
	block, _ := pem.Decode(caPEM)
	ca0, parseCaErr := x509.ParseCertificate(block.Bytes)
	if parseCaErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(parseCaErr)
		return
	}
	sscConfig := afssl.CertificateConfig{
		Subject: &afssl.CertificatePkixName{
			Country:            ca0.Subject.Country[0],
			Province:           ca0.Subject.Province[0],
			Locality:           ca0.Subject.Locality[0],
			Organization:       ca0.Subject.Organization[0],
			OrganizationalUnit: ca0.Subject.OrganizationalUnit[0],
			CommonName:         ca0.Subject.CommonName,
		},
		IPs:      nil,
		Emails:   nil,
		DNSNames: nil,
	}
	serverCert, serverKey, createServerErr := afssl.GenerateCertificate(sscConfig, afssl.WithParent(caPEM, caKeyPEM), afssl.WithExpirationDays(int(ca0.NotAfter.Sub(time.Now()).Hours())/24))
	if createServerErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(createServerErr)
		return
	}
	cas := x509.NewCertPool()
	if !cas.AppendCertsFromPEM(caPEM) {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(fmt.Errorf("append client ca pool failed"))
		return
	}
	serverCertificate, serverCertificateErr := tls.X509KeyPair(serverCert, serverKey)
	if serverCertificateErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(serverCertificateErr)
		return
	}
	config.serverTLS = &tls.Config{
		ClientCAs:    cas,
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   opt.ClientAuthType(),
	}
	expireDays := opt.ExpireDays
	if expireDays < 1 {
		expireDays = 365
	}
	clientCert, clientKey, createClientErr := afssl.GenerateCertificate(sscConfig, afssl.WithParent(caPEM, caKeyPEM), afssl.WithExpirationDays(expireDays))
	if createClientErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(createClientErr)
		return
	}
	clientCertificate, clientCertificateErr := tls.X509KeyPair(clientCert, clientKey)
	if clientCertificateErr != nil {
		err = errors.Warning("fns: load ssc kind tls config failed").WithCause(clientCertificateErr)
		return
	}
	config.clientTLS = &tls.Config{
		RootCAs:            cas,
		Certificates:       []tls.Certificate{clientCertificate},
		InsecureSkipVerify: opt.InsecureSkipVerify,
	}
	return
}

func (config *SSCConfig) Server() (srvTLS *tls.Config, ln ListenerFunc) {

	return
}

func (config *SSCConfig) Client() (cliTLS *tls.Config, dialer Dialer) {

	return
}

func (config *SSCConfig) TLS() (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	serverTLS = config.serverTLS
	clientTLS = config.clientTLS
	return
}

func (config *SSCConfig) NewListener(inner net.Listener) (ln net.Listener) {
	ln = tls.NewListener(inner, config.serverTLS.Clone())
	return
}

var (
	defaultTestSSCCaPEM = []byte{
		45, 45, 45, 45, 45, 66, 69, 71, 73, 78, 32, 67, 69, 82, 84, 73, 70, 73, 67, 65,
		84, 69, 45, 45, 45, 45, 45, 10, 77, 73, 73, 68, 114, 84, 67, 67, 65, 112, 87, 103,
		65, 119, 73, 66, 65, 103, 73, 85, 86, 49, 77, 48, 86, 101, 57, 116, 111, 101, 89, 89,
		116, 118, 112, 76, 117, 82, 75, 55, 65, 89, 67, 113, 67, 52, 56, 119, 68, 81, 89, 74,
		75, 111, 90, 73, 104, 118, 99, 78, 65, 81, 69, 76, 10, 66, 81, 65, 119, 90, 84, 69,
		76, 77, 65, 107, 71, 65, 49, 85, 69, 66, 104, 77, 67, 81, 48, 52, 120, 69, 84, 65,
		80, 66, 103, 78, 86, 66, 65, 103, 77, 67, 70, 78, 111, 89, 87, 53, 110, 97, 71, 70,
		112, 77, 82, 69, 119, 68, 119, 89, 68, 86, 81, 81, 72, 68, 65, 104, 84, 10, 97, 71,
		70, 117, 90, 50, 104, 104, 97, 84, 69, 84, 77, 66, 69, 71, 65, 49, 85, 69, 67, 103,
		119, 75, 81, 85, 70, 68, 82, 107, 70, 68, 86, 69, 57, 83, 87, 84, 69, 78, 77, 65,
		115, 71, 65, 49, 85, 69, 67, 119, 119, 69, 86, 69, 86, 68, 83, 68, 69, 77, 77, 65,
		111, 71, 10, 65, 49, 85, 69, 65, 119, 119, 68, 82, 107, 53, 84, 77, 67, 65, 88, 68,
		84, 73, 121, 77, 68, 85, 121, 79, 68, 73, 119, 77, 84, 103, 48, 79, 70, 111, 89, 68,
		122, 73, 120, 77, 106, 73, 119, 78, 84, 65, 48, 77, 106, 65, 120, 79, 68, 81, 52, 87,
		106, 66, 108, 77, 81, 115, 119, 10, 67, 81, 89, 68, 86, 81, 81, 71, 69, 119, 74, 68,
		84, 106, 69, 82, 77, 65, 56, 71, 65, 49, 85, 69, 67, 65, 119, 73, 85, 50, 104, 104,
		98, 109, 100, 111, 89, 87, 107, 120, 69, 84, 65, 80, 66, 103, 78, 86, 66, 65, 99, 77,
		67, 70, 78, 111, 89, 87, 53, 110, 97, 71, 70, 112, 10, 77, 82, 77, 119, 69, 81, 89,
		68, 86, 81, 81, 75, 68, 65, 112, 66, 81, 85, 78, 71, 81, 85, 78, 85, 84, 49, 74,
		90, 77, 81, 48, 119, 67, 119, 89, 68, 86, 81, 81, 76, 68, 65, 82, 85, 82, 85, 78,
		73, 77, 81, 119, 119, 67, 103, 89, 68, 86, 81, 81, 68, 68, 65, 78, 71, 10, 84, 108,
		77, 119, 103, 103, 69, 105, 77, 65, 48, 71, 67, 83, 113, 71, 83, 73, 98, 51, 68, 81,
		69, 66, 65, 81, 85, 65, 65, 52, 73, 66, 68, 119, 65, 119, 103, 103, 69, 75, 65, 111,
		73, 66, 65, 81, 68, 108, 53, 79, 52, 107, 76, 115, 111, 65, 53, 121, 48, 105, 57, 108,
		110, 89, 10, 89, 107, 51, 48, 65, 100, 56, 114, 73, 79, 114, 70, 73, 77, 98, 48, 68,
		107, 122, 114, 52, 88, 111, 80, 47, 69, 43, 78, 79, 50, 106, 43, 77, 101, 101, 80, 56,
		99, 117, 55, 72, 52, 72, 103, 54, 50, 113, 114, 82, 71, 112, 107, 53, 101, 52, 107, 78,
		102, 100, 102, 73, 66, 78, 97, 10, 80, 83, 122, 73, 50, 89, 98, 101, 105, 110, 118, 52,
		82, 53, 83, 115, 87, 68, 80, 98, 70, 83, 108, 122, 106, 117, 74, 73, 111, 87, 110, 108,
		97, 74, 71, 115, 70, 108, 48, 115, 103, 51, 83, 72, 68, 121, 51, 56, 85, 66, 114, 51,
		104, 55, 100, 88, 69, 73, 99, 104, 85, 113, 81, 112, 10, 98, 98, 120, 54, 87, 77, 48,
		104, 107, 52, 122, 71, 112, 114, 48, 103, 78, 70, 54, 110, 89, 75, 51, 118, 114, 68, 69,
		109, 78, 77, 101, 114, 75, 73, 79, 87, 65, 110, 55, 99, 82, 122, 118, 89, 109, 101, 48,
		43, 97, 57, 49, 102, 108, 56, 71, 119, 120, 52, 112, 101, 51, 107, 51, 97, 10, 47, 81,
		48, 55, 57, 74, 81, 48, 106, 100, 57, 115, 51, 107, 84, 55, 116, 67, 113, 53, 107, 110,
		69, 114, 122, 122, 88, 75, 79, 77, 54, 86, 87, 100, 66, 66, 79, 110, 100, 72, 54, 115,
		47, 47, 101, 56, 118, 52, 98, 50, 65, 104, 82, 50, 50, 48, 84, 99, 76, 50, 121, 50,
		89, 74, 10, 85, 76, 102, 97, 54, 113, 57, 50, 70, 107, 103, 115, 73, 67, 116, 85, 85,
		51, 78, 111, 67, 108, 88, 108, 87, 53, 99, 80, 98, 88, 121, 74, 85, 65, 112, 51, 78,
		115, 106, 77, 120, 111, 65, 114, 72, 109, 71, 85, 99, 99, 98, 106, 119, 83, 52, 105, 114,
		121, 97, 54, 80, 84, 111, 65, 10, 86, 50, 110, 55, 65, 103, 77, 66, 65, 65, 71, 106,
		85, 122, 66, 82, 77, 66, 48, 71, 65, 49, 85, 100, 68, 103, 81, 87, 66, 66, 81, 49,
		74, 111, 67, 97, 109, 111, 113, 71, 87, 110, 70, 70, 47, 121, 82, 80, 81, 100, 55, 89,
		76, 50, 111, 65, 48, 68, 65, 102, 66, 103, 78, 86, 10, 72, 83, 77, 69, 71, 68, 65,
		87, 103, 66, 81, 49, 74, 111, 67, 97, 109, 111, 113, 71, 87, 110, 70, 70, 47, 121, 82,
		80, 81, 100, 55, 89, 76, 50, 111, 65, 48, 68, 65, 80, 66, 103, 78, 86, 72, 82, 77,
		66, 65, 102, 56, 69, 66, 84, 65, 68, 65, 81, 72, 47, 77, 65, 48, 71, 10, 67, 83,
		113, 71, 83, 73, 98, 51, 68, 81, 69, 66, 67, 119, 85, 65, 65, 52, 73, 66, 65, 81,
		65, 103, 120, 52, 115, 87, 76, 89, 112, 122, 108, 90, 105, 97, 119, 65, 119, 106, 65, 108,
		81, 108, 56, 66, 116, 82, 81, 90, 107, 43, 83, 115, 73, 120, 99, 67, 76, 110, 81, 90,
		97, 86, 10, 53, 67, 84, 77, 105, 48, 104, 90, 57, 79, 90, 53, 111, 100, 98, 51, 76,
		90, 55, 119, 48, 110, 90, 74, 97, 53, 77, 77, 108, 117, 114, 74, 106, 70, 69, 112, 49,
		82, 82, 110, 104, 69, 71, 106, 105, 71, 53, 86, 84, 105, 48, 85, 121, 108, 83, 112, 83,
		116, 43, 115, 83, 71, 56, 68, 10, 114, 57, 110, 118, 72, 108, 108, 88, 48, 88, 84, 106,
		69, 84, 69, 121, 86, 105, 103, 66, 57, 52, 105, 97, 53, 54, 74, 43, 84, 106, 70, 66,
		50, 67, 50, 66, 52, 69, 87, 101, 121, 57, 97, 54, 110, 51, 57, 102, 51, 82, 89, 86,
		77, 47, 66, 113, 76, 74, 66, 51, 101, 81, 77, 107, 10, 109, 100, 48, 114, 98, 69, 90,
		52, 106, 99, 105, 105, 111, 116, 52, 116, 114, 84, 89, 72, 65, 79, 118, 117, 71, 57, 90,
		97, 71, 71, 52, 83, 87, 107, 76, 98, 106, 65, 122, 51, 97, 66, 69, 76, 97, 76, 69,
		51, 114, 80, 71, 56, 70, 77, 100, 69, 82, 104, 43, 110, 69, 51, 75, 110, 10, 51, 54,
		54, 116, 110, 117, 86, 70, 87, 104, 106, 48, 57, 97, 85, 111, 116, 117, 86, 79, 55, 117,
		110, 67, 50, 55, 106, 53, 87, 56, 56, 84, 77, 78, 89, 83, 103, 48, 66, 116, 54, 76,
		89, 105, 57, 88, 57, 43, 107, 67, 81, 122, 101, 119, 98, 54, 73, 54, 50, 83, 51, 57,
		66, 103, 10, 75, 122, 102, 48, 67, 79, 98, 66, 108, 84, 105, 70, 90, 50, 90, 115, 99,
		102, 101, 83, 99, 68, 73, 69, 84, 73, 98, 88, 84, 120, 86, 71, 56, 107, 84, 120, 102,
		104, 68, 80, 87, 107, 102, 105, 10, 45, 45, 45, 45, 45, 69, 78, 68, 32, 67, 69, 82,
		84, 73, 70, 73, 67, 65, 84, 69, 45, 45, 45, 45, 45,
	}
	defaultTestSSCCaKeyPEM = []byte{
		45, 45, 45, 45, 45, 66, 69, 71, 73, 78, 32, 82, 83, 65, 32, 80, 82, 73, 86, 65,
		84, 69, 32, 75, 69, 89, 45, 45, 45, 45, 45, 10, 77, 73, 73, 69, 111, 119, 73, 66,
		65, 65, 75, 67, 65, 81, 69, 65, 53, 101, 84, 117, 74, 67, 55, 75, 65, 79, 99, 116,
		73, 118, 90, 90, 50, 71, 74, 78, 57, 65, 72, 102, 75, 121, 68, 113, 120, 83, 68, 71,
		57, 65, 53, 77, 54, 43, 70, 54, 68, 47, 120, 80, 106, 84, 116, 111, 10, 47, 106, 72,
		110, 106, 47, 72, 76, 117, 120, 43, 66, 52, 79, 116, 113, 113, 48, 82, 113, 90, 79, 88,
		117, 74, 68, 88, 51, 88, 121, 65, 84, 87, 106, 48, 115, 121, 78, 109, 71, 51, 111, 112,
		55, 43, 69, 101, 85, 114, 70, 103, 122, 50, 120, 85, 112, 99, 52, 55, 105, 83, 75, 70,
		112, 10, 53, 87, 105, 82, 114, 66, 90, 100, 76, 73, 78, 48, 104, 119, 56, 116, 47, 70,
		65, 97, 57, 52, 101, 51, 86, 120, 67, 72, 73, 86, 75, 107, 75, 87, 50, 56, 101, 108,
		106, 78, 73, 90, 79, 77, 120, 113, 97, 57, 73, 68, 82, 101, 112, 50, 67, 116, 55, 54,
		119, 120, 74, 106, 84, 72, 10, 113, 121, 105, 68, 108, 103, 74, 43, 51, 69, 99, 55, 50,
		74, 110, 116, 80, 109, 118, 100, 88, 53, 102, 66, 115, 77, 101, 75, 88, 116, 53, 78, 50,
		118, 48, 78, 79, 47, 83, 85, 78, 73, 51, 102, 98, 78, 53, 69, 43, 55, 81, 113, 117,
		90, 74, 120, 75, 56, 56, 49, 121, 106, 106, 79, 10, 108, 86, 110, 81, 81, 84, 112, 51,
		82, 43, 114, 80, 47, 51, 118, 76, 43, 71, 57, 103, 73, 85, 100, 116, 116, 69, 51, 67,
		57, 115, 116, 109, 67, 86, 67, 51, 50, 117, 113, 118, 100, 104, 90, 73, 76, 67, 65, 114,
		86, 70, 78, 122, 97, 65, 112, 86, 53, 86, 117, 88, 68, 50, 49, 56, 10, 105, 86, 65,
		75, 100, 122, 98, 73, 122, 77, 97, 65, 75, 120, 53, 104, 108, 72, 72, 71, 52, 56, 69,
		117, 73, 113, 56, 109, 117, 106, 48, 54, 65, 70, 100, 112, 43, 119, 73, 68, 65, 81, 65,
		66, 65, 111, 73, 66, 65, 69, 108, 100, 53, 81, 52, 82, 68, 74, 66, 55, 78, 110, 70,
		111, 10, 56, 48, 86, 87, 73, 104, 67, 85, 74, 70, 101, 77, 79, 115, 66, 77, 100, 74,
		72, 103, 109, 110, 88, 81, 48, 72, 97, 88, 105, 47, 47, 68, 106, 80, 57, 75, 104, 57,
		55, 116, 83, 74, 112, 103, 78, 76, 47, 71, 65, 90, 88, 69, 48, 76, 117, 65, 107, 90,
		53, 109, 120, 112, 112, 75, 10, 68, 48, 77, 71, 77, 79, 117, 115, 87, 66, 108, 102, 85,
		113, 55, 113, 107, 83, 122, 114, 80, 83, 108, 87, 117, 74, 76, 84, 98, 54, 51, 69, 76,
		90, 112, 122, 52, 56, 70, 113, 112, 98, 79, 87, 66, 68, 77, 121, 67, 102, 102, 121, 122,
		74, 104, 103, 98, 73, 100, 82, 107, 47, 53, 122, 10, 100, 69, 90, 119, 97, 101, 48, 86,
		116, 43, 108, 87, 81, 71, 65, 74, 83, 71, 81, 108, 115, 109, 116, 121, 78, 68, 65, 47,
		82, 97, 111, 118, 86, 120, 85, 69, 107, 109, 114, 117, 65, 85, 75, 72, 103, 108, 48, 101,
		118, 70, 79, 54, 108, 118, 84, 84, 68, 53, 106, 72, 110, 107, 98, 79, 10, 121, 51, 48,
		54, 106, 75, 113, 67, 122, 54, 101, 100, 56, 100, 49, 113, 65, 57, 102, 52, 50, 112, 51,
		48, 78, 88, 114, 113, 48, 122, 82, 97, 65, 121, 53, 111, 82, 79, 85, 120, 70, 68, 99,
		101, 50, 67, 104, 52, 47, 83, 75, 49, 51, 113, 73, 105, 70, 111, 106, 88, 51, 122, 53,
		57, 10, 117, 110, 122, 98, 120, 67, 79, 49, 85, 100, 108, 100, 98, 104, 100, 118, 99, 97,
		121, 105, 79, 72, 74, 78, 79, 50, 43, 53, 100, 67, 119, 112, 54, 50, 109, 118, 56, 112,
		101, 121, 87, 112, 103, 73, 55, 119, 72, 66, 49, 103, 80, 122, 81, 101, 79, 109, 66, 50,
		81, 51, 114, 107, 72, 112, 10, 73, 55, 67, 81, 105, 111, 69, 67, 103, 89, 69, 65, 47,
		120, 120, 70, 118, 71, 107, 85, 70, 72, 97, 67, 116, 119, 110, 81, 113, 102, 76, 72, 104,
		84, 107, 54, 54, 99, 118, 114, 119, 112, 118, 97, 82, 113, 89, 122, 100, 103, 99, 106, 82,
		116, 66, 108, 54, 101, 77, 55, 79, 78, 66, 50, 10, 120, 43, 80, 89, 56, 104, 111, 76,
		70, 109, 56, 73, 107, 107, 55, 97, 54, 85, 104, 51, 114, 120, 97, 56, 84, 54, 79, 50,
		77, 87, 82, 117, 83, 55, 110, 77, 78, 99, 74, 79, 113, 67, 84, 107, 102, 122, 100, 85,
		55, 119, 98, 65, 65, 69, 101, 74, 57, 56, 111, 113, 83, 76, 47, 69, 10, 80, 51, 99,
		75, 121, 114, 118, 118, 43, 103, 47, 55, 50, 105, 86, 52, 104, 116, 85, 99, 97, 111, 110,
		108, 47, 89, 108, 72, 87, 68, 68, 86, 103, 51, 116, 101, 114, 78, 118, 112, 77, 122, 117,
		119, 119, 110, 120, 116, 49, 108, 76, 55, 89, 77, 69, 67, 103, 89, 69, 65, 53, 114, 73,
		108, 10, 57, 106, 66, 77, 110, 50, 100, 101, 67, 51, 81, 48, 111, 108, 49, 51, 104, 77,
		112, 50, 56, 90, 110, 106, 117, 51, 104, 71, 119, 48, 115, 54, 102, 81, 71, 100, 109, 114,
		119, 97, 88, 118, 65, 54, 83, 89, 117, 121, 116, 110, 65, 87, 67, 104, 48, 75, 80, 109,
		47, 98, 72, 86, 113, 71, 10, 76, 117, 53, 102, 89, 43, 83, 88, 73, 98, 111, 103, 98,
		113, 54, 102, 97, 48, 84, 97, 56, 71, 98, 51, 121, 77, 114, 98, 73, 101, 108, 55, 113,
		115, 112, 85, 52, 67, 117, 72, 73, 99, 56, 67, 116, 97, 113, 107, 87, 66, 48, 54, 71,
		115, 70, 79, 102, 102, 55, 48, 90, 79, 122, 82, 10, 53, 98, 89, 70, 89, 79, 85, 119,
		87, 114, 108, 105, 50, 68, 66, 84, 104, 79, 104, 85, 51, 111, 104, 50, 101, 99, 69, 43,
		110, 120, 74, 116, 78, 53, 70, 90, 47, 98, 115, 67, 103, 89, 65, 99, 54, 104, 78, 116,
		87, 50, 117, 80, 78, 105, 57, 121, 108, 52, 89, 121, 47, 80, 86, 111, 10, 81, 67, 104,
		82, 80, 50, 43, 108, 83, 119, 122, 101, 88, 82, 65, 81, 72, 74, 98, 43, 43, 55, 102,
		82, 88, 112, 80, 106, 121, 74, 122, 116, 52, 119, 69, 47, 122, 51, 118, 97, 79, 120, 78,
		53, 111, 98, 53, 109, 71, 110, 83, 87, 80, 55, 108, 119, 80, 86, 110, 49, 70, 122, 68,
		53, 10, 72, 69, 72, 116, 66, 101, 122, 115, 87, 101, 73, 99, 71, 83, 86, 106, 81, 104,
		121, 89, 54, 52, 76, 84, 116, 118, 73, 55, 57, 75, 66, 70, 111, 84, 82, 122, 55, 103,
		69, 120, 69, 111, 97, 49, 72, 118, 73, 101, 78, 105, 70, 87, 89, 102, 76, 84, 88, 97,
		47, 99, 97, 119, 121, 73, 10, 76, 110, 57, 52, 107, 67, 82, 75, 84, 107, 87, 109, 104,
		88, 118, 100, 103, 117, 74, 68, 65, 81, 75, 66, 103, 71, 55, 108, 108, 72, 111, 85, 72,
		99, 70, 67, 51, 50, 75, 67, 75, 103, 115, 106, 65, 121, 70, 67, 99, 111, 115, 82, 102,
		118, 102, 80, 105, 98, 83, 51, 112, 82, 89, 83, 10, 103, 77, 67, 120, 83, 55, 118, 51,
		110, 119, 49, 98, 113, 106, 48, 112, 66, 71, 56, 52, 74, 111, 82, 57, 73, 77, 101, 114,
		106, 72, 86, 106, 65, 86, 102, 122, 117, 118, 76, 114, 108, 107, 117, 101, 104, 101, 80, 54,
		53, 89, 82, 75, 43, 122, 72, 54, 48, 102, 119, 114, 85, 78, 100, 53, 10, 67, 47, 80,
		50, 43, 75, 54, 51, 53, 73, 87, 80, 48, 104, 68, 74, 101, 47, 85, 65, 108, 56, 114,
		90, 108, 73, 73, 118, 108, 88, 98, 110, 87, 81, 54, 76, 72, 103, 78, 43, 117, 112, 86,
		83, 74, 100, 80, 117, 71, 49, 52, 101, 71, 49, 88, 77, 72, 49, 52, 52, 98, 87, 103,
		89, 10, 53, 122, 109, 104, 65, 111, 71, 66, 65, 76, 102, 68, 110, 86, 116, 107, 106, 122,
		65, 83, 112, 97, 98, 110, 112, 100, 119, 86, 122, 78, 49, 79, 53, 86, 48, 118, 48, 114,
		65, 53, 88, 112, 97, 105, 87, 82, 101, 89, 87, 113, 74, 74, 88, 89, 54, 100, 113, 87,
		51, 107, 50, 97, 57, 110, 10, 49, 107, 106, 56, 71, 106, 53, 72, 67, 81, 121, 88, 43,
		119, 43, 113, 71, 103, 121, 90, 78, 76, 101, 116, 54, 97, 76, 110, 106, 84, 71, 51, 43,
		97, 65, 76, 102, 90, 86, 97, 119, 104, 75, 68, 74, 43, 85, 112, 51, 49, 67, 73, 108,
		90, 121, 52, 116, 98, 55, 81, 74, 111, 84, 77, 10, 51, 77, 83, 106, 102, 104, 118, 51,
		114, 102, 65, 117, 114, 53, 100, 79, 116, 115, 55, 76, 53, 65, 99, 65, 98, 72, 111, 76,
		65, 77, 107, 51, 74, 114, 116, 82, 76, 55, 109, 80, 116, 108, 103, 80, 115, 90, 121, 57,
		75, 70, 89, 65, 10, 45, 45, 45, 45, 45, 69, 78, 68, 32, 82, 83, 65, 32, 80, 82,
		73, 86, 65, 84, 69, 32, 75, 69, 89, 45, 45, 45, 45, 45,
	}
)
