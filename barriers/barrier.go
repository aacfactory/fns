package barriers

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/fns/commons/objects"
	"github.com/aacfactory/fns/shareds"
	"github.com/aacfactory/json"
	"golang.org/x/sync/singleflight"
	"time"
)

func Cluster(store shareds.Store, ttl time.Duration, interval time.Duration) (b *Barrier) {
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
	b = &Barrier{
		group:      new(singleflight.Group),
		standalone: false,
		store:      store,
		ttl:        ttl,
		interval:   interval,
		loops:      loops,
	}
	return
}

func Standalone() (b *Barrier) {
	b = &Barrier{
		group:      new(singleflight.Group),
		standalone: true,
	}
	return
}

type ErrorReporter func(ctx context.Context, cause error)

type Result struct {
	objects.Scanner
}

// Barrier
// @barrier
// todo 当@authorization 存在时，则key增加user，不存在时，不加user
type Barrier struct {
	group      *singleflight.Group
	standalone bool
	store      shareds.Store
	ttl        time.Duration
	interval   time.Duration
	loops      int
}

func (b *Barrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err error)) (result Result, err error) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	r, doErr, _ := b.group.Do(bytex.ToString(key), func() (r interface{}, err error) {
		if b.standalone {
			r, err = fn()
			return
		}
		times, incrErr := b.store.Incr(ctx, key, 1)
		if incrErr != nil {
			err = errors.Warning("fns: barrier failed").WithCause(incrErr)
			return
		}

		valueKey := append(key, bytex.FromString("-value")...)
		value := make([]byte, 0, 1)
		if times == 1 {
			ttlErr := b.store.ExpireKey(ctx, key, b.ttl)
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
			_ = b.store.SetWithTTL(ctx, valueKey, value, b.ttl)
		} else {
			fetched := false
			for i := 0; i < b.loops; i++ {
				p, exist, getErr := b.store.Get(ctx, valueKey)
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
	result = Result{
		objects.NewScanner(r),
	}
	return
}

func (b *Barrier) Forget(ctx context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	defer b.group.Forget(bytex.ToString(key))
	if b.standalone {
		return
	}
	for i := 0; i < b.loops; i++ {
		n, incrErr := b.store.Incr(ctx, key, -1)
		if incrErr != nil {
			continue
		}
		if n < 1 {
			valueKey := append(key, bytex.FromString("-value")...)
			rmErr := b.store.Remove(ctx, valueKey)
			if rmErr != nil {
				continue
			}
		}
		break
	}
}
