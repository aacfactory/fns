package cachecontrol

import (
	"bytes"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/context"
	"github.com/aacfactory/fns/services"
	"github.com/aacfactory/fns/transports"
	"github.com/valyala/bytebufferpool"
	"strconv"
)

var (
	pragma          = []byte("Pragma")
	noCache         = []byte("no-cache")
	noStore         = []byte("no-store")
	zeroMaxAge      = []byte("max-age=0")
	noTransform     = []byte("no-transform")
	public          = []byte("public")
	private         = []byte("private")
	maxAge          = []byte("max-age")
	mustRevalidate  = []byte("must-revalidate")
	proxyRevalidate = []byte("proxy-revalidate")
	comma           = []byte(", ")
	equal           = []byte("=")
)

type MakeOptions struct {
	mustRevalidate  bool
	proxyRevalidate bool
	public          bool
	maxAge          int
}

type MakeOption func(option *MakeOptions)

func MustRevalidate() MakeOption {
	return func(option *MakeOptions) {
		option.mustRevalidate = true
	}
}

func ProxyRevalidate() MakeOption {
	return func(option *MakeOptions) {
		option.proxyRevalidate = true
	}
}

func Private() MakeOption {
	return func(option *MakeOptions) {
		option.public = false
	}
}

func Public() MakeOption {
	return func(option *MakeOptions) {
		option.public = true
	}
}

func MaxAge(age int) MakeOption {
	return func(option *MakeOptions) {
		if age < 0 {
			age = 0
		}
		option.maxAge = age
	}
}

func Make(ctx context.Context, options ...MakeOption) {
	// check internal
	sr, hasSR := services.TryLoadRequest(ctx)
	if !hasSR {
		return
	}
	if sr.Header().Internal() {
		return
	}
	// check transport request path
	tr, hasTR := transports.TryLoadRequest(ctx)
	if !hasTR {
		return
	}
	path := tr.Path()
	s, f := sr.Fn()
	sf := append([]byte{'/'}, s...)
	sf = append(sf, '/')
	sf = append(sf, f...)
	if !bytes.Equal(path, sf) {
		return
	}

	header := tr.Header()
	// pragma
	ph := header.Get(pragma)
	if len(ph) > 0 && bytes.Equal(ph, noCache) {
		return
	}
	//
	noTransformEnabled := false
	// cache control
	cch := header.Get(bytex.FromString(transports.CacheControlHeaderName))
	if len(cch) > 0 {
		// no-cache, no-store, max-age=0
		if bytes.Contains(cch, noCache) || bytes.Contains(cch, noStore) || bytes.Contains(cch, zeroMaxAge) {
			return
		}
		if bytes.Contains(cch, noTransform) {
			noTransformEnabled = true
		}
	}
	// write response header
	responseHeader, hasResponseHeader := transports.TryLoadResponseHeader(ctx)
	if !hasResponseHeader {
		return
	}
	opt := MakeOptions{}
	for _, option := range options {
		option(&opt)
	}
	ccr := bytebufferpool.Get()
	if opt.public {
		_, _ = ccr.Write(comma)
		_, _ = ccr.Write(public)
		if opt.proxyRevalidate {
			_, _ = ccr.Write(comma)
			_, _ = ccr.Write(proxyRevalidate)
		}
		if opt.mustRevalidate {
			_, _ = ccr.Write(comma)
			_, _ = ccr.Write(mustRevalidate)
		}
		if noTransformEnabled {
			_, _ = ccr.Write(comma)
			_, _ = ccr.Write(noTransform)
		}
	} else {
		_, _ = ccr.Write(comma)
		_, _ = ccr.Write(private)
		if opt.mustRevalidate {
			_, _ = ccr.Write(comma)
			_, _ = ccr.Write(mustRevalidate)
		}
	}
	if opt.maxAge > 0 {
		_, _ = ccr.Write(comma)
		_, _ = ccr.Write(maxAge)
		_, _ = ccr.Write(equal)
		_, _ = ccr.Write(bytex.FromString(strconv.Itoa(opt.maxAge)))
	}
	h := ccr.Bytes()
	if len(h) > 0 {
		h = h[2:]
	}
	responseHeader.Set(bytex.FromString(transports.CacheControlHeaderName), h)
	bytebufferpool.Put(ccr)
	return
}
