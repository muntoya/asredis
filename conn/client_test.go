package conn

import (
	"time"
//	"runtime/debug"

	"testing"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	client, err := NewClient("tcp", "127.0.0.1:6379", time.Second * 10)
	defer client.Close()

	if err != nil {
		t.Fatal(err)
	}

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
	client, err := NewClient("tcp", "127.0.0.1:6379", time.Second * 10)
	defer client.Close()

	if err != nil {
		t.Fatal(err)
	}

//	for i:= 0; i < 5; i++ {
//		req := client.Go(nil, "SET", "int", 1)
//		req.GetReply()
//		time.Sleep(time.Second * 1)
//	}
}
