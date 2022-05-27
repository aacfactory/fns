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
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Discovery interface {
	Register(registrations []*Registration) (err error)
	Deregister(registrations []*Registration) (err error)
	GetRegistrations(name string) (registrations *Registrations, err errors.CodeError)
	GetRegistration(name string, registrationId string) (registration *Registration, err errors.CodeError)
	Close() (err error)
}

var discoveryBuilderMap = make(map[string]DiscoveryBuilder)

type DiscoveryBuilder func(env Environments) (discovery Discovery, err error)

func RegisterDiscoveryBuilder(kind string, retriever DiscoveryBuilder) {
	discoveryBuilderMap[kind] = retriever
}

func newDiscovery(env Environments) (discovery Discovery, err error) {
	config, hasConfig := env.Config("discovery")
	if !hasConfig {
		return
	}
	kind := ""
	hasKind, getKindErr := config.Get("kind", &kind)
	if getKindErr != nil {
		err = fmt.Errorf("fns: create discovery failed for there is no kind in discovery config node")
		return
	}
	kind = strings.TrimSpace(kind)
	if !hasKind || kind == "" {
		err = fmt.Errorf("fns: create discovery failed for kind in discovery config node is undefined")
		return
	}
	builder, hasBuilder := discoveryBuilderMap[kind]
	if !hasBuilder {
		err = fmt.Errorf("fns: create discovery failed for there is no %s kind discovery builder", kind)
		return
	}
	discovery, err = builder(env)
	if err != nil {
		err = fmt.Errorf("fns: create discovery failed, %v", err)
		return
	}
	return
}

type RegistrationClientTLS struct {
	Enable             bool     `json:"enable"`
	Trusted            bool     `json:"trusted"`
	Cert               string   `json:"cert"`
	Key                string   `json:"key"`
	RootCAs            []string `json:"rootCAs"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
}

func (config *RegistrationClientTLS) Config() (v *tls.Config, err error) {
	if !config.Enable || config.Trusted {
		return
	}
	var rootCAs *x509.CertPool
	if config.RootCAs != nil && len(config.RootCAs) > 0 {
		rootCAs = x509.NewCertPool()
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
			if ok := rootCAs.AppendCertsFromPEM(caPEM); !ok {
				err = fmt.Errorf("fns: append root ca cert failed")
				return
			}
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
	certificate, certificateErr := tls.X509KeyPair(certPEM, keyPEM)
	if certificateErr != nil {
		err = fmt.Errorf("fns: create client x509 keypair failed, %v", certificateErr)
		return
	}
	v = &tls.Config{
		Certificates:       []tls.Certificate{certificate},
		RootCAs:            rootCAs,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}
	return
}

type Registration struct {
	Id                string                `json:"id"`
	Name              string                `json:"name"`
	Internal          bool                  `json:"internal"`
	Address           string                `json:"address"`
	HttpVersion       string                `json:"httpVersion"`
	ClientTLS         RegistrationClientTLS `json:"clientTLS"`
	Reversion         int64                 `json:"-"`
	once              sync.Once
	serviceProxy      ServiceProxy
	lastAccessSeconds int64
	checkHealthFailed int64
}

func (r *Registration) Close() {
	if r.serviceProxy != nil {
		r.serviceProxy.Close()
	}
	return
}

func (r *Registration) Key() (key string) {
	key = r.Id
	return
}

func (r *Registration) proxy() (proxy ServiceProxy) {
	r.once.Do(func() {
		r.serviceProxy = newServiceProxy(r)
	})
	proxy = r.serviceProxy
	return
}

func (r *Registration) CheckHealth() (ok bool) {
	now := time.Now().UnixMicro() / 1000
	lastAccessSeconds := atomic.LoadInt64(&r.lastAccessSeconds)
	if now-lastAccessSeconds > 300 {
		atomic.StoreInt64(&r.lastAccessSeconds, now)
		if r.proxy().Available() {
			atomic.StoreInt64(&r.checkHealthFailed, 0)
		} else {
			atomic.StoreInt64(&r.checkHealthFailed, 1)
		}
		return
	}
	ok = atomic.LoadInt64(&r.checkHealthFailed) == 0
	return
}

func NewRegistrations() (registrations *Registrations) {
	registrations = &Registrations{
		r: commons.NewRing(),
	}
	return
}

type Registrations struct {
	r *commons.Ring
}

func (r *Registrations) Next() (v *Registration, has bool) {
	p := r.r.Next()
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}

func (r *Registrations) Append(v *Registration) {
	r.r.Append(v)
	return
}

func (r *Registrations) Remove(v *Registration) {
	r.r.Remove(v)
	return
}

func (r *Registrations) Size() (size int) {
	size = r.r.Size()
	return
}

func (r *Registrations) Get(id string) (v *Registration, has bool) {
	if id == "" {
		return
	}
	p := r.r.Get(id)
	if p == nil {
		return
	}
	v, has = p.(*Registration)
	return
}
