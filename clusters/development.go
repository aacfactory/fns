package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/clusters/proxy"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
	"time"
)

const developmentName = "dev"

type DevelopmentConfig struct {
	ProxyAddr string `json:"proxyAddr"`
}

func NewDevelopment(dialer transports.Dialer, signature signatures.Signature) Cluster {
	return &Development{
		log:       nil,
		signature: signature,
		address:   nil,
		dialer:    dialer,
		client:    nil,
		events:    make(chan NodeEvent, 64),
		closeFn:   nil,
		nodes:     make(Nodes, 0, 1),
	}
}

type Development struct {
	log       logs.Logger
	signature signatures.Signature
	address   []byte
	dialer    transports.Dialer
	client    transports.Client
	events    chan NodeEvent
	closeFn   context.CancelFunc
	nodes     Nodes
}

func (cluster *Development) Construct(options ClusterOptions) (err error) {
	cluster.log = options.Log
	config := DevelopmentConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(configErr)
		return
	}
	address := strings.TrimSpace(config.ProxyAddr)
	if address == "" {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(fmt.Errorf("address is required"))
		return
	}
	cluster.address = []byte(address)
	cluster.client, err = cluster.dialer.Dial(cluster.address)
	if err != nil {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(err)
		return
	}
	return
}

func (cluster *Development) AddService(_ Service) {
	return
}

func (cluster *Development) Join(ctx context.Context) (err error) {
	ctx, cluster.closeFn = context.WithCancel(ctx)
	go cluster.watching(ctx)
	return
}

func (cluster *Development) Leave(_ context.Context) (err error) {
	cluster.closeFn()
	close(cluster.events)
	return
}

func (cluster *Development) NodeEvents() (events <-chan NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Development) Shared() (shared shareds.Shared) {
	shared = proxy.NewShared(cluster.client, cluster.signature)
	return
}

func (cluster *Development) watching(ctx context.Context) {
	stop := false
	timer := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			stop = true
			break
		case <-timer.C:
			nodes := cluster.fetchNodes(ctx)
			events := nodes.Difference(cluster.nodes)
			for _, event := range events {
				cluster.events <- event
			}
			cluster.nodes = nodes
			break
		}
		if stop {
			timer.Stop()
			break
		}
	}
}

func (cluster *Development) fetchNodes(ctx context.Context) (nodes Nodes) {
	infos, infosErr := proxy.FetchEndpointInfos(ctx, cluster.client, cluster.signature)
	if infosErr != nil {
		if cluster.log.WarnEnabled() {
			cluster.log.Warn().Cause(infosErr).With("cluster", developmentName).Message("fns: fetch endpoint infos failed")
		}
		return
	}
	nodes = MapEndpointInfosToNodes(infos)
	return
}
