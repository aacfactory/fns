package barriers_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/aacfactory/fns/barriers"
	"sync"
	"testing"
	"time"
)

func TestStandalone(t *testing.T) {
	barrier := barriers.Standalone()
	r, err := barrier.Do(context.TODO(), []byte("key"), func() (result interface{}, err error) {
		result = 1
		return
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	v := 0
	err = r.Scan(&v)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(v)
}

func TestNew(t *testing.T) {
	store := &LocalStore{
		data:   sync.Map{},
		locker: new(sync.Mutex),
	}
	barrier := barriers.New(store, 0, 0, nil)
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(barrier *barriers.Barrier, wg *sync.WaitGroup) {
			defer wg.Done()
			r, err := barrier.Do(context.TODO(), []byte("key"), func() (result interface{}, err error) {
				result = 1
				fmt.Println("do")
				return
			})
			barrier.Forget(context.TODO(), []byte("key"))
			if err != nil {
				fmt.Println(err)
				return
			}
			v := 0
			err = r.Scan(&v)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(v)
		}(barrier, wg)
	}
	wg.Wait()
}

type LocalStore struct {
	data   sync.Map
	locker sync.Locker
}

func (store *LocalStore) Incr(ctx context.Context, key []byte, delta int64) (value int64, err error) {
	store.locker.Lock()
	defer store.locker.Unlock()
	var p []byte
	v, has := store.data.Load(string(key))
	if has {
		p = v.([]byte)
	} else {
		p = make([]byte, 10)
	}
	value, _ = binary.Varint(p)
	value += delta
	b := make([]byte, 10)
	binary.PutVarint(b, value)
	store.data.Store(string(key), b)
	return
}

func (store *LocalStore) Get(ctx context.Context, key []byte) (value []byte, has bool, err error) {
	v, ok := store.data.Load(string(key))
	if !ok {
		return
	}
	value = v.([]byte)
	has = true
	return
}

func (store *LocalStore) Set(ctx context.Context, key []byte, value []byte, ttl time.Duration) (err error) {
	store.data.Store(string(key), value)
	return
}

func (store *LocalStore) TTL(ctx context.Context, key []byte, ttl time.Duration) (err error) {
	return
}

func (store *LocalStore) Remove(ctx context.Context, key []byte) (err error) {
	store.data.Delete(string(key))
	return
}
