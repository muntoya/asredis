package conn

import (
	"time"
	"net"
	"bufio"
	"bytes"
	"fmt"
	"github.com/muntoya/asredis/common"
)

type Connection struct {
	conn 		net.Conn
	readBuffer	*bufio.Reader
	timeout		time.Duration
	writeBuf    *bytes.Buffer
	Network		string
	Addr		string
}

func (conn Connection) String() string {
	return fmt.Sprintf("%s %s", conn.Network, conn.Addr)
}

func DialTimeout(network, addr string, timeout time.Duration) (*Connection, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	return &Connection{
		conn:          conn,
		readBuffer:    bufio.NewReader(conn),
		timeout:       timeout,
		writeBuf:      bytes.NewBuffer(make([]byte, 0, 128)),
		Network:       network,
		Addr:          addr,
	}, nil
}
