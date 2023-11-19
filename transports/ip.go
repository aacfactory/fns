package transports

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/ipx"
	"github.com/aacfactory/fns/context"
	"net"
)

func DeviceIp(ctx context.Context) (ip []byte) {
	r := LoadRequest(ctx)
	if r == nil {
		return
	}
	ip = r.Header().Get(DeviceIpHeaderName)
	if len(ip) > 0 {
		return
	}
	if trueClientIp := r.Header().Get(TrueClientIpHeaderName); len(trueClientIp) > 0 {
		ip = trueClientIp
	} else if realIp := r.Header().Get(XRealIpHeaderName); len(realIp) > 0 {
		ip = realIp
	} else if forwarded := r.Header().Get(XForwardedForHeaderName); len(forwarded) > 0 {
		i := bytes.Index(forwarded, []byte{',', ' '})
		if i == -1 {
			i = len(forwarded)
		}
		ip = forwarded[:i]
	} else {
		remoteIp, _, remoteIpErr := net.SplitHostPort(bytex.ToString(r.RemoteAddr()))
		if remoteIpErr != nil {
			remoteIp = bytex.ToString(r.RemoteAddr())
		}
		ip = bytex.FromString(remoteIp)
	}
	ip = ipx.CanonicalizeIp(ip)
	r.Header().Set(DeviceIpHeaderName, ip)
	return
}
