package fns_test

import (
	"github.com/aacfactory/fns"
	"testing"
)

func TestIpFromHostname(t *testing.T) {
	t.Log(fns.IpFromHostname(true))
	t.Log(fns.IpFromHostname(false))
}
