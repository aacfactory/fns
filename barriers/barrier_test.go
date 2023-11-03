package barriers_test

import (
	"context"
	"fmt"
	"github.com/aacfactory/fns/barriers"
	"github.com/aacfactory/fns/shareds"
	"sync"
	"testing"
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
	barrier := barriers.Cluster(shareds.LocalStore(), 0, 0)
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
