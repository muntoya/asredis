package asredis

import (
	"time"
//	"runtime/debug"
	"testing"
	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestConnection(t *testing.T) {
	client:= NewConnection("127.0.0.1:6379")
	defer client.Shutdown()

	c := make(chan *Request, 1)
	reply, err := client.Call(c, "SET", "int", 2)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	reply, err = client.Call(c, "GET", "int")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}
	fmt.Print("end")
}

func TestError(t *testing.T) {
	t.Skip("skip connnectiong loop")
	client := NewConnection("127.0.0.1:6379")

	c := make(chan *Request, 1)

	go func() {
		time.Sleep(time.Second * 1)
		client.Conn.Close()
	}()

	for i:= 0; i < 3; i++ {
		fmt.Println("go", i)
		reply, err := client.Call(c, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply.Type, reply.Value)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
