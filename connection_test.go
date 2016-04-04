package asredis

import (
	"time"
	//	"runtime/debug"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type ConnectionTestSuite struct {
	suite.Suite

	Conn *Connection
}

func (s *ConnectionTestSuite) SetupSuite() {
	spec := DefaultConnectionSpec()
	reqChan := make(chan *RequestsPkg, 10)
	s.Conn = NewConnection(*spec, reqChan)
}

func (s *ConnectionTestSuite) TearDownSuite() {
	fmt.Println("close")
	s.Conn.Close()
}

func (s *ConnectionTestSuite) SetupTest() {
}

func (s *ConnectionTestSuite) TearDownTest() {

}

func Do(conn *Connection, cmd string, args ...interface{}) (interface{}, error) {
	r := NewRequstPkg()
	r.Add(cmd, args...)
	conn.waitingChan <- r
	r.wait()
	return r.reply(), r.err()
}

func (s *ConnectionTestSuite) TestConnection() {
	var testCommands = []struct {
		args     []interface{}
		expected interface{}
	}{
		{
			[]interface{}{"SET", "int", 2},
			"OK",
		},
		{
			[]interface{}{"GET", "int"},
			2,
		},
		{
			[]interface{}{"RPUSH", "list", "1", "2", "3", "4", "5"},
			[]interface{}{"1", "2", "3", "4", "5"},
		},
	}

	t := s.T()
	//t.Skip("skip connection test")

	for _, command := range testCommands {
		var reply interface{}
		var err error
		reply, err = Do(s.Conn, command.args[0].(string), command.args[1:]...)
		assert.Nil(t, err)
		assert.Equal(t, command.expected, reply)
	}
}

func (s *ConnectionTestSuite) TestConnRoutine() {
	t := s.T()
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
				_, e := Do(s.Conn, "set", key, n)
				if e != nil {
					t.Fatal(e)
				}
			}
			w.Done()
		}(i)
	}
	w.Wait()
}

func (s *ConnectionTestSuite) TestConnError() {
	t := s.T()
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
		reply, err := Do(s.Conn, "GET", "int")
		fmt.Println(i, err)
		if err == nil {
			t.Log(reply)
		} else {
			t.Log(err)
		}
		time.Sleep(time.Second * 1)
	}
}

func TestRunSuite(t *testing.T) {
	suiteTester := new(ConnectionTestSuite)
	suite.Run(t, suiteTester)
}
