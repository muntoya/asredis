package asredis

import (
//	"time"
//	"runtime/debug"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	spec := DefaultPoolSpec()
	pool := NewPool(spec)

	for _, command := range testCommands {
		var reply interface{}
		var err error
		reply, err = pool.Call(command.args[0].(string), command.args[1:]...)
		assert.Nil(t, err)
		assert.Equal(t, command.expected, reply)
	}
}
