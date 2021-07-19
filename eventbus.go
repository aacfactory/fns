package fns

import (
	"encoding/json"
	"fmt"
	"github.com/aacfactory/cluster"
	"github.com/aacfactory/eventbus"
	"strings"
)

// +-------------------------------------------------------------------------------------------------------------------+

type EventbusConfig struct {
	Kind   string          `json:"kind,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

var eventbusRetrievers map[string]EventbusRetriever = make(map[string]EventbusRetriever)

type EventbusRetriever func(config []byte) (bus eventbus.Eventbus, err error)

//RegisterEventbusRetriever 在支持的包里调用这个函数，如 INIT 中，在使用的时候如注入SQL驱动一样
func RegisterEventbusRetriever(name string, fn EventbusRetriever) {
	eventbusRetrievers[name] = fn
}

// +-------------------------------------------------------------------------------------------------------------------+

func newEventBus(discovery cluster.ServiceDiscovery, config EventbusConfig) (bus eventbus.Eventbus, err error) {
	kind := strings.ToLower(strings.TrimSpace(config.Kind))
	if kind == "local" {
		bus, err = newLocalEventbus(config)
	} else if kind == "tcp" {
		bus, err = newTcpEventbus(discovery, config)
	} else if kind == "nats" {
		bus, err = newNatsEventbus(discovery, config)
	} else {
		retriever, has := eventbusRetrievers[kind]
		if !has || retriever == nil {
			err = fmt.Errorf("unknown %s kind of eventbus", kind)
			return
		}
		bus, err = retriever(config.Config)
		if err != nil {
			err = fmt.Errorf("create  %s kind eventbus from retriever failed, %v", kind, err)
			return
		}
	}
	return
}

func newLocalEventbus(config EventbusConfig) (bus eventbus.Eventbus, err error) {
	option := eventbus.LocaledEventbusOption{}
	decodeErr := JsonAPI().Unmarshal(config.Config, &option)
	if decodeErr != nil {
		err = fmt.Errorf("create local event bus failed for decode config")
		return
	}
	bus = eventbus.NewEventbusWithOption(option)
	return
}

func newTcpEventbus(discovery cluster.ServiceDiscovery, config EventbusConfig) (bus eventbus.Eventbus, err error) {
	option := eventbus.ClusterEventbusOption{}
	decodeErr := JsonAPI().Unmarshal(config.Config, &option)
	if decodeErr != nil {
		err = fmt.Errorf("create tcp based event bus failed for decode config")
		return
	}
	bus, err = eventbus.NewClusterEventbus(discovery, option)
	return
}

func newNatsEventbus(discovery cluster.ServiceDiscovery, config EventbusConfig) (bus eventbus.Eventbus, err error) {
	option := eventbus.NatsEventbusOption{}
	decodeErr := JsonAPI().Unmarshal(config.Config, &option)
	if decodeErr != nil {
		err = fmt.Errorf("create nats based event bus failed for decode config")
		return
	}
	bus, err = eventbus.NewNatsEventbus(discovery, option)
	return
}
