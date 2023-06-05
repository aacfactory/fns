package ssl

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
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
)

type GMCConfigOptions struct {
	Cert     string                `json:"cert"`
	Key      string                `json:"key"`
	Password string                `json:"password"`
	Switch   *DefaultConfigOptions `json:"switch"`
}

type GMConfig struct {
	serverTLS *tlcp.Config
	switchTLS *tls.Config
}

func (config *GMConfig) Build(options configures.Config) (err error) {
	opt := &GMCConfigOptions{}
	optErr := options.As(opt)
	if optErr != nil {
		err = errors.Warning("fns: build gm tls config failed").WithCause(optErr)
		return
	}
	cert := strings.TrimSpace(opt.Cert)
	if cert == "" {
		err = errors.Warning("fns: build gm tls config failed").WithCause(fmt.Errorf("cert is undefined"))
		return
	}
	certPEM, readCertErr := os.ReadFile(cert)
	if readCertErr != nil {
		err = errors.Warning("fns: build gm tls config failed").WithCause(readCertErr)
		return
	}
	key := strings.TrimSpace(opt.Key)
	if key == "" {
		err = errors.Warning("fns: build gm tls config failed").WithCause(fmt.Errorf("key is undefined"))
		return
	}
	keyPEM, readKeyErr := os.ReadFile(key)
	if readKeyErr != nil {
		err = errors.Warning("fns: build gm tls config failed").WithCause(readKeyErr)
		return
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock.Type == "CFCA KEY" {
		passFilePath := strings.TrimSpace(opt.Password)
		if passFilePath == "" {
			err = errors.Warning("fns: build gm tls config failed").WithCause(fmt.Errorf("pass is undefined"))
			return
		}
		pass, readPassErr := os.ReadFile(passFilePath)
		if readPassErr != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(readPassErr)
			return
		}
		cfcaCertificate, privateKey, cfcaErr := cfca.Parse(keyPEM, pass)
		if cfcaErr != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(cfcaErr)
			return
		}
		certBlock, _ := pem.Decode(certPEM)
		if certBlock == nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("x509: failed to decode PEM block containing certificate"))
			return
		}
		certificate, certificateErr := smx509.ParseCertificate(certBlock.Bytes)
		if certificateErr != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(certificateErr)
			return
		}
		checkSignatureErr := certificate.CheckSignature(smx509.SignatureAlgorithm(cfcaCertificate.SignatureAlgorithm), cfcaCertificate.RawTBSCertificate, cfcaCertificate.Signature)
		if checkSignatureErr != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(checkSignatureErr)
			return
		}
		cert0 := tlcp.Certificate{
			Certificate: [][]byte{certPEM},
			PrivateKey:  privateKey,
			Leaf:        certificate,
		}
		switch pub := certificate.PublicKey.(type) {
		case *rsa.PublicKey:
			priv, ok := cert0.PrivateKey.(*rsa.PrivateKey)
			if !ok {
				err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("tlcp: private key type does not match public key type"))
				return
			}
			if pub.N.Cmp(priv.N) != 0 {
				err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("tlcp: private key does not match public key"))
				return
			}
		case *ecdsa.PublicKey:
			priv, ok := cert0.PrivateKey.(*sm2.PrivateKey)
			if !ok {
				err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("tlcp: private key type does not match public key type"))
				return
			}
			if pub.X.Cmp(priv.X) != 0 || pub.Y.Cmp(priv.Y) != 0 {
				err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("tlcp: private key does not match public key"))
				return
			}
		default:
			err = errors.Warning("fns: build gm tls config failed").WithCause(errors.Warning("tlcp: unknown public key algorithm"))
			return
		}
		config.serverTLS = &tlcp.Config{
			Certificates: []tlcp.Certificate{cert0},
			ClientAuth:   tlcp.NoClientCert,
		}
	} else {
		certificate, certificateErr := tlcp.X509KeyPair(certPEM, keyPEM)
		if certificateErr != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(certificateErr)
			return
		}
		config.serverTLS = &tlcp.Config{
			Certificates: []tlcp.Certificate{certificate},
			ClientAuth:   tlcp.NoClientCert,
		}
	}
	if opt.Switch != nil {
		config.switchTLS, err = opt.Switch.Build()
		if err != nil {
			err = errors.Warning("fns: build gm tls config failed").WithCause(err)
			return
		}
	}
	return
}

func (config *GMConfig) TLS() (serverTLS *tls.Config, clientTLS *tls.Config, err error) {
	err = errors.Warning("fns: it is gm tls")
	return
}

func (config *GMConfig) NewListener(inner net.Listener) (ln net.Listener) {
	if config.switchTLS != nil {
		ln = tlcp.NewProtocolSwitcherListener(inner, config.serverTLS, config.switchTLS)
	} else {
		ln = tlcp.NewListener(inner, config.serverTLS)
	}
	return
}
