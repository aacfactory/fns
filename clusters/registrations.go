package clusters

import (
	"context"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/signatures"
	"github.com/aacfactory/fns/commons/versions"
	"github.com/aacfactory/fns/commons/window"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/services/documents"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/json"
	"github.com/aacfactory/logs"
	"github.com/aacfactory/rings"
	"github.com/aacfactory/workers"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Registrations struct {
	id              string
	name            string
	version         versions.Version
	log             logs.Logger
	cluster         Cluster
	devMode         bool
	values          sync.Map
	nodes           map[string]*Node
	signer          signatures.Signature
	dialer          transports.Dialer
	worker          workers.Workers
	timeout         time.Duration
	refreshInterval time.Duration
}

func (r *Registrations) add(registration *Registration) {
	var ring *rings.Ring[*Registration]
	v, loaded := r.values.Load(registration.name)
	if !loaded || v == nil {
		v = rings.New[*Registration](registration.name)
		r.values.Store(registration.name, v)
	}
	ring, _ = v.(*rings.Ring[*Registration])
	ring.Push(registration)
	return
}

func (r *Registrations) Remove(id string) {
	empties := make([]string, 0, 1)
	r.values.Range(func(key, value any) bool {
		ring, _ := value.(*rings.Ring[*Registration])
		_, has := ring.Get(id)
		if has {
			ring.Remove(id)
			if ring.Len() == 0 {
				empties = append(empties, key.(string))
			}
		}
		return true
	})
	for _, empty := range empties {
		r.values.Delete(empty)
	}
	return
}

func (r *Registrations) GetExact(name string, id string) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded || v == nil {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	registration, has = ring.Get(id)
	if !has || registration == nil {
		return
	}
	if registration.closed.Load() {
		r.Remove(registration.id)
		registration = nil
		has = false
		return
	}
	if registration.errs.Value() > 10 {
		registration = nil
		has = false
		return
	}
	return
}

func (r *Registrations) Get(name string, rvs services.RequestVersions) (registration *Registration, has bool) {
	v, loaded := r.values.Load(name)
	if !loaded || v == nil {
		return
	}
	ring, _ := v.(*rings.Ring[*Registration])
	if ring.Len() == 0 {
		return
	}
	size := ring.Len()
	for i := 0; i < size; i++ {
		registration = ring.Next()
		if registration == nil {
			continue
		}
		if registration.closed.Load() {
			r.Remove(registration.id)
			continue
		}
		if registration.errs.Value() > 10 {
			continue
		}
		if rvs == nil || len(rvs) == 0 {
			has = true
			return
		}
		if rvs.Accept(name, registration.version) {
			has = true
			return
		}
	}
	return
}

func (r *Registrations) Close() {
	r.values.Range(func(key, value any) bool {
		entries := value.(*rings.Ring[*Registration])
		size := entries.Len()
		for i := 0; i < size; i++ {
			entry, ok := entries.Pop()
			if ok {
				entry.Close()
			}
		}
		return true
	})
	return
}

func (r *Registrations) List() (values map[string]RegistrationList) {
	values = make(map[string]RegistrationList)
	r.values.Range(func(key, value any) bool {
		name := key.(string)
		group, has := values[name]
		if !has {
			group = make([]*Registration, 0, 1)
			values[name] = group
		}
		ring, _ := value.(*rings.Ring[*Registration])
		size := ring.Len()
		for i := 0; i < size; i++ {
			registration := ring.Next()
			if registration == nil {
				continue
			}
			if registration.closed.Load() {
				continue
			}
			group = append(group, registration)
		}
		return true
	})
	empties := make([]string, 0, 1)
	for name, list := range values {
		if len(list) == 0 {
			empties = append(empties, name)
			continue
		}
		sort.Sort(list)
	}
	for _, empty := range empties {
		delete(values, empty)
	}
	return
}

func (r *Registrations) AddNode(node *Node) (err error) {
	// todo use transport
	// get service names
	// foreach names to get doc
	if node.Services == nil || len(node.Services) == 0 {
		return
	}
	client, clientErr := r.dialer.Dial(node.Address)
	if clientErr != nil {
		err = errors.Warning("registrations: add node failed").WithCause(clientErr)
		return
	}

	for _, svc := range node.Services {
		registration := &Registration{
			hostId:  r.id,
			id:      node.Id,
			version: node.Version,
			address: node.Address,
			devMode: r.devMode,
			name:    svc.Name,
			client:  client,
			signer:  r.signer,
			worker:  r.worker,
			timeout: r.timeout,
			pool:    sync.Pool{},
			closed:  &atomic.Bool{},
			errs:    window.NewTimes(10 * time.Second),
		}
		registration.pool.New = func() any {
			return newRegistrationTask(registration, registration.timeout, func(task *registrationTask) {
				registration.release(task)
			})
		}
		r.add(registration)
	}
	r.nodes[node.Id] = node
	return
}

func (r *Registrations) FetchNodeDocuments(ctx context.Context, node *Node) (v documents.Documents, err error) {
	client, clientErr := r.dialer.Dial(node.Address)
	if clientErr != nil {
		err = errors.Warning("registrations: fetch node documents failed").WithCause(clientErr)
		return
	}

	req := transports.NewUnsafeRequest(ctx, transports.MethodGET, bytex.FromString(documents.ServicesDocumentsPath))
	req.Header().Set(transports.DeviceIdHeaderName, r.id)

	for i := 0; i < 5; i++ {
		resp, doErr := client.Do(ctx, req)
		if doErr != nil {
			err = errors.Warning("registrations: fetch node documents failed").WithCause(doErr)
			return
		}
		if resp.Status == http.StatusTooEarly {
			time.Sleep(1 * time.Second)
			continue
		}
		if resp.Status != http.StatusOK {
			err = errors.Warning("registrations: fetch node documents failed")
			return
		}
		if resp.Body == nil || len(resp.Body) == 0 {
			return
		}
		v = documents.NewDocuments()
		decodeErr := json.Unmarshal(resp.Body, &v)
		if decodeErr != nil {
			err = errors.Warning("registrations: fetch node documents failed").WithCause(decodeErr)
			return
		}
		break
	}
	return
}
