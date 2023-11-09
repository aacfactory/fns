package proxies

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/fns/transports"
	"golang.org/x/net/http/httpguts"
	"net/textproto"
	"sync"
)

func NewProxyMiddleware(rt *runtime.Runtime) transports.Middleware {
	return &ProxyMiddleware{
		rt:      rt,
		counter: new(sync.WaitGroup),
	}
}

type ProxyMiddleware struct {
	rt      *runtime.Runtime
	counter *sync.WaitGroup
}

func (middleware *ProxyMiddleware) Name() string {
	return "proxy"
}

func (middleware *ProxyMiddleware) Construct(_ transports.MiddlewareOptions) error {
	return nil
}

func (middleware *ProxyMiddleware) Handler(next transports.Handler) transports.Handler {
	return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
		running, upped := middleware.rt.Running()
		if !running {
			w.Header().Set(bytex.FromString(transports.ConnectionHeaderName), bytex.FromString(transports.CloseHeaderValue))
			w.Failed(ErrUnavailable)
			return
		}
		if !upped {
			w.Header().Set(bytex.FromString(transports.ResponseRetryAfterHeaderName), bytex.FromString("3"))
			w.Failed(ErrTooEarly)
			return
		}
		// header >>>
		header := r.Header()
		// remove hop by hop headers
		connections := header.Values(bytex.FromString(transports.ConnectionHeaderName))
		// RFC 7230, section 6.1: Remove headers listed in the "Connection" header.
		for _, f := range connections {
			for _, sf := range bytes.Split(f, commaBytes) {
				if sf = bytex.FromString(textproto.TrimString(bytex.ToString(sf))); len(sf) > 0 {
					header.Del(sf)
				}
			}
		}
		// RFC 2616, section 13.5.1: Remove a set of known hop-by-hop headers.
		// This behavior is superseded by the RFC 7230 Connection header, but
		// preserve it for backwards compatibility.
		for _, f := range hopHeaders {
			header.Del(f)
		}
		// Issue 21096: tell backend applications that care about trailer support
		// that we support trailers. (We do, but we don't go out of our way to
		// advertise that unless the incoming client request thought it was worth
		// mentioning.) Note that we look at req.Header, not outreq.Header, since
		// the latter has passed through removeHopByHopHeaders.
		te := header.Values(bytex.FromString("Te"))
		if len(te) > 0 {
			tes := make([]string, 0, len(te))
			for _, s := range te {
				tes = append(tes, bytex.ToString(s))
			}
			if httpguts.HeaderValuesContainsToken(tes, "trailers") {
				header.Set(bytex.FromString("Te"), bytex.FromString("trailers"))
			}
		}

		// header <<<

		middleware.counter.Add(1)
		// set runtime into request context
		runtime.With(r, middleware.rt)
		// set request and response into context
		transports.WithRequest(r, r)
		transports.WithResponse(r, w)
		// next
		next.Handle(w, r)

		// check hijacked
		if w.Hijacked() {
			middleware.counter.Done()
			return
		}
		// done
		middleware.counter.Done()
	})
}

func (middleware *ProxyMiddleware) Close() {
	middleware.counter.Wait()
}

var hopHeaders = [][]byte{
	[]byte("Connection"),
	[]byte("Proxy-Connection"), // non-standard but still sent by libcurl and rejected by e.g. google
	[]byte("Keep-Alive"),
	[]byte("Proxy-Authenticate"),
	[]byte("Proxy-Authorization"),
	[]byte("Te"),      // canonicalized version of "TE"
	[]byte("Trailer"), // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	[]byte("Transfer-Encoding"),
	[]byte("Upgrade"),
}
