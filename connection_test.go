package asredis

import (
	"time"
	//	"runtime/debug"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)




func do(conn *Connection, cmd string, args ...interface{}) (interface{}, error) {
	r := NewRequstPkg()
	r.Add(cmd, args...)
	conn.waitingChan <- r
	r.wait()
	return r.reply(), r.err()
}

func TestConnection(t *testing.T) {
	var testCommands = []struct {
		args     []interface{}
		expected interface{}
	}{
		{
			[]interface{}{"SET", "int", "2"},
			"OK",
		},
		{
			[]interface{}{"GET", "int"},
			"2",
		},
		{
			[]interface{}{"DEL", "int"},
			1,
		},
		{
			[]interface{}{"DEL", "list"},
			0,
		},
		{
			[]interface{}{"RPUSH", "list", "1", "2", "3", "4", "5"},
			5,
		},
		{
			[]interface{}{"LRANGE", "list", 0, -1},
			[]interface{}{"1", "2", "3", "4", "5"},
		},
		{
			[]interface{}{"DEL", "list"},
			1,
		},
	}

	//t.Skip("skip connection test")

	for _, command := range testCommands {
		var reply interface{}
		var err error
		reply, err = do(Conn, command.args[0].(string), command.args[1:]...)
		t.Log(command)
		assert.Nil(t, err)
		assert.Equal(t, command.expected, reply)
	}
}

func TestConnRoutine(t *testing.T) {
	return
	//t.Skip("skip connection routine")
	spec := DefaultConnectionSpec()
	reqChan := make(chan *RequestsPkg, 10)
	conn := NewConnection(*spec, reqChan)
	defer conn.Close()

	routineNum := 80
	times := 10
	var w sync.WaitGroup
	w.Add(routineNum)
	for i := 0; i < routineNum; i++ {
		go func(n int) {
			key := fmt.Sprintf("int%d", n)
			for j := 0; j < times; j++ {
				_, e := do(Conn, "set", key, n)
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
	defer conn.Close()

	go func() {
		time.Sleep(time.Second * 1)
		conn.Conn.Close()
	}()

	for i := 0; i < 3; i++ {
		fmt.Println("go", i)
		reply, err := do(Conn, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}
