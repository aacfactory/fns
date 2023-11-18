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
)

func NewService(name string, internal bool, functions services.FnInfos, document documents.Endpoint) (service Service, err error) {
	service = Service{
		Name:        name,
		Internal:    internal,
		Functions:   functions,
		DocumentRaw: nil,
	}
	if !internal && !document.IsEmpty() {
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
	if service.Internal || len(service.DocumentRaw) == 0 {
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
	Name     string           `json:"name"`
	Version  versions.Version `json:"version"`
	Address  string           `json:"address"`
	Services []Service        `json:"services"`
}

const (
	Add    = NodeEventKind(1)
	Remove = NodeEventKind(2)
)

type NodeEventKind int

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
