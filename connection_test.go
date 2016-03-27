package asredis

import (
	"time"
	//	"runtime/debug"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func call(c *Connection, cmd string, args...interface{}) (*Reply, error) {
	r := NewRequstPkg()
	r.Add(cmd, args...)
	c.waitingChan <- r
	r.wait()
	return r.reply(), r.err()
}

func TestConnection(t *testing.T) {
	//t.Skip("skip connection test")
	spec := DefaultConnectionSpec()
	reqChan := make(chan *RequestsPkg, 10)
	conn := NewConnection(*spec, reqChan)
	defer conn.close()


	var reply *Reply
	var err error
	reply, err = call(conn, "SET", "int", 2)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")

	reply, err = call(conn, "GET", "int")
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(reply.Type, reply.Value)
	}

	l := []interface{}{"1", "2", "3", "4", "5"}
	call(conn, "DEL", "list")
	reply, err = call(conn, "RPUSH", append([]interface{}{"list"}, l...)...)
	reply, err = call(conn, "LRANGE", "list", 0, -1)
	assert.Equal(t, reply.Array, l)
}

func TestConnRoutine(t *testing.T) {
	//t.Skip("skip connection routine")
	spec := DefaultConnectionSpec()
	reqChan := make(chan *RequestsPkg, 10)
	conn := NewConnection(*spec, reqChan)
	defer conn.close()

	routineNum := 80
	times := 100
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			key := fmt.Sprintf("int%d", n)
			for j := 0; j < times; j++ {
				_, e := call(conn, "set", key, n)
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
	reqChan := make(chan *RequestsPkg, 10)
	conn := NewConnection(*spec, reqChan)
	defer conn.close()

	go func() {
		time.Sleep(time.Second * 1)
		conn.Conn.Close()
	}()

	for i := 0; i < 3; i++ {
		fmt.Println("go", i)
		reply, err := call(conn, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply.Type, reply.Value)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
