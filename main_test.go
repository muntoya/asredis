package asredis

import (
	"os"
	"testing"
)

var testConn *Connection

func TestMain(m *testing.M) {
	SetupSuite()
	ret := m.Run()
	TearDownSuite()
	os.Exit(ret)
}

func SetupSuite() {
	spec := DefaultSpec()
	reqChan := make(chan *RequestsPkg, 10)
	testConn = newConnection(spec, reqChan)
}

func TearDownSuite() {
	testConn.Close()
}

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
