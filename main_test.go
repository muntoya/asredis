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
