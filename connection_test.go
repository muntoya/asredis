package asredis

import (
	"time"
	//	"runtime/debug"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestConnection(t *testing.T) {
	//t.Skip("skip connection test")
	spec := DefaultConnectionSpec()
	conn := NewConnection(spec)
	defer conn.close()

	c := make(chan *Request, 1)
	reply, err := conn.call(c, "SET", "int", 2)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	reply, err = conn.call(c, "GET", "int")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	l := []interface{}{"1", "2", "3", "4", "5"}
	conn.call(c, "DEL", "list")
	reply, err = conn.call(c, "RPUSH", append([]interface{}{"list"}, l...)...)
	reply, err = conn.call(c, "LRANGE", "list", 0, -1)
	assert.Equal(t, reply.Array, l)
}

func TestConnRoutine(t *testing.T) {
	//t.Skip("skip connection routine")
	spec := DefaultConnectionSpec()
	conn := NewConnection(spec)
	defer conn.close()

	routineNum := 50
	times := 100000
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			c := make(chan *Request, 1)
			key := fmt.Sprintf("int%d", n)
			for j := 0; j < times; j++ {
				_, e := conn.call(c, "set", key, n)
				if e != nil {
					t.Fatal(e)
				}
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func TestConnError(t *testing.T) {
	t.Skip("skip connnection loop")
	spec := DefaultConnectionSpec()
	conn := NewConnection(spec)
	c := make(chan *Request, 1)

	go func() {
		time.Sleep(time.Second * 1)
		conn.Conn.Close()
	}()

	for i := 0; i < 3; i++ {
		fmt.Println("go", i)
		reply, err := conn.call(c, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply.Type, reply.Value)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
