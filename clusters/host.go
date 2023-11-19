package clusters

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/ipx"
	"os"
)

type HostRetriever func() (host string, err error)

func defaultHostRetriever() (host string, err error) {
	ip := ipx.GetGlobalUniCastIpFromHostname()
	if ip == nil {
		err = errors.Warning("fns: get host from hostname failed")
		return
	}
	host = ip.String()
	return
}

func environmentHostRetriever() (host string, err error) {
	v, has := os.LookupEnv("FNS-HOST")
	if has {
		host = v
		return
	}
	err = errors.Warning("fns: get host from FNS-HOST env failed")
	return
}

var (
	hostRetrievers = map[string]HostRetriever{
		"default": defaultHostRetriever,
		"env":     environmentHostRetriever,
	}
)

func RegisterHostRetriever(name string, fn HostRetriever) {
	hostRetrievers[name] = fn
}

func getHostRetriever(name string) (fn HostRetriever, has bool) {
	fn, has = hostRetrievers[name]
	return
}
