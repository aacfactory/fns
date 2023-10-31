package clusters

import (
	"github.com/aacfactory/errors"
	"net/http"
)

var (
	ErrTooEarly               = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable            = errors.Unavailable("fns: service is closed")
	ErrDeviceId               = errors.Warning("fns: device id was required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
	ErrTooMayRequest          = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.")
	ErrSignatureLost          = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified    = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
)
