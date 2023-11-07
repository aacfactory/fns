package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/clusters/development"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"strings"
)

func NewDevelopment(options Options) (discovery services.Discovery, cluster Cluster, barrier barriers.Barrier, handlers []transports.MuxHandler, err error) {
	// signature
	signature := NewSignature(options.Config.Secret)
	// cluster
	if options.Config.Option == nil && len(options.Config.Option) < 2 {
		options.Config.Option = []byte{'{', '}'}
	}
	clusterConfig, clusterConfigErr := configures.NewJsonConfig(options.Config.Option)
	if clusterConfigErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterConfigErr).WithMeta("name", options.Config.Name)
		return
	}
	config := DevelopmentConfig{}
	configErr := clusterConfig.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(configErr).WithMeta("name", options.Config.Name)
		return
	}
	address := strings.TrimSpace(config.Address)
	if address == "" {
		err = errors.Warning("fns: new cluster failed").WithCause(fmt.Errorf("address in cluster config is required")).WithMeta("name", options.Config.Name)
		return
	}
	cluster = &Development{
		log:       nil,
		signature: signature,
		address:   nil,
		dialer:    options.Dialer,
		events:    make(chan NodeEvent, 8),
	}
	clusterErr := cluster.Construct(ClusterOptions{
		Log:     options.Log.With("cluster", options.Config.Name),
		Config:  clusterConfig,
		Id:      options.Id,
		Name:    options.Name,
		Version: options.Version,
		Address: fmt.Sprintf("localhost:%d", options.Port),
	})
	if clusterErr != nil {
		err = errors.Warning("fns: new cluster failed").WithCause(clusterErr).WithMeta("name", options.Config.Name)
		return
	}
	// barrier
	barrier = NewBarrier(options.Config.Barrier)
	// discovery
	discovery = development.NewDiscovery(options.Log.With("cluster", "discovery"), address, options.Dialer, signature)
	return
}

const developmentName = "dev"

type DevelopmentConfig struct {
	Address string
}

type Development struct {
	log       logs.Logger
	signature signatures.Signature
	address   []byte
	dialer    transports.Dialer
	events    chan NodeEvent
}

func (cluster *Development) Construct(options ClusterOptions) (err error) {
	cluster.log = options.Log
	config := DevelopmentConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(configErr)
		return
	}
	address := strings.TrimSpace(config.Address)
	if address == "" {
		err = errors.Warning("fns: dev cluster construct failed").WithCause(fmt.Errorf("address is required"))
		return
	}
	cluster.address = []byte(address)
	return
}

func (cluster *Development) AddEndpoint(_ EndpointInfo) {
	return
}

func (cluster *Development) Join(_ context.Context) (err error) {
	return
}

func (cluster *Development) Leave(_ context.Context) (err error) {
	close(cluster.events)
	return
}

func (cluster *Development) NodeEvents() (events <-chan NodeEvent) {
	events = cluster.events
	return
}

func (cluster *Development) Shared() (shared shareds.Shared) {
	shared = development.NewShared(cluster.dialer, cluster.address, cluster.signature)
	return
}
