package clusters

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/json"
	"github.com/klauspost/compress/zlib"
	"github.com/valyala/bytebufferpool"
	"io"
)

func NewEndpointInfo(name string, internal bool, document documents.Document) (info EndpointInfo, err error) {
	info = EndpointInfo{
		Name:        name,
		Internal:    internal,
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
		info.DocumentRaw = buf.Bytes()
	}
	return
}

type EndpointInfo struct {
	Name        string `json:"name"`
	Internal    bool   `json:"internal"`
	DocumentRaw []byte `json:"document"`
}

func (info EndpointInfo) Document() (document documents.Document, err error) {
	if info.Internal || len(info.DocumentRaw) == 0 {
		return
	}
	r, rErr := zlib.NewReader(bytes.NewReader(info.DocumentRaw))
	if rErr != nil {
		err = errors.Warning("fns: endpoint info get document failed").WithCause(rErr)
		return
	}
	p, readErr := io.ReadAll(r)
	if readErr != nil {
		_ = r.Close()
		err = errors.Warning("fns: endpoint info get document failed").WithCause(readErr)
		return
	}
	_ = r.Close()
	document = documents.Document{}
	decodeErr := json.Unmarshal(p, &document)
	if decodeErr != nil {
		err = errors.Warning("fns: endpoint info get document failed").WithCause(decodeErr)
		return
	}
	return
}

type Node struct {
	Id        string           `json:"id"`
	Name      string           `json:"name"`
	Version   versions.Version `json:"version"`
	Address   string           `json:"address"`
	Endpoints []EndpointInfo   `json:"endpoints"`
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
