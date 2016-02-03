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
	pool := NewPool("127.0.0.1:6379", 5, 10)
	for i := 0; i < 100; i++ {
		_, err := pool.Exec("set", fmt.Sprintf("int%d", i), i)
		assert.Equal(t, err, nil)
	}

	for i := 0; i < 100; i++ {
		reply, err := pool.Exec("get", fmt.Sprintf("int%d", i))
		assert.Equal(t, reply.Value, strconv.Itoa(i))
		assert.Equal(t, err, nil)
	}
}


func BenchmarkSet(b *testing.B) {
	pool := NewPool("127.0.0.1:6379", 10, 5000)

	routineNum := 100
	times := 10000
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			b.Logf("key %v", key)
			for t := 0; t < times; t++ {
				pool.Exec("set", key, n)
			}
			w.Done()
		}(i)
	}
	w.Wait()

	b.Logf("get %d", routineNum * times)
}
