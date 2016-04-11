package asredis

import (
	"testing"
	"fmt"
	"sync"
)

func BenchmarkSet(b *testing.B) {
	spec := DefaultPoolSpec()
	pool := NewPool(spec)

	routineNum := 80
	times := 10000
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			key := fmt.Sprintf("int%d", i)
			for t := 0; t < times; t++ {
				_, e := pool.Call("set", key, n)
				if e != nil {
					panic(e)
				}
			}
			w.Done()
		}(i)
	}
	w.Wait()

	b.Logf("get %d", routineNum * times)
}

