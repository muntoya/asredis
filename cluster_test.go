package asredis

import (
	"testing"
	"fmt"
	"strconv"
	"github.com/stretchr/testify/assert"
)


func TestCluster(t *testing.T) {
	cluster, err := NewCluster([]string{"127.0.0.1:7000"})

	for i := 0; i < 100; i++ {
		_, err := cluster.Exec("set", fmt.Sprintf("int%d", i), i)
		assert.Equal(t, err, nil)
	}

	for i := 0; i < 100; i++ {
		reply, err := cluster.Exec("get", fmt.Sprintf("int%d", i))
		assert.Equal(t, reply.Value, strconv.Itoa(i))
		assert.Equal(t, err, nil)
	}
}