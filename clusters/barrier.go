package clusters

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/runtime"
	"github.com/aacfactory/json"
	"golang.org/x/sync/singleflight"
	"time"
)

func NewDefaultBarrier() (b barriers.Barrier) {
	b = &DefaultBarrier{}
	return
}

type DefaultBarrierConfig struct {
	TTL        time.Duration `json:"ttl"`
	Interval   time.Duration `json:"interval"`
	Standalone bool          `json:"standalone"`
}

type DefaultBarrier struct {
	group      *singleflight.Group
	standalone bool
	ttl        time.Duration
	interval   time.Duration
	loops      int
}

func (b *DefaultBarrier) Construct(options barriers.Options) (err error) {
	config := DefaultBarrierConfig{}
	configErr := options.Config.As(&config)
	if configErr != nil {
		err = errors.Warning("fns: default cluster barrier construct failed").WithCause(configErr)
		return
	}
	ttl := config.TTL
	interval := config.Interval
	loops := 0
	if ttl < 1 {
		ttl = 10 * time.Second
	}
	if interval < 1 {
		interval = 100 * time.Millisecond
	}
	if interval >= ttl {
		loops = 10
		interval = ttl / time.Duration(loops)
	} else {
		loops = int(ttl / interval)
	}
	b.standalone = config.Standalone
	b.group = new(singleflight.Group)
	return
}

func (b *DefaultBarrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result barriers.Result, err error) {

	if len(key) == 0 {
		key = []byte{'-'}
	}
	r, doErr, _ := b.group.Do(bytex.ToString(key), func() (r interface{}, err error) {
		if b.standalone {
			r, err = fn()
			return
		}
		store := runtime.Load(ctx).Shared().Store()
		times, incrErr := store.Incr(ctx, key, 1)
		if incrErr != nil {
			err = errors.Warning("fns: barrier failed").WithCause(incrErr)
			return
		}

		valueKey := append(key, bytex.FromString("-value")...)
		value := make([]byte, 0, 1)
		if times == 1 {
			ttlErr := store.ExpireKey(ctx, key, b.ttl)
			if ttlErr != nil {
				err = errors.Warning("fns: barrier failed").WithCause(incrErr)
				return
			}
			r, err = fn()
			if err == nil {
				value = append(value, 'T')
				if r == nil {
					value = append(value, 'N')
				} else {
					value = append(value, 'V')
					p, encodeErr := json.Marshal(r)
					if encodeErr != nil {
						panic(fmt.Sprintf("%+v", errors.Warning("fns: barrier failed").WithCause(encodeErr)))
					}
					value = append(value, p...)
				}
			} else {
				value = append(value, 'F')
				codeErr, isCodeErr := err.(errors.CodeError)
				if isCodeErr {
					p, _ := json.Marshal(codeErr)
					value = append(value, 'C')
					value = append(value, p...)
				} else {
					value = append(value, 'S')
					value = append(value, bytex.FromString(err.Error())...)
				}
			}
			_ = store.SetWithTTL(ctx, valueKey, value, b.ttl)
		} else {
			fetched := false
			for i := 0; i < b.loops; i++ {
				p, exist, getErr := store.Get(ctx, valueKey)
				if getErr != nil {
					err = errors.Warning("fns: barrier failed").WithCause(getErr)
					return
				}
				if !exist {
					time.Sleep(b.interval)
					continue
				}
				if len(p) < 2 {
					err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
					return
				}
				if p[0] == 'T' {
					if p[1] == 'N' {
						r = nil
					} else if p[1] == 'V' {
						r = p[2:]
					} else {
						err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
						return
					}
				} else if p[0] == 'F' {
					if p[1] == 'C' {
						err = errors.Decode(p[2:])
					} else if p[1] == 'S' {
						err = fmt.Errorf(bytex.ToString(p[2:]))
					} else {
						err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
						return
					}
				} else {
					err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
					return
				}
				fetched = true
				break
			}
			if !fetched {
				r, err = fn()
			}
		}
		return
	})
	if doErr != nil {
		err = errors.Map(doErr)
		return
	}
	result = barriers.Result{
		Scanner: objects.NewScanner(r),
	}
	return
}

func (b *DefaultBarrier) Forget(ctx context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	if b.standalone {
		b.group.Forget(bytex.ToString(key))
		return
	}
	store := runtime.Load(ctx).Shared().Store()
	for i := 0; i < b.loops; i++ {
		n, incrErr := store.Incr(ctx, key, -1)
		if incrErr != nil {
			continue
		}
		if n < 1 {
			valueKey := append(key, bytex.FromString("-value")...)
			rmErr := store.Remove(ctx, valueKey)
			if rmErr != nil {
				continue
			}
		}
		break
	}
	b.group.Forget(bytex.ToString(key))
}
