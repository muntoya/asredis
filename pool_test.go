package asredis

import (
	"time"
//	"runtime/debug"
	"testing"
//	"github.com/stretchr/testify/assert"
//	"runtime/debug"
	"fmt"
)

func TestPool(t *testing.T) {
	pool := NewPool("tcp", "127.0.0.1:6379", 5, 50)
	reply, err := pool.Exec("get", "int")
	time.Sleep(time.Second * 1)
	fmt.Println(reply, err)
}
