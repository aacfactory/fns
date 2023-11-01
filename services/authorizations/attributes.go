package authorizations

import (
	"bytes"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/json"
)

type Attribute struct {
	Key   []byte
	Value json.RawMessage
}

type Attributes []Attribute

func (attributes *Attributes) Get(key []byte, value interface{}) (has bool, err error) {
	attrs := *attributes
	for _, attribute := range attrs {
		if bytes.Equal(key, attribute.Key) {
			decodeErr := json.Unmarshal(attribute.Value, value)
			if decodeErr != nil {
				err = errors.Warning("fns: attributes get failed").WithCause(decodeErr).WithMeta("key", string(key))
				return
			}
			has = true
			return
		}
	}
	return
}

func (attributes *Attributes) Set(key []byte, value interface{}) (err error) {
	p, encodeErr := json.Marshal(value)
	if encodeErr != nil {
		err = errors.Warning("fns: attributes set failed").WithCause(encodeErr).WithMeta("key", string(key))
		return
	}
	attrs := *attributes
	attrs = append(attrs, Attribute{
		Key:   key,
		Value: p,
	})
	*attributes = attrs
	return
}

func (attributes *Attributes) Remove(key []byte) {
	attrs := *attributes
	n := -1
	for i, attribute := range attrs {
		if bytes.Equal(key, attribute.Key) {
			n = i
			break
		}
	}
	if n == -1 {
		return
	}
	attrs = append(attrs[:n], attrs[n+1:]...)
	*attributes = attrs
	return
}
