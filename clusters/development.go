package clusters

import (
	"context"
	"fmt"
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
