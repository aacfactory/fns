package users

import (
	"encoding/binary"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
)

type Id []byte

func (id Id) Int() (n int64) {
	n, _ = binary.Varint(id)
	return
}

func (id Id) String() string {
	return string(id)
}

func (id Id) Exist() (ok bool) {
	ok = len(id) > 0
	return
}

func StringId(id string) Id {
	return bytex.FromString(id)
}

func IntId(id int64) Id {
	v := make([]byte, 10)
	binary.PutVarint(v, id)
	return v
}

type Attributes struct {
	value *json.Object
}

func (attr *Attributes) Set(key []byte, value interface{}) (err error) {
	err = attr.value.Put(bytex.ToString(key), value)
	if err != nil {
		err = errors.Warning("fns: user attribute put failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	return
}

func (attr *Attributes) Get(key []byte, value interface{}) (has bool, err error) {
	has = attr.value.Contains(bytex.ToString(key))
	if !has {
		return
	}
	err = attr.value.Get(bytex.ToString(key), value)
	if err != nil {
		err = errors.Warning("fns: user attribute get failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	return
}

func (attr *Attributes) Remove(key []byte) (err error) {
	err = attr.value.Remove(bytex.ToString(key))
	if err != nil {
		err = errors.Warning("fns: user attribute remove failed").WithMeta("key", bytex.ToString(key)).WithCause(err)
		return
	}
	return
}

func (attr *Attributes) MarshalJSON() ([]byte, error) {
	return attr.value.MarshalJSON()
}

func (attr *Attributes) UnmarshalJSON(p []byte) error {
	if len(p) < 2 {
		p = []byte{'{', '}'}
	}
	if !json.Validate(p) {
		return errors.Warning("fns: unmarshal user attribute failed")
	}
	attr.value = json.NewObjectFromBytes(p)
	return nil
}

func New(id Id) User {
	return &user{
		Id_: id,
		Attr: Attributes{
			value: json.NewObject(),
		},
	}
}

type User interface {
	json.Marshaler
	json.Unmarshaler
	Id() Id
	Attributes() Attributes
	Exist() bool
}

type user struct {
	Id_  Id         `json:"id"`
	Attr Attributes `json:"attr"`
}

func (u *user) Id() Id {
	return u.Id_
}

func (u *user) Attributes() Attributes {
	return u.Attr
}

func (u *user) Exist() bool {
	return u.Id().Exist()
}

func (u *user) MarshalJSON() ([]byte, error) {
	return json.Marshal(u)
}

func (u *user) UnmarshalJSON(p []byte) error {
	return json.Unmarshal(p, u)
}
