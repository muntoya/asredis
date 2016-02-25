package asredis

import (
//	"time"
//	"runtime/debug"
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
	"strconv"
	"sync"
)

func TestPool(t *testing.T) {
	t.Skip("skip pool")
	spec := DefaultPoolSpec()
	pool := NewPool(spec)
	for i := 0; i < 100; i++ {
		_, err := pool.Call("set", fmt.Sprintf("int%d", i), i)
		assert.Equal(t, err, nil)
	}

	for i := 0; i < 100; i++ {
		reply, err := pool.Call("get", fmt.Sprintf("int%d", i))
		assert.Equal(t, reply.Value, strconv.Itoa(i))
		assert.Equal(t, err, nil)
	}
}


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
