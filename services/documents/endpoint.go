package documents

import "github.com/aacfactory/fns/commons/versions"

func New(name string, description string, internal bool, ver versions.Version) Endpoint {
	return Endpoint{
		Name:        name,
		Description: description,
		Internal:    internal,
		Version:     ver,
		Functions:   make(Fns, 0, 1),
		Elements:    make(Elements, 0, 1),
	}
}

type Endpoint struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Internal    bool             `json:"internal"`
	Version     versions.Version `json:"version"`
	Functions   Fns              `json:"functions"`
	Elements    Elements         `json:"elements"`
}

func (endpoint *Endpoint) IsEmpty() bool {
	return endpoint.Name == ""
}

func (endpoint *Endpoint) AddFn(fn Fn) {
	if fn.Param.Exist() {
		paramRef := endpoint.addElement(fn.Param)
		fn.Param = paramRef
	}
	if fn.Result.Exist() {
		paramRef := endpoint.addElement(fn.Result)
		fn.Result = paramRef
	}
	if endpoint.Internal {
		fn.Internal = true
	}
	endpoint.Functions = endpoint.Functions.Add(fn)
}

func (endpoint *Endpoint) addElement(element Element) (ref Element) {
	if !element.Exist() {
		return
	}
	unpacks := element.unpack()
	ref = unpacks[0]
	if len(unpacks) <= 1 {
		return
	}
	remains := unpacks[1:]
	for _, remain := range remains {
		if remain.isBuiltin() || remain.isRef() || remain.Path == "" {
			continue
		}
		endpoint.Elements = endpoint.Elements.Add(remain)
	}
	return
}

type Endpoints []Endpoint

func (endpoints Endpoints) Len() int {
	return len(endpoints)
}

func (endpoints Endpoints) Less(i, j int) bool {
	return endpoints[i].Name < endpoints[j].Name
}

func (endpoints Endpoints) Swap(i, j int) {
	endpoints[i], endpoints[j] = endpoints[j], endpoints[i]
	return
}
