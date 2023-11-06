package development

import (
	"bytes"
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
)

const (
	Name = "dev"
)

func New(secret string, address string, dialer transports.Dialer) (clusters.Cluster, error) {
	address = strings.TrimSpace(address)
	if len(address) == 0 {
		return nil, errors.Warning("fns: new development cluster failed, address is required")
	}
	return &Cluster{
		signature: signatures.HMAC([]byte(secret)),
		address:   []byte(address),
		dialer:    dialer,
		events:    make(chan clusters.NodeEvent, 1024),
	}, nil
}

func Register(secret string, address string, dialer transports.Transport) (err error) {
	c, cErr := New(secret, address, dialer)
	if cErr != nil {
		err = cErr
		return
	}
	clusters.RegisterCluster(Name, c)
	return nil
}

type Cluster struct {
	log       logs.Logger
	signature signatures.Signature
	address   []byte
	dialer    transports.Dialer
	events    chan clusters.NodeEvent
}

func (cluster *Cluster) Construct(options clusters.ClusterOptions) (err error) {
	cluster.log = options.Log
	return
}

func (cluster *Cluster) Join(_ context.Context, _ clusters.Node) (err error) {
	// todo get nodes from remote and send into events
	return
}

func (cluster *Cluster) Leave(_ context.Context) (err error) {
	close(cluster.events)
	return
}

func (cluster *Cluster) NodeEvents() (events <-chan clusters.NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Cluster) Shared() (shared shareds.Shared) {
	shared = NewShared(cluster.dialer, cluster.address, cluster.signature)
	return
}

func (cluster *Cluster) Barrier() (barrier barriers.Barrier) {
	barrier = NewBarrier(cluster.dialer, cluster.address, cluster.signature)
	return
}

// +-------------------------------------------------------------------------------------------------------------------+

var (
	nodesContentType = bytex.FromString("application/json+dev+nodes")
	nodesPathPrefix  = []byte("/nodes/")
)

func NewNodesHandler(secret string, events <-chan clusters.NodeEvent) transports.MuxHandler {
	return &NodesHandler{
		signature: signatures.HMAC([]byte(secret)),
		events:    events,
	}
}

type NodesHandler struct {
	log       logs.Logger
	signature signatures.Signature
	events    <-chan clusters.NodeEvent
}

func (handler *NodesHandler) Name() string {
	return "development_nodes"
}

func (handler *NodesHandler) Construct(options transports.MuxHandlerOptions) error {
	handler.log = options.Log
	return nil
}

func (handler *NodesHandler) Match(method []byte, path []byte, header transports.Header) bool {
	ok := bytes.Equal(method, methodPost) &&
		len(bytes.Split(path, slashBytes)) == 3 && bytes.LastIndex(path, nodesPathPrefix) == 0 &&
		len(header.Get(bytex.FromString(transports.SignatureHeaderName))) != 0 &&
		bytes.Equal(header.Get(bytex.FromString(transports.ContentTypeHeaderName)), nodesContentType)
	return ok
}

func (handler *NodesHandler) Handle(w transports.ResponseWriter, r transports.Request) {
	//TODO implement me
	panic("implement me")
}
