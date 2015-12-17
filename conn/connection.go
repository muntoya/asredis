package conn

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type Connection struct {
	conn        net.Conn
	readBuffer  *bufio.Reader
	timeout     time.Duration
	writeBuffer *bufio.Writer
	Network     string
	Addr        string
}

func (this *Connection) String() string {
	return fmt.Sprintf("%s %s", this.Network, this.Addr)
}

func (this *Connection) Close() error {
	return this.conn.Close()
}

func (this *Connection) send(b []byte) {
	this.writeBuffer.Write(b)
	this.writeBuffer.Flush()
}

func DialTimeout(network, addr string, timeout time.Duration) (*Connection, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	return &Connection{
		conn:        conn,
		readBuffer:  bufio.NewReader(conn),
		timeout:     timeout,
		writeBuffer: bufio.NewWriter(conn),
		Network:     network,
		Addr:        addr,
	}, nil
}
