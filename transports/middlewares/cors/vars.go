package cors

import (
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"net/http"
)

var (
	varyHeader                               = transports.VaryHeaderName
	accessControlRequestMethodHeader         = transports.AccessControlRequestMethodHeaderName
	originHeader                             = transports.OriginHeaderName
	accessControlRequestHeadersHeader        = transports.AccessControlRequestHeadersHeaderName
	accessControlRequestPrivateNetworkHeader = transports.AccessControlRequestPrivateNetworkHeaderName
	accessControlAllowOriginHeader           = transports.AccessControlAllowOriginHeaderName
	accessControlAllowMethodsHeader          = transports.AccessControlAllowMethodsHeaderName
	accessControlAllowHeadersHeader          = transports.AccessControlAllowHeadersHeaderName
	accessControlAllowCredentialsHeader      = transports.AccessControlAllowCredentialsHeaderName
	accessControlAllowPrivateNetworkHeader   = transports.AccessControlAllowPrivateNetworkHeaderName
	accessControlMaxAgeHeader                = transports.AccessControlMaxAgeHeaderName
	accessControlExposeHeadersHeader         = transports.AccessControlExposeHeadersHeaderName
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
