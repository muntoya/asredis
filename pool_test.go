package asredis

import (
//	"time"
//	"runtime/debug"
	"testing"
	"github.com/stretchr/testify/assert"
//	"runtime/debug"
	"fmt"
	"strconv"
)

func TestPool(t *testing.T) {
	pool := NewPool("tcp", "127.0.0.1:6379", 5, 10)
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
