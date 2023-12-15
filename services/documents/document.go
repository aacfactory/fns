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

package documents

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"sort"
)

type Document struct {
	Id        string           `json:"id"`
	Version   versions.Version `json:"version"`
	Endpoints Endpoints        `json:"endpoints"`
}

func (document *Document) Add(endpoint Endpoint) {
	if endpoint.Defined() {
		for i, ep := range document.Endpoints {
			if ep.Name == endpoint.Name {
				ep.Id = append(ep.Id, endpoint.Id...)
				document.Endpoints[i] = ep
				return
			}
		}
		endpoint.Id = []string{document.Id}
		endpoint.Version = document.Version
		document.Endpoints = append(document.Endpoints, endpoint)
		sort.Sort(document.Endpoints)
	}
}

func (document *Document) Get(name []byte) (v Endpoint) {
	for _, endpoint := range document.Endpoints {
		if bytes.Equal(name, bytex.FromString(endpoint.Name)) {
			return endpoint
		}
	}
	return Endpoint{}
}

// Documents
// version based
type Documents []Document

func (documents Documents) Len() int {
	return len(documents)
}

func (documents Documents) Less(i, j int) bool {
	return documents[i].Version.LessThan(documents[j].Version)
}

func (documents Documents) Swap(i, j int) {
	documents[i], documents[j] = documents[j], documents[i]
}

func (documents Documents) Latest() (document Document) {
	size := documents.Len()
	if documents.Len() == 0 {
		return
	}
	return documents[size-1]
}

func (documents Documents) Version(ver versions.Version) (document Document, has bool) {
	for _, d := range documents {
		if d.Version.Equals(ver) {
			document = d
			has = true
			return
		}
	}
	return
}

func (documents Documents) Add(endpoint Endpoint) Documents {
	for i, stored := range documents {
		if stored.Version.Equals(endpoint.Version) {
			stored.Add(endpoint)
			documents[i] = stored
			return documents
		}
	}
	doc := Document{
		Version:   endpoint.Version,
		Endpoints: []Endpoint{endpoint},
	}
	v := append(documents, doc)
	sort.Sort(v)
	return v
}
