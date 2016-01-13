package asredis

import (
	"testing"
	"fmt"
)

func TestLuaEval(t *testing.T) {
	l, _ := NewLuaEval("scripts/get_test.lua")
	pool := NewPool("127.0.0.1:6379", 5, 10)
	reply, err := pool.Eval(l, 0)
	fmt.Println(reply, err)
}