package asredis

import (
	"time"
//	"runtime/debug"
	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"
//	"runtime/debug"
)

func TestClient(t *testing.T) {
	client:= NewClient("tcp", "127.0.0.1:6379", time.Second * 10)

	defer client.Close()

	req := client.Go(nil, "SET", "int", 1)
	reply, _ := req.GetReply()
	t.Log(reply.Type, reply.Value)

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	req = client.Go(nil, "GET", "int")
	reply, _ = req.GetReply()
	t.Log(reply.Type, reply.Value)
}

func TestError(t *testing.T) {

	client := NewClient("tcp", "127.0.0.1:6379", time.Second * 10)

	defer client.Close()

	go func() {
		time.Sleep(time.Second * 2)
		client.Close()
	}()

	for i:= 0; i < 5; i++ {
		req := client.Go(nil, "SET", "int", 1)
		reply, err := req.GetReply()
		if err == nil {
			fmt.Println(reply.Type, reply.Value)
		} else {
			fmt.Println(err)
		}
		time.Sleep(time.Second * 1)
	}
}
