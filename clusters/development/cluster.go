package development

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters"
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
	return
}

func (cluster *Cluster) Leave(_ context.Context) (err error) {
	return
}

func (cluster *Cluster) NodeEvents() (events <-chan clusters.NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Cluster) Shared() (shared shareds.Shared) {
	// use dialer
	return
}

func (cluster *Cluster) Barrier() (barrier barriers.Barrier) {
	// use dialer
	return
}
