package conn

import (
	"bufio"
	"fmt"
	"net"
	"time"
	"github.com/muntoya/asredis/common"
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

func (this *Connection) readToCRLF() []byte {
	//	var buf []byte
	buf, e := this.readBuffer.ReadBytes(cr_byte)
	if e != nil {
		panic(common.NewRedisErrorf("readToCRLF - ReadBytes", e))
	}

	var b byte
	b, e = this.readBuffer.ReadByte()
	if e != nil {
		panic(common.NewRedisErrorf("readToCRLF - ReadByte", e))
	}
	if b != lf_byte {
		e = common.NewRedisError("<BUG> Expecting a Linefeed byte here!")
	}
	return buf[0 : len(buf)-1]
}

func (this *Connection) send(str string) {
	this.writeBuffer.WriteString(str)
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
