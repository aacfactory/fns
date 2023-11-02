package clusters

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/ipx"
)

type HostRetriever func() (host string, err error)

func defaultHostRetriever() (host string, err error) {
	ip := ipx.GetGlobalUniCastIpFromHostname()
	if ip == nil {
		err = errors.Warning("fns: get host from  hostname failed")
		return
	}
	host = ip.String()
	return
}

var (
	hostRetrievers = map[string]HostRetriever{
		"default": defaultHostRetriever,
	}
)

func RegisterHostRetriever(name string, fn HostRetriever) {
	hostRetrievers[name] = fn
}

func getHostRetriever(name string) (fn HostRetriever, has bool) {
	fn, has = hostRetrievers[name]
	return
}
