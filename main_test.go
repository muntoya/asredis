package asredis

import (
	"os"
	"testing"
)

var Conn *Connection

func TestMain(m *testing.M) {
	SetupSuite()
	ret := m.Run()
	TearDownSuite()
	os.Exit(ret)
}

func SetupSuite() {
	spec := DefaultConnectionSpec()
	reqChan := make(chan *RequestsPkg, 10)
	Conn = NewConnection(*spec, reqChan)
}

func TearDownSuite() {
	Conn.Close()
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
