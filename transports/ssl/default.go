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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/aacfactory/afssl/gmsm/cfca"
	"github.com/aacfactory/afssl/gmsm/sm2"
	"github.com/aacfactory/afssl/gmsm/smx509"
	"github.com/aacfactory/afssl/gmsm/tlcp"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"net"
	"os"
	"strings"
	"time"
)

type Keypair struct {
	Cert     string `json:"cert"`
	Key      string `json:"key"`
	Password string `json:"password"`
}

type Keypairs []Keypair

func (kps Keypairs) Certificates() (tlcps []tlcp.Certificate, standards []tls.Certificate, err error) {
	if len(kps) == 0 {
		return
	}
	for _, keypair := range kps {
		cert := strings.TrimSpace(keypair.Cert)
		key := strings.TrimSpace(keypair.Key)
		// key
		if key == "" {
			err = errors.Warning("fns: keypairs build certificates failed").WithCause(fmt.Errorf("key is undefined"))
			return
		}
		var keyPEM []byte
		if strings.IndexAny(key, "-----BEGIN") < 0 {
			keyPEM, err = os.ReadFile(key)
			if err != nil {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(err)
				return
			}
		} else {
			keyPEM = []byte(key)
		}
		keyBlock, _ := pem.Decode(keyPEM)
		if keyBlock.Type == "CFCA" {
			password := strings.TrimSpace(keypair.Password)
			if password == "" {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(fmt.Errorf("password is undefined"))
				return
			}
			pass, readPassErr := os.ReadFile(password)
			if readPassErr != nil {
				if !os.IsNotExist(readPassErr) {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(readPassErr)
					return
				}
				pass = []byte(password)
			}
			cfcaCert, cfcaKey, cfcaErr := cfca.Parse(keyPEM, pass)
			if cfcaErr != nil {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(cfcaErr)
				return
			}
			if cert != "" {
				var certPEM []byte
				if strings.IndexAny(cert, "-----BEGIN") < 0 {
					certPEM, err = os.ReadFile(cert)
					if err != nil {
						err = errors.Warning("fns: keypairs build certificates failed").WithCause(err)
						return
					}
				} else {
					certPEM = []byte(cert)
				}
				certBlock, _ := pem.Decode(certPEM)
				if certBlock == nil {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("x509: failed to decode PEM block containing certificate"))
					return
				}
				rootCert, rootCertErr := smx509.ParseCertificate(certBlock.Bytes)
				if rootCertErr != nil {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(rootCertErr)
					return
				}
				checkSignatureErr := rootCert.CheckSignature(smx509.SignatureAlgorithm(cfcaCert.SignatureAlgorithm), cfcaCert.RawTBSCertificate, cfcaCert.Signature)
				if checkSignatureErr != nil {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(checkSignatureErr)
					return
				}
			}
			certificate := tlcp.Certificate{
				Certificate: [][]byte{keyBlock.Bytes},
				PrivateKey:  cfcaKey,
				Leaf:        cfcaCert,
			}
			switch pub := cfcaCert.PublicKey.(type) {
			case *rsa.PublicKey:
				priv, ok := certificate.PrivateKey.(*rsa.PrivateKey)
				if !ok {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("tlcp: private key type does not match public key type"))
					return
				}
				if pub.N.Cmp(priv.N) != 0 {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("tlcp: private key does not match public key"))
					return
				}
			case *ecdsa.PublicKey:
				priv, ok := certificate.PrivateKey.(*sm2.PrivateKey)
				if !ok {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("tlcp: private key type does not match public key type"))
					return
				}
				if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
					err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("tlcp: private key does not match public key"))
					return
				}
			default:
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("tlcp: unknown public key algorithm"))
				return
			}
			if tlcps == nil {
				tlcps = make([]tlcp.Certificate, 0, 1)
			}
			tlcps = append(tlcps, certificate)
			continue
		}
		keyType, getKeyTypeErr := smx509.GetGMPrivateKeyType(keyBlock.Bytes)
		if getKeyTypeErr != nil {
			err = errors.Warning("fns: keypairs build certificates failed").WithCause(getKeyTypeErr)
			return
		}
		// certPEM
		var certPEM []byte
		if strings.IndexAny(cert, "-----BEGIN") < 0 {
			certPEM, err = os.ReadFile(cert)
			if err != nil {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(err)
				return
			}
		} else {
			certPEM = []byte(cert)
		}
		if keyType == smx509.SM2Key {
			certificate, certificateErr := tlcp.X509KeyPair(certPEM, keyPEM)
			if certificateErr != nil {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(certificateErr)
				return
			}
			if tlcps == nil {
				tlcps = make([]tlcp.Certificate, 0, 1)
			}
			tlcps = append(tlcps, certificate)
		} else if keyType == smx509.SM9Key {
			err = errors.Warning("fns: keypairs build certificates failed").WithCause(errors.Warning("sm9 key is unsupported"))
			return
		} else {
			certificate, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
			if certificateErr != nil {
				err = errors.Warning("fns: keypairs build certificates failed").WithCause(certificateErr)
				return
			}
			if standards == nil {
				standards = make([]tls.Certificate, 0, 1)
			}
			standards = append(standards, certificate)
		}
	}
	return
}

