package limiter

import (
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/transports"
	"github.com/aacfactory/logs"
	"github.com/hashicorp/golang-lru/arc/v2"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
	"net/http"
	"time"
)

var (
	ErrDeviceId   = errors.New(http.StatusNotAcceptable, "***NOT ACCEPTABLE**", "fns: X-Fns-Device-Id is required")
	ErrNotAllowed = errors.New(http.StatusTooManyRequests, "***TOO MANY REQUESTS***", "fns: too many requests")
)

type Config struct {
	Enable       bool `json:"enable"`
	EverySeconds int  `json:"everySeconds"`
	Burst        int  `json:"burst"`
	MaxDevice    int  `json:"maxDevice"`
}

func New() transports.Middleware {
	return &middleware{}
}

type middleware struct {
	log    logs.Logger
	enable bool
	every  time.Duration
	burst  int
	cache  *arc.ARCCache[string, *rate.Limiter]
	group  *singleflight.Group
}

func (middle *middleware) Name() string {
	return "limiter"
}

func (middle *middleware) Construct(options transports.MiddlewareOptions) (err error) {
	middle.log = options.Log
	config := Config{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: construct limiter middleware failed").WithCause(configErr)
		return
	}
	if config.Enable {
		middle.enable = true
		everySeconds := config.EverySeconds
		if everySeconds < 1 {
			everySeconds = 10
		}
		middle.every = time.Duration(everySeconds) * time.Second
		burst := config.Burst
		if burst < 1 {
			burst = 10
		}
		middle.burst = burst
		maxDevice := config.MaxDevice
		if maxDevice < 1 {
			maxDevice = 4096
		}
		middle.cache, err = arc.NewARC[string, *rate.Limiter](maxDevice)
		if err != nil {
			err = errors.Warning("fns: construct limiter middleware failed").WithCause(err)
			return
		}
		middle.group = new(singleflight.Group)
	}
	return
}

func (middle *middleware) Handler(next transports.Handler) transports.Handler {
	if middle.enable {
		return transports.HandlerFunc(func(w transports.ResponseWriter, r transports.Request) {
			deviceId := r.Header().Get(transports.DeviceIdHeaderName)
			if len(deviceId) == 0 {
				w.Failed(ErrDeviceId)
				return
			}
			limiter := middle.getDeviceLimiter(deviceId)
			allowed := limiter.Allow()
			if !allowed {
				w.Failed(ErrNotAllowed)
				return
			}
			next.Handle(w, r)
		})
	}
	return next
}

func (middle *middleware) Close() {
	return
}

func (middle *middleware) getDeviceLimiter(deviceId []byte) (limiter *rate.Limiter) {
	id := bytex.ToString(deviceId)
	v, _, _ := middle.group.Do(id, func() (v interface{}, err error) {
		has := false
		limiter, has = middle.cache.Get(id)
		if !has {
			limiter = rate.NewLimiter(rate.Every(middle.every), middle.burst)
			middle.cache.Add(id, limiter)
		}
		v = limiter
		return
	})
	limiter = v.(*rate.Limiter)
	return
}
