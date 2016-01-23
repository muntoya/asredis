package asredis

import (
	"testing"
	"fmt"
)


func TestCluster(t *testing.T) {gaa
	cluster, err := NewCluster([]string{"127.0.0.1:7000"})
	fmt.Println(cluster, err)
}