type ServerConfig struct {
	ClientAuth int      `json:"clientAuth"`
	Keypair    Keypairs `json:"keypair"`
}

func (config *ServerConfig) Config() (gm *tlcp.Config, standard *tls.Config, err error) {
	clientAuth := tls.ClientAuthType(config.ClientAuth)
	if clientAuth < tls.NoClientCert || clientAuth > tls.RequireAndVerifyClientCert {
		err = errors.Warning("fns: build server side tls config failed").WithCause(fmt.Errorf("clientAuth is invalid"))
		return
	}
	if len(config.Keypair) == 0 {
		err = errors.Warning("fns: build server side tls config failed").WithCause(fmt.Errorf("keypair is undefined"))
		return
	}
	tlcps, standards, certErr := config.Keypair.Certificates()
	if certErr != nil {
		err = errors.Warning("fns: build server side tls config failed").WithCause(certErr)
		return
	}
	if len(tlcps) > 0 {
		gm = &tlcp.Config{
			Certificates: tlcps,
			ClientAuth:   tlcp.ClientAuthType(clientAuth),
		}
	}
	if len(standards) > 0 {
		standard = &tls.Config{
			Certificates: standards,
			ClientAuth:   clientAuth,
		}
	}
	return
}

type ClientConfig struct {
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
	Keypair            Keypairs `json:"keypair"`
}

