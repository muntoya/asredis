package asredis

import (
	"time"
//	"runtime/debug"
//	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"
//	"runtime/debug"
	"fmt"
)

func TestClient(t *testing.T) {
	client:= NewClient("tcp", "127.0.0.1:6379")
	defer client.Shutdown()

	c := make(chan *Request, 1)
	req := client.Go(c, "SET", "int", 2)
	reply, err := req.GetReply()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	req = client.Go(c, "GET", "int")
	reply, err = req.GetReply()
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}
	fmt.Print("end")
}

func TestError(t *testing.T) {
	t.Skip("skip connnectiong loop")
	client := NewClient("tcp", "127.0.0.1:6379")

	c := make(chan *Request, 1)

	go func() {
		time.Sleep(time.Second * 1)
		client.Conn.Close()
	}()

	for i:= 0; i < 3; i++ {
		fmt.Println("go", i)
		req := client.Go(c, "GET", "int")
		reply, err := req.GetReply()
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply.Type, reply.Value)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
