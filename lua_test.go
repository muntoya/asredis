package asredis

import (
	"testing"
	"fmt"
)

func TestLuaEval(t *testing.T) {
	t.Skip("skip lua test")
	l, _ := NewLuaEval("scripts/get_test.lua")
	spec := DefaultPoolSpec()
	pool := NewPool(spec)
	reply, err := pool.Eval(l, 0)
	fmt.Println(fmt.Sprint(reply), err)
}