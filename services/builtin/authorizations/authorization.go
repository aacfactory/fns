package authorizations

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
	"strconv"
	"time"
)

type Id string

func (id Id) Int() int64 {
	if len(id) == 0 {
		return 0
	}
	v, err := strconv.ParseInt(string(id), 10, 64)
	if err != nil {
		panic(errors.Warning("fns: get int value from id failed").WithCause(err).WithMeta("id", id.String()))
		return 0
	}
	return v
}

func (id Id) String() string {
	return string(id)
}

func (id Id) Exist() (ok bool) {
	ok = len(id) > 0
	return
}

func StringId(id string) Id {
	return Id(id)
}

func IntId(id int64) Id {
	return Id(strconv.FormatInt(id, 10))
}

type Attributes map[string]json.RawMessage

func (attributes Attributes) Get(key []byte, value interface{}) (has bool, err error) {
	k := bytex.ToString(key)
	p, exist := attributes[k]
	if !exist {
		return
	}
	has = true
	decodeErr := json.Unmarshal(p, value)
	if decodeErr != nil {
		err = errors.Warning("fns: attributes get failed").WithCause(decodeErr).WithMeta("key", k)
		return
	}
	return
}

func (attributes Attributes) Set(key []byte, value interface{}) (err error) {
	k := bytex.ToString(key)
	p, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		err = errors.Warning("fns: attributes set failed").WithCause(encodeErr).WithMeta("key", k)
		return
	}
	attributes[k] = p
	return
}

func (attributes Attributes) Remove(key []byte) {
	k := bytex.ToString(key)
	delete(attributes, k)
	return
}

type Authorization struct {
	Id         Id         `json:"id"`
	Account    Id         `json:"account"`
	Attributes Attributes `json:"attributes"`
	ExpireAT   time.Time  `json:"expireAT"`
}
