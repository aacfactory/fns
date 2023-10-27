package cors

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	varyHeader                               = bytex.FromString(transports.VaryHeaderName)
	accessControlRequestMethodHeader         = bytex.FromString(transports.AccessControlRequestMethodHeaderName)
	originHeader                             = bytex.FromString(transports.OriginHeaderName)
	accessControlRequestHeadersHeader        = bytex.FromString(transports.AccessControlRequestHeadersHeaderName)
	accessControlRequestPrivateNetworkHeader = bytex.FromString(transports.AccessControlRequestPrivateNetworkHeaderName)
	accessControlAllowOriginHeader           = bytex.FromString(transports.AccessControlAllowOriginHeaderName)
	accessControlAllowMethodsHeader          = bytex.FromString(transports.AccessControlAllowMethodsHeaderName)
	accessControlAllowHeadersHeader          = bytex.FromString(transports.AccessControlAllowHeadersHeaderName)
	accessControlAllowCredentialsHeader      = bytex.FromString(transports.AccessControlAllowCredentialsHeaderName)
	accessControlAllowPrivateNetworkHeader   = bytex.FromString(transports.AccessControlAllowPrivateNetworkHeaderName)
	accessControlMaxAgeHeader                = bytex.FromString(transports.AccessControlMaxAgeHeaderName)
	accessControlExposeHeadersHeader         = bytex.FromString(transports.AccessControlExposeHeadersHeaderName)
)

var (
	methodOptions = bytex.FromString(http.MethodOptions)
	methodHead    = bytex.FromString(http.MethodHead)
	methodGet     = bytex.FromString(http.MethodGet)
	methodPost    = bytex.FromString(http.MethodPost)
)

var (
	all       = []byte{'*'}
	trueBytes = []byte{'t', 'r', 'u', 'e'}
	joinBytes = []byte{',', ' '}
)
