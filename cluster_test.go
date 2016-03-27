package asredis

import (
	"testing"
	"fmt"
	"strconv"
	"github.com/stretchr/testify/assert"
)


func TestCluster(t *testing.T) {
	t.Skip("cluster")
	spec := DefaultClusterSpec()
	spec.Port = 7000
	cluster, _ := NewCluster(spec)

	for i := 0; i < 100; i++ {
		_, err := cluster.Call("set", fmt.Sprintf("int%d", i), i)
		assert.Equal(t, err, nil)
	}

	for i := 0; i < 100; i++ {
		reply, err := cluster.Call("get", fmt.Sprintf("int%d", i))
		assert.Equal(t, reply.Value, strconv.Itoa(i))
		assert.Equal(t, err, nil)
	}
}