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

package clusters

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/json"
	"github.com/klauspost/compress/zlib"
	"github.com/valyala/bytebufferpool"
	"io"
	"slices"
	"sort"
	"strings"
)

func NewService(name string, internal bool, functions services.FnInfos, document documents.Endpoint) (service Service, err error) {
	service = Service{
		Name:        name,
		Internal:    internal,
		Functions:   functions,
		DocumentRaw: nil,
	}
	if document.Defined() {
		p, encodeErr := json.Marshal(document)
		if encodeErr != nil {
			err = errors.Warning("fns: new endpoint info failed").WithCause(encodeErr)
			return
		}
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		w, wErr := zlib.NewWriterLevel(buf, zlib.BestCompression)
		if wErr != nil {
			err = errors.Warning("fns: new endpoint info failed").WithCause(wErr)
			return
		}
		_, _ = w.Write(p)
		_ = w.Close()
		service.DocumentRaw = buf.Bytes()
	}
	return
}

type Service struct {
	Name        string           `json:"name"`
	Internal    bool             `json:"internal"`
	Functions   services.FnInfos `json:"functions"`
	DocumentRaw []byte           `json:"document"`
}

func (service Service) Document() (document documents.Endpoint, err error) {
	if len(service.DocumentRaw) == 0 {
		return
	}
	r, rErr := zlib.NewReader(bytes.NewReader(service.DocumentRaw))
	if rErr != nil {
		err = errors.Warning("fns: service get document failed").WithCause(rErr)
		return
	}
	p, readErr := io.ReadAll(r)
	if readErr != nil {
		_ = r.Close()
		err = errors.Warning("fns: service get document failed").WithCause(readErr)
		return
	}
	_ = r.Close()
	document = documents.Endpoint{}
	decodeErr := json.Unmarshal(p, &document)
	if decodeErr != nil {
		err = errors.Warning("fns: service get document failed").WithCause(decodeErr)
		return
	}
	return
}

type Node struct {
	Id       string           `json:"id"`
	Version  versions.Version `json:"version"`
	Address  string           `json:"address"`
	Services []Service        `json:"services"`
}

const (
	Add    = NodeEventKind(1)
	Remove = NodeEventKind(2)
)

type NodeEventKind int

func (kind NodeEventKind) String() string {
	switch kind {
	case Add:
		return "add"
	case Remove:
		return "evict"
	default:
		return "unknown"
	}
}

type NodeEvent struct {
	Kind NodeEventKind
	Node Node
}

type Nodes []Node

func (nodes Nodes) Len() int {
	return len(nodes)
}

func (nodes Nodes) Less(i, j int) bool {
	return nodes[i].Id < nodes[j].Id
}

func (nodes Nodes) Swap(i, j int) {
	nodes[i], nodes[j] = nodes[j], nodes[i]
	return
}

func (nodes Nodes) Add(node Node) Nodes {
	n := append(nodes, node)
	sort.Sort(n)
	return n
}

func (nodes Nodes) Remove(node Node) Nodes {
	idx, found := slices.BinarySearchFunc(nodes, node, func(x Node, j Node) int {
		return strings.Compare(x.Id, j.Id)
	})
	if found {
		return append(nodes[:idx], nodes[idx+1:]...)
	}
	return nodes
}

func (nodes Nodes) Difference(olds Nodes) (events []NodeEvent) {
	events = make([]NodeEvent, 0, 1)
	// remove
	for _, old := range olds {
		_, found := slices.BinarySearchFunc(nodes, old, func(x Node, j Node) int {
			return strings.Compare(x.Id, j.Id)
		})
		if !found {
			events = append(events, NodeEvent{
				Kind: Remove,
				Node: old,
			})
		}
	}
	// add
	for _, node := range nodes {
		_, found := slices.BinarySearchFunc(olds, node, func(x Node, j Node) int {
			return strings.Compare(x.Id, j.Id)
		})
		if !found {
			events = append(events, NodeEvent{
				Kind: Add,
				Node: node,
			})
		}
	}
	return
}

func MapEndpointInfosToNodes(infos services.EndpointInfos) (nodes Nodes) {
	nodes = make(Nodes, 0, 1)
	for _, info := range infos {
		service, serviceErr := NewService(info.Name, info.Internal, info.Functions, info.Document)
		if serviceErr != nil {
			continue
		}
		exist := false
		for i, node := range nodes {
			if node.Id == info.Id {
				node.Services = append(node.Services, service)
				nodes[i] = node
				exist = true
				break
			}
		}
		if exist {
			continue
		}
		nodes = nodes.Add(Node{
			Id:       info.Id,
			Version:  info.Version,
			Address:  info.Address,
			Services: []Service{service},
		})
	}
	sort.Sort(nodes)
	return
}
