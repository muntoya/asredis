package asredis

import (
	"time"
//	"runtime/debug"
	"testing"
	"fmt"
	"sync"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	//t.Skip("skip connection test")
	client:= NewConnection("127.0.0.1:6379", defaultPPLen, defaultSendTimeout)
	defer client.close()

	c := make(chan *Request, 1)
	reply, err := client.call(c, "SET", "int", 2)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	reply, err = client.call(c, "GET", "int")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	l := []interface{}{"1", "2", "3", "4", "5"}
	client.call(c, "DEL", "list")
	reply, err = client.call(c, "RPUSH", append([]interface{}{"list"}, l...)...)
	reply, err = client.call(c, "LRANGE", "list", 0, -1)
	assert.Equal(t, reply.Array, l)
}

func TestConnRoutine(t *testing.T) {
	//t.Skip("skip connection routine")
	conn:= NewConnection("127.0.0.1:6379", defaultPPLen, defaultSendTimeout)
	defer conn.close()

	routineNum := 100
	times := 100
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			c := make(chan *Request, 1)
			key := fmt.Sprintf("int%d", i)
			for t := 0; t < times; t++ {
				_, e := conn.call(c, "set", key, n)
				if e != nil {
					panic(e)
				}
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func TestConnError(t *testing.T) {
	t.Skip("skip connnection loop")
	client := NewConnection("127.0.0.1:6379", defaultPPLen, defaultSendTimeout)

	c := make(chan *Request, 1)

	go func() {
		time.Sleep(time.Second * 1)
		client.Conn.Close()
	}()

	for i:= 0; i < 3; i++ {
		fmt.Println("go", i)
		reply, err := client.call(c, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply.Type, reply.Value)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
