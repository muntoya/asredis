package asredis

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime/debug"
	"time"
)

var (
	ErrNotConnected       = errors.New("redis: not connected")
	ErrNotRunning         = errors.New("redis: shutdown and can't use any more")
	ErrUnexpectedCtrlType = errors.New("redis: can't process control command")
)

const (
	defaultSendTimeout       time.Duration = time.Millisecond
	defaultHost              string        = "127.0.0.1"
	defaultPort              int16         = 6379
	defaultPassword          string        = ""
	defaultDB                int           = 0
	defaultTCPConnectTimeout time.Duration = time.Second
	defaultTCPReadBufSize    int           = 1024 * 256
	defaultTCPWriteBufSize   int           = 1024 * 256
	defaultTCPReadTimeout    time.Duration = time.Millisecond
	defaultTCPWriteTimeout   time.Duration = time.Millisecond
	defaultTCPLinger                       = 0 // -n: finish io; 0: discard, +n: wait for n secs to finish
	defaultTCPKeepalive                    = true
	defaultIOReadBufSize     int           = 1024 * 256
	defaultIOWriteBufSize    int           = 1024 * 256
	defaultCommandTimeout    time.Duration = time.Millisecond
	defaultReconnectInterval time.Duration = time.Second
	defaultPingInterval      time.Duration = time.Second
)

type ConnectionSpec struct {
	Host              string
	Port              int16
	Password          string
	DB                int
	TCPConnectTimeout time.Duration
	TCPReadBufSize    int
	TCPWritBufSize    int
	TCPReadTimeout    time.Duration
	TCPWriteTimeout   time.Duration
	TCPLinger         int
	TCPKeepalive      bool
	IOReadBufSize     int
	IOWriteBufSize    int
	CommandTimeout    time.Duration
	ReconnectInterval time.Duration
	PingInterval      time.Duration
}

func DefaultConnectionSpec() *ConnectionSpec {
	return &ConnectionSpec{
		defaultHost,
		defaultPort,
		defaultPassword,
		defaultDB,
		defaultTCPConnectTimeout,
		defaultTCPReadBufSize,
		defaultTCPWriteBufSize,
		defaultTCPReadTimeout,
		defaultTCPWriteTimeout,
		defaultTCPLinger,
		defaultTCPKeepalive,
		defaultIOReadBufSize,
		defaultIOWriteBufSize,
		defaultCommandTimeout,
		defaultReconnectInterval,
		defaultPingInterval,
	}
}

type Connection struct {
	net.Conn
	ConnectionSpec
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer
	stop        bool

	//等待送入发送线程的请求
	waitingChan chan *requestsPkg

	connected bool
	err       error
	pingTick  <-chan time.Time

	lastConnect time.Time

	pubsubChan chan struct{}
}

func createTCPConnection(s *ConnectionSpec) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	conn, err := net.DialTimeout("tcp", addr, s.TCPConnectTimeout)
	if err != nil {
		return nil, err
	}

	c := conn.(*net.TCPConn)
	c.SetKeepAlive(s.TCPKeepalive)
	c.SetReadBuffer(s.TCPReadBufSize)
	c.SetWriteBuffer(s.TCPWritBufSize)
	c.SetLinger(s.TCPLinger)

	return conn, nil
}

func (c *Connection) connect() {
	c.connected = false

	var err error
	c.Conn, err = createTCPConnection(&c.ConnectionSpec)
	if err != nil {
		c.err = err
		log.Printf("can't connect to redis %v, error:%v", c.Host, err)
		return
	}

	c.readBuffer = bufio.NewReaderSize(c.Conn, c.IOReadBufSize)
	c.writeBuffer = bufio.NewWriterSize(c.Conn, c.IOWriteBufSize)

	c.connected = true
	c.err = nil
}

func (c *Connection) close() {
	c.sendShutdownCtrl()
	c.connected = false
}

func (c *Connection) ping() {
	req := NewRequest("PING")
	c.pushRequst(nil, req)
}

func (c *Connection) isShutDown() bool {
	return c.stop
}

func (c *Connection) isConnected() bool {
	return c.connected
}

func (c *Connection) sendReconnectCtrl() {
	req := NewRequestType(type_ctrl_reconnect, "")
	c.pushRequst(nil, req)
}

func (c *Connection) sendShutdownCtrl() {
	req := NewRequestType(type_ctrl_shutdown, "")
	c.pushRequst(nil, req)
}

func (c *Connection) pushRequst(done chan struct{}, reqs ...*Request) {
	reqsPkg := &requestsPkg{reqs, done}
	if done != nil {
		if cap(done) == 0 {
			log.Panic("redis client: done channel is unbuffered")
		}
		reqsPkg.Done = done
	}

	c.waitingChan <- reqs
	reqsPkg.wait()
}

func (c *Connection) sendRequest(reqsPkg *requestsPkg) {
	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			for _, req := range reqsPkg.requests {
				req.Err = e
			}
			reqsPkg.done()
			c.recover(e)
			fmt.Println(string(debug.Stack()))
		}
	}()

	if c.isShutDown() {
		panic(ErrNotRunning)
	}

	if !c.isConnected() {
		panic(ErrNotConnected)
	}

	c.doPipelining(reqsPkg)
}

func (c *Connection) pubsubWait() {
	req := NewRequestType(type_only_wait, "")
	c.pushRequst(c.pubsubChan, req)
}

func (c *Connection) pubsubSend(cmd string, args ...interface{}) {
	req := NewRequestType(type_only_send, cmd, args...)
	c.pushRequst(nil, req)
}

func (c *Connection) recover(err error) {
	//一定时间段内只尝试重连一次
	if c.lastConnect.Add(c.ReconnectInterval).After(time.Now()) {
		return
	}

	c.lastConnect = time.Now()
	c.connect()
}

func (c *Connection) handleCtrl(ctrltype requestType) {
	switch ctrltype {
	case type_ctrl_reconnect:
		c.recover(ErrNotConnected)
	case type_ctrl_shutdown:
		c.stop = true
		if c.Conn != nil {
			c.Conn.Close()
		}
	default:

	}
}

//处理读请求和控制请求
func (c *Connection) process() {
	for {
		if c.stop {
			break
		}

		select {
		case reqs := <-c.waitingChan:
			c.sendRequest(reqs)
		case <-c.pingTick:
			c.ping()
		}
	}
}

func (c *Connection) doPipelining(reqs *requestsPkg) {
	c.writeAllRequst(reqs)
	c.readAllReply(reqs)
}

func (c *Connection) writeAllRequst(reqs *requestsPkg) {
	for _, req := range reqs.requests {
		switch req.reqtype {
		case type_normal:
			fallthrough
		case type_only_send:
			writeReqToBuf(c.writeBuffer, req)
		case type_only_wait:
		default:

		}
	}
	c.writeBuffer.Flush()
}

func (c *Connection) readAllReply(reqs *requestsPkg) {
	for _, req := range reqs.requests {
		switch req.reqtype {
		case type_normal:
			fallthrough
		case type_only_wait:
			reply := readReply(c.readBuffer)
			req.Reply = reply
		case type_only_send:
		default:

		}
	}

	reqs.done()
}

func NewConnection(spec ConnectionSpec, c chan *requestsPkg) (conn *Connection) {
	conn = &Connection{
		stop:           false,
		connected:      false,
		ConnectionSpec: spec,
		waitingChan:    c,
		pingTick:       time.Tick(spec.PingInterval),
		lastConnect:    time.Now(),
		pubsubChan:		make(chan struct{}, 1),
	}

	conn.connect()

	go conn.process()

	return
}
