package fns_test

import (
	"github.com/aacfactory/fns"
	"testing"
)

func TestNewJsonObject(t *testing.T) {
	o := fns.NewJsonObject()
	_ = o.Put("id", "id")
	t.Log(o.Contains("id"))
	_ = o.Rem("id")
	t.Log(o.Contains("id"))
	m := make(map[string]string)
	m["1"] = "1"
	m["2"] = "2"
	_ = o.Put("m", m)
	nm := make(map[string]string)
	getErr := o.Get("m", &nm)
	t.Log(getErr, nm)
}

func TestNewJsonArray(t *testing.T) {
	a := fns.NewJsonArray()
	_ = a.Add(1, 2, 3, 4, 5)
	t.Log(a.Len(), string(a.Raw()))
	_ = a.Rem(1)
	t.Log(a.Len(), string(a.Raw()))
	n0 := 0
	_ = a.Get(0, &n0)
	t.Log(n0)
}