func (config *ClientConfig) Config() (gm *tlcp.Config, standard *tls.Config, err error) {
	if len(config.Keypair) == 0 {
		err = errors.Warning("fns: build client side tls config failed").WithCause(fmt.Errorf("keypair is undefined"))
		return
	}
	tlcps, standards, certErr := config.Keypair.Certificates()
	if certErr != nil {
		err = errors.Warning("fns: build client side tls config failed").WithCause(certErr)
		return
	}
	if len(tlcps) > 0 {
		gm = &tlcp.Config{
			Certificates:       tlcps,
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}
	if len(standards) > 0 {
		standard = &tls.Config{
			Certificates:       standards,
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}
	return
}

type DefaultConfigOptions struct {
	CA     []string      `json:"ca"`
	Server *ServerConfig `json:"server"`
	Client *ClientConfig `json:"client"`
}

func (options DefaultConfigOptions) Build() (srvGmTLS *tlcp.Config, cliGmTLS *tlcp.Config, srvStdTLS *tls.Config, cliStdTLS *tls.Config, err error) {
	if options.Server == nil {
		err = errors.Warning("fns: build default tls config failed").WithCause(fmt.Errorf("server side config is required"))
		return
	}
	srvGmTLS, srvStdTLS, err = options.Server.Config()
	if err != nil {
		err = errors.Warning("fns: build default tls config failed").WithCause(err)
		return
	}
	if options.Client != nil {
		cliGmTLS, cliStdTLS, err = options.Client.Config()
		if err != nil {
			err = errors.Warning("fns: build default tls config failed").WithCause(err)
			return
		}
	}
	var gmCAS *smx509.CertPool
	var stCAS *x509.CertPool
	if len(options.CA) > 0 {
		caPEMs := make([][]byte, 0, 1)
		for _, ca := range options.CA {
			ca = strings.TrimSpace(ca)
			if ca == "" {
				continue
			}
			var caPEM []byte
			if strings.IndexAny(ca, "-----BEGIN") < 0 {
				caPEM, err = os.ReadFile(ca)
				if err != nil {
					err = errors.Warning("fns: build default tls config failed").WithCause(err)
					return
				}
			} else {
				caPEM = []byte(ca)
			}
			caPEMs = append(caPEMs, caPEM)
		}
		if srvGmTLS != nil {
			gmCAS = smx509.NewCertPool()
			for _, caPEM := range caPEMs {
				gmCAS.AppendCertsFromPEM(caPEM)
			}
			srvGmTLS.ClientCAs = gmCAS
		}
		if srvStdTLS != nil {
			stCAS = x509.NewCertPool()
			for _, caPEM := range caPEMs {
				stCAS.AppendCertsFromPEM(caPEM)
			}
			srvStdTLS.ClientCAs = stCAS
		}
		if cliGmTLS != nil {
			cliGmTLS.RootCAs = gmCAS
		}
		if cliStdTLS != nil {
			cliStdTLS.RootCAs = stCAS
		}
	}
	return
}

func NewDefaultConfig(srv *tls.Config, cli *tls.Config, srvGM *tlcp.Config, cliGM *tlcp.Config) *DefaultConfig {
	return &DefaultConfig{
		srvStdTLS: srv,
		cliStdTLS: cli,
		srvGmTLS:  srvGM,
		cliGmTLS:  cliGM,
	}
}

type DefaultConfig struct {
	srvStdTLS *tls.Config
	cliStdTLS *tls.Config
	srvGmTLS  *tlcp.Config
	cliGmTLS  *tlcp.Config
}

func (config *DefaultConfig) Construct(options configures.Config) (err error) {
	opt := DefaultConfigOptions{}
	optErr := options.As(&opt)
	if optErr != nil {
		err = errors.Warning("fns: build default tls config failed").WithCause(optErr)
		return
	}
	config.srvGmTLS, config.cliGmTLS, config.srvStdTLS, config.cliStdTLS, err = opt.Build()
	return
}

func (config *DefaultConfig) Server() (srvTLS *tls.Config, ln ListenerFunc) {
	if config.srvGmTLS != nil {
		if config.srvStdTLS != nil {
			srvTLS = config.srvStdTLS
			ln = func(inner net.Listener) (v net.Listener) {
				v = tlcp.NewProtocolSwitcherListener(inner, config.srvGmTLS.Clone(), config.srvStdTLS.Clone())
				return
			}
			return
		}
		ln = func(inner net.Listener) (v net.Listener) {
			v = tlcp.NewListener(inner, config.srvGmTLS.Clone())
			return
		}
		return
	}
	if config.srvStdTLS != nil {
		srvTLS = config.srvStdTLS
		ln = func(inner net.Listener) (v net.Listener) {
			v = tls.NewListener(inner, config.srvStdTLS)
			return
		}
	}
	return
}

func (config *DefaultConfig) Client() (cliTLS *tls.Config, dialer Dialer) {
	if config.cliStdTLS != nil {
		cliTLS = config.cliStdTLS
		return
	}
	if config.cliGmTLS != nil {
		if config.cliStdTLS != nil {
			cliTLS = config.cliStdTLS
		}
		nd := &net.Dialer{
			Timeout:        30 * time.Second,
			Deadline:       time.Time{},
			LocalAddr:      nil,
			FallbackDelay:  0,
			KeepAlive:      60 * time.Second,
			Resolver:       nil,
			Control:        nil,
			ControlContext: nil,
		}
		dialer = &tlcp.Dialer{NetDialer: nd, Config: config.cliGmTLS}
		return
	}
	return
}
