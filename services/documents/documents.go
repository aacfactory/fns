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

package documents

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"sort"
)

type NameSortedDocuments []Document

func (documents NameSortedDocuments) Len() int {
	return len(documents)
}

func (documents NameSortedDocuments) Less(i, j int) bool {
	return documents[i].Name < documents[j].Name
}

func (documents NameSortedDocuments) Swap(i, j int) {
	documents[i], documents[j] = documents[j], documents[i]
	return
}

type VersionSortedDocuments []Documents

func (documents VersionSortedDocuments) Len() int {
	return len(documents)
}

func (documents VersionSortedDocuments) Less(i, j int) bool {
	return documents[i].Version.LessThan(documents[j].Version)
}

func (documents VersionSortedDocuments) Swap(i, j int) {
	documents[i], documents[j] = documents[j], documents[i]
}

func (documents VersionSortedDocuments) Add(id []byte, doc Document) VersionSortedDocuments {
	for _, document := range documents {
		if document.Id == bytex.ToString(id) {
			document.Add(doc)
			return documents
		}
	}
	document := NewDocuments(id, doc.Version)
	document.Add(doc)
	n := append(documents, document)
	sort.Sort(n)
	return n
}

func NewDocuments(id []byte, version versions.Version) Documents {
	return Documents{
		Id:        string(id),
		Version:   version,
		Endpoints: make(NameSortedDocuments, 0, 1),
	}
}

type Documents struct {
	Id        string              `json:"id"`
	Version   versions.Version    `json:"version"`
	Endpoints NameSortedDocuments `json:"endpoints"`
}

func (documents *Documents) Add(doc Document) {
	if doc.IsEmpty() {
		return
	}
	documents.Endpoints = append(documents.Endpoints, doc)
	sort.Sort(documents.Endpoints)
}

func (documents *Documents) Get(name []byte) (v Document) {
	for _, endpoint := range documents.Endpoints {
		if bytes.Equal(name, bytex.FromString(endpoint.Name)) {
			return endpoint
		}
	}
	return Document{}
}
