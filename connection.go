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
	ErrEmptyReqests			= errors.New("redis: package has no request")
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
	waitingChan chan *RequestsPkg

	//控制消息
	ctrlChan chan ctrlType

	connected bool
	err       error
	pingTick  <-chan time.Time

	lastConnect time.Time
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
	r := NewRequstPkg()
	r.Add("PING")
	c.doPipelining(r)
}

func (c *Connection) isShutDown() bool {
	return c.stop
}

func (c *Connection) isConnected() bool {
	return c.connected
}

func (c *Connection) sendReconnectCtrl() {
	c.ctrlChan <- type_ctrl_reconnect
}

func (c *Connection) sendShutdownCtrl() {
	c.ctrlChan <- type_ctrl_shutdown
}

func (c *Connection) sendRequest(reqsPkg *RequestsPkg) {
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

func (c *Connection) recover(err error) {
	//一定时间段内只尝试重连一次
	if c.lastConnect.Add(c.ReconnectInterval).After(time.Now()) {
		return
	}

	c.lastConnect = time.Now()
	c.connect()
}

func (c *Connection) handleCtrl(ctrltype ctrlType) {
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
		case ctrl := <-c.ctrlChan:
			c.handleCtrl(ctrl)
		case reqs := <-c.waitingChan:
			c.sendRequest(reqs)
		case <-c.pingTick:
			c.ping()
		}
	}
}

func (c *Connection) doPipelining(reqs *RequestsPkg) {
	c.writeAllRequst(reqs)
	c.readAllReply(reqs)
}

func (c *Connection) writeAllRequst(reqs *RequestsPkg) {
	for _, req := range reqs.requests {
		writeReqToBuf(c.writeBuffer, req)
	}
	c.writeBuffer.Flush()
}

func (c *Connection) readAllReply(reqs *RequestsPkg) {
	for _, req := range reqs.requests {
		req.Reply = readReply(c.readBuffer)
	}
	reqs.done()
}

func NewConnection(spec ConnectionSpec, c chan *RequestsPkg) (conn *Connection) {
	conn = &Connection{
		stop:           false,
		connected:      false,
		ConnectionSpec: spec,
		waitingChan:    c,
		pingTick:       time.Tick(spec.PingInterval),
		lastConnect:    time.Now(),
	}

	conn.connect()

	go conn.process()

	return
}
