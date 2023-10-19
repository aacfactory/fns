package barriers

import (
	"context"
	"fmt"
	"github.com/aacfactory/errors"
	"github.com/aacfactory/fns/commons/bytex"
	"github.com/aacfactory/json"
	"golang.org/x/sync/singleflight"
	"time"
)

func New(store Store, ttl time.Duration, interval time.Duration, errorReporter ErrorReporter) (b *Barrier) {
	standalone := store == nil
	loops := 0
	if !standalone {
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
	}
	b = &Barrier{
		group:         new(singleflight.Group),
		standalone:    standalone,
		store:         store,
		ttl:           ttl,
		interval:      interval,
		loops:         loops,
		errorReporter: errorReporter,
	}
	return
}

func Standalone() (b *Barrier) {
	b = New(nil, 0, 0, nil)
	return
}

type Store interface {
	Incr(ctx context.Context, key []byte, delta int64) (value int64, err error)
	Get(ctx context.Context, key []byte) (value []byte, has bool, err error)
	Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error)
	TTL(ctx context.Context, key []byte, ttl time.Duration) (err error)
	Remove(ctx context.Context, key []byte) (err error)
}

type ErrorReporter func(ctx context.Context, cause error)

// Barrier
// @barrier
// 当@authorization 存在时，则key增加user，不存在时，不加user
type Barrier struct {
	group         *singleflight.Group
	standalone    bool
	store         Store
	ttl           time.Duration
	interval      time.Duration
	loops         int
	errorReporter ErrorReporter
}

func (b *Barrier) Do(ctx context.Context, key []byte, fn func() (result interface{}, err errors.CodeError)) (result interface{}, err errors.CodeError) {
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
			if b.errorReporter != nil {
				b.errorReporter(ctx, errors.Warning("fns: barrier incr failed").WithCause(incrErr))
			}
			return
		}
		valueKey := append(key, bytex.FromString("-value")...)
		value := make([]byte, 0, 1)
		if times == 1 {
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
			setErr := b.store.Set(ctx, valueKey, value, b.ttl)
			if setErr != nil {
				if b.errorReporter != nil {
					b.errorReporter(ctx, errors.Warning("fns: barrier set value failed").WithCause(setErr))
				}
			}
		} else {
			fetched := false
			for i := 0; i < b.loops; i++ {
				p, exist, getErr := b.store.Get(ctx, valueKey)
				if getErr != nil {
					err = errors.Warning("fns: barrier failed").WithCause(getErr)
					if b.errorReporter != nil {
						b.errorReporter(ctx, errors.Warning("fns: barrier get value failed").WithCause(getErr))
					}
					return
				}
				if !exist {
					time.Sleep(b.interval)
					continue
				}
				if len(p) < 2 {
					err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
					if b.errorReporter != nil {
						b.errorReporter(ctx, errors.Warning("fns: barrier get value failed").WithCause(fmt.Errorf("invalid value")))
					}
					return
				}
				if p[0] == 'T' {
					if p[1] == 'N' {
						break
					} else if p[1] == 'V' {

						break
					} else {
						err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
						if b.errorReporter != nil {
							b.errorReporter(ctx, errors.Warning("fns: barrier get value failed").WithCause(fmt.Errorf("invalid value")))
						}
						return
					}
				} else if p[0] == 'F' {
					// todo map bytes to object
				} else {
					err = errors.Warning("fns: barrier failed").WithCause(fmt.Errorf("invalid value"))
					if b.errorReporter != nil {
						b.errorReporter(ctx, errors.Warning("fns: barrier get value failed").WithCause(fmt.Errorf("invalid value")))
					}
					return
				}

				fetched = true
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
	result = r
	// incr: when 1, first then do fn, else wait
	return
}

func (b *Barrier) Forget(ctx context.Context, key []byte) {
	if len(key) == 0 {
		key = []byte{'-'}
	}
	defer b.group.Forget(bytex.ToString(key))
	// decr: when 0, last then remove, else skip
}
