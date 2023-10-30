package clusters

import (
	"context"
	"github.com/aacfactory/fns/services"
)

type Registrations struct {
	node *Node
}

// Add
// 在services.Add后立即调用
func (rs *Registrations) Add(name string, internal bool, listenable bool) {
	rs.node.Endpoints = append(rs.node.Endpoints, EndpointInfo{
		Name:       name,
		Internal:   internal,
		Listenable: listenable,
	})
	return
}

func (rs *Registrations) Get(ctx context.Context, service []byte, options ...services.EndpointGetOption) (endpoint services.Endpoint, has bool) {

	return
}
