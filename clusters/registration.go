package clusters

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/versions"
	"sort"
	"sync/atomic"
)

type Registration struct {
	name   string
	length int
	pos    uint64
	values Endpoints
}

func (registration *Registration) Add(endpoint *Endpoint) {
	_, exist := registration.Get(bytex.FromString(endpoint.id))
	if exist {
		return
	}
	registration.values = append(registration.values, endpoint)
	sort.Sort(registration.values)
	registration.length = len(registration.values)
}

func (registration *Registration) Remove(id string) {
	n := -1
	for i, value := range registration.values {
		if value.id == id {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	registration.values = append(registration.values[:n], registration.values[n+1:]...)
	registration.length = len(registration.values)
}

func (registration *Registration) Get(id []byte) (endpoint *Endpoint, has bool) {
	if registration.length == 0 {
		return
	}
	idString := bytex.ToString(id)
	for _, value := range registration.values {
		if value.Running() {
			if value.id == idString {
				endpoint = value
				has = true
				break
			}
		}
	}
	return
}

func (registration *Registration) Range(interval versions.Interval) (endpoint *Endpoint, has bool) {
	if registration.length == 0 {
		return
	}
	targets := make([]*Endpoint, 0, 1)
	for _, value := range registration.values {
		if value.Running() {
			if interval.Accept(value.version) {
				targets = append(targets, value)
			}
		}
	}
	n := uint64(len(targets))
	pos := int(atomic.AddUint64(&registration.pos, 1) % n)
	endpoint = targets[pos]
	has = true
	return
}

func (registration *Registration) MaxOne() (endpoint *Endpoint, has bool) {
	if registration.length == 0 {
		return
	}
	targets := make([]*Endpoint, 0, 1)
	maxed := registration.values[registration.length-1]
	if maxed.Running() {
		targets = append(targets, maxed)
	}
	for i := registration.length - 2; i > -1; i-- {
		prev := registration.values[i]
		if prev.Running() {
			if prev.version.Equals(maxed.version) {
				targets = append(targets, prev)
				continue
			}
		}
		break
	}
	n := uint64(len(targets))
	pos := int(atomic.AddUint64(&registration.pos, 1) % n)
	endpoint = targets[pos]
	has = true
	return
}

type Registrations []*Registration

func (registrations Registrations) Get(name []byte) (v *Registration, has bool) {
	ns := bytex.ToString(name)
	for _, registration := range registrations {
		if registration.length > 0 && registration.name == ns {
			v = registration
			has = true
			break
		}
	}
	return
}

func (registrations Registrations) Add(endpoint *Endpoint) Registrations {
	name := endpoint.name
	for i, named := range registrations {
		if named.length > 0 && named.name == name {
			named.Add(endpoint)
			registrations[i] = named
			return registrations
		}
	}
	registration := Registration{
		name:   name,
		length: 0,
		pos:    0,
		values: make(Endpoints, 0, 1),
	}
	registration.Add(endpoint)
	return append(registrations, &registration)
}

func (registrations Registrations) Remove(name string, id string) Registrations {
	for i, named := range registrations {
		if named.length > 0 && named.name == name {
			named.Remove(id)
			if named.length == 0 {
				return append(registrations[:i], registrations[i+1:]...)
			}
			registrations[i] = named
			return registrations
		}
	}
	return registrations
}
