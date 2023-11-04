package proxies

import (
	"github.com/aacfactory/errors"
	"net/http"
)

var (
	ErrTooEarly               = errors.New(http.StatusTooEarly, "***TOO EARLY***", "fns: service is not ready, try later again")
	ErrUnavailable            = errors.Unavailable("fns: service is closed")
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidBody            = errors.Warning("fns: invalid body")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
)
