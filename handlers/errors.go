package handlers

import (
	"github.com/aacfactory/errors"
	"net/http"
)

var (
	ErrServiceOverload        = errors.Unavailable("fns: service is overload").WithMeta("fns", "overload")
	ErrTooEarly               = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable            = errors.Unavailable("fns: service is closed")
	ErrDeviceId               = errors.Warning("fns: device id was required")
	ErrNotFound               = errors.NotFound("fns: no handlers accept request")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
	ErrTooMayRequest          = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUEST***", "fns: too may request, try again later.")
	ErrLockedRequest          = errors.New(http.StatusLocked, "***REQUEST LOCKED***", "fns: request was locked for idempotent.")
	ErrSignatureLost          = errors.New(488, "***SIGNATURE LOST***", "X-Fns-Signature was required")
	ErrSignatureUnverified    = errors.New(458, "***SIGNATURE INVALID***", "X-Fns-Signature was invalid")
)
