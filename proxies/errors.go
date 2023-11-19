package proxies

import (
	"github.com/aacfactory/errors"
	"net/http"
)

var (
	ErrDeviceId               = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrInvalidPath            = errors.Warning("fns: invalid path")
	ErrInvalidRequestVersions = errors.Warning("fns: invalid request versions")
)
