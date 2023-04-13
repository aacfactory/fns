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

package ssc

import (
	"fmt"
	"github.com/aacfactory/afssl"
	"os"
	"path/filepath"
)

func generate(cn string, full bool, outputDir string) (err error) {
	ca, caKey, caErr := create(cn, nil, nil)
	if caErr != nil {
		return
	}
	var serverCrt, serverKey []byte
	var clientCrt, clientKey []byte
	if full {
		serverCrt, serverKey, err = create(cn, ca, caKey)
		if err != nil {
			return
		}
		clientCrt, clientKey, err = create(cn, ca, caKey)
		if err != nil {
			return
		}
	}
	err = os.WriteFile(filepath.Join(outputDir, "ca.crt"), ca, 0644)
	if err != nil {
		err = fmt.Errorf("fnc: create ssc failed, %v", err)
		return
	}
	err = os.WriteFile(filepath.Join(outputDir, "ca.key"), caKey, 0644)
	if err != nil {
		err = fmt.Errorf("fnc: create ssc failed, %v", err)
		return
	}
	if full {
		err = os.WriteFile(filepath.Join(outputDir, "server.crt"), serverCrt, 0644)
		if err != nil {
			err = fmt.Errorf("fnc: create ssc failed, %v", err)
			return
		}
		err = os.WriteFile(filepath.Join(outputDir, "server.key"), serverKey, 0644)
		if err != nil {
			err = fmt.Errorf("fnc: create ssc failed, %v", err)
			return
		}
		err = os.WriteFile(filepath.Join(outputDir, "client.crt"), clientCrt, 0644)
		if err != nil {
			err = fmt.Errorf("fnc: create ssc failed, %v", err)
			return
		}
		err = os.WriteFile(filepath.Join(outputDir, "client.key"), clientKey, 0644)
		if err != nil {
			err = fmt.Errorf("fnc: create ssc failed, %v", err)
			return
		}
	}
	return
}

func create(cn string, ca []byte, caKey []byte) (cert []byte, key []byte, err error) {
	config := afssl.CertificateConfig{
		Country:            "",
		Province:           "",
		City:               "",
		Organization:       "",
		OrganizationalUnit: "",
		CommonName:         cn,
		IPs:                nil,
		Emails:             nil,
		DNSNames:           nil,
	}
	if ca == nil || len(ca) == 0 {
		// ca
		cert, key, err = afssl.GenerateCertificate(config, afssl.CA())
		if err != nil {
			err = fmt.Errorf("fnc: create ssc failed")
			return
		}
		return
	}
	cert, key, err = afssl.GenerateCertificate(config, afssl.WithParent(ca, caKey))
	if err != nil {
		err = fmt.Errorf("fnc: create ssc failed")
		return
	}
	return
}
