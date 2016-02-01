package asredis

import (
	"testing"
	"fmt"
)


func TestCluster(t *testing.T) {
	cluster, err := NewCluster([]string{"127.0.0.1:7000"})
	fmt.Println(cluster.addrs, err)
	var r *Reply
	r, err = cluster.Exec("set", "int", 8)
	fmt.Println(r, err)
	r, err = cluster.Exec("get", "int")
	fmt.Println(r, err)
}