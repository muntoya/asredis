package asredis

import (
	"bufio"
	"container/list"
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
	defaultPipeliningSize    int           = 20
	defaultCommandTimeout    time.Duration = time.Millisecond
	defaultReconnectInterval time.Duration = time.Second
	defaultPingInterval      time.Duration = time.Second
	defaultWaitingChanSize   int           = 100
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
	PipeliningSize    int
	CommandTimeout    time.Duration
	ReconnectInterval time.Duration
	PingInterval      time.Duration
	WaitingChanSize   int
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
		defaultPipeliningSize,
		defaultCommandTimeout,
		defaultReconnectInterval,
		defaultPingInterval,
		defaultWaitingChanSize,
	}
}

type Connection struct {
	net.Conn
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer
	connSpec    *ConnectionSpec
	stop        bool

	//等待送入发送线程的请求
	waitingChan chan []*Request

	//等待发送的请求
	reqsPending *list.List

	//发送超时信号的channel
	sendTime <-chan time.Time

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
	c.Conn, err = createTCPConnection(c.connSpec)
	if err != nil {
		c.err = err
		log.Printf("can't connect to redis %v, error:%v", c.connSpec.Host, err)
		return
	}

	c.readBuffer = bufio.NewReaderSize(c.Conn, c.connSpec.IOReadBufSize)
	c.writeBuffer = bufio.NewWriterSize(c.Conn, c.connSpec.IOWriteBufSize)

	c.connected = true
	c.err = nil
}

func (c *Connection) close() {
	c.sendShutdownCtrl()
	c.connected = false
}

func (c *Connection) ping() {
	c.pushRequst(type_normal, nil, "PING")
}

func (c *Connection) isShutDown() bool {
	return c.stop
}

func (c *Connection) isConnected() bool {
	return c.connected
}

func (c *Connection) sendReconnectCtrl() {
	req := NewRequestType(type_ctrl_reconnect, nil, "")
	c.pushRequst(req)
}

func (c *Connection) sendShutdownCtrl() {
	req := NewRequestType(type_ctrl_shutdown, nil, "")
	c.pushRequst(req)
}

func (c *Connection) handleRequetList(f func(*Request)) {
	for e := c.reqsPending.Front(); e != nil; e = e.Next() {
		req := e.Value.(*Request)
		f(req)
	}
}

func (c *Connection) pushRequst(req... *Request) {
	c.waitingChan <- req
	for _, r := range req {
		r.wait()
	}
}

func (c *Connection) sendRequest(req *Request) {
	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			req.err = e
			c.recover(e)
			fmt.Println(string(debug.Stack()))
			fmt.Println(req.cmd, req.reqtype)
		}
	}()

	if req.reqtype > type_ctrl_begin && req.reqtype < type_ctrl_end {
		c.handleCtrl(req.reqtype)
		return
	}

	if c.reqsPending.Len() == 0 {
		c.sendTime = time.After(c.connSpec.CommandTimeout)
	}
	c.reqsPending.PushBack(req)

	if c.isShutDown() {
		panic(ErrNotRunning)
	}

	if !c.isConnected() {
		panic(ErrNotConnected)
	}

	if c.reqsPending.Len() < c.connSpec.PipeliningSize {
		return
	}

	c.doPipelining()
}

func (c *Connection) pubsubWait(done chan *Request) (*Reply, error) {
	req := NewRequestPubsubWait(done)
	c.pushRequst(req)
	return req.GetReply()
}

func (c *Connection) pubsubSend(cmd string, args ...interface{}) error {
	req := NewRequestPubsubSend(cmd, args...)
	c.pushRequst(req)
	_, err := req.GetReply()
	return err
}

func (c *Connection) recover(err error) {
	//一定时间段内只尝试重连一次
	if c.lastConnect.Add(c.connSpec.ReconnectInterval).After(time.Now()) {
		return
	}

	c.lastConnect = time.Now()
	c.clear(err)
	c.connect()
}

//清空等待的请求
func (c *Connection) clear(err error) {
	c.handleRequetList(func(r *Request) {
		r.done()
	})

	c.reqsPending.Init()
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
		c.clear(ErrNotRunning)
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
		case <-c.sendTime:
			c.doPipelining()
		case req := <-c.waitingChan:
			c.sendRequest(req)
		case <-c.pingTick:
			c.ping()
		}
	}
}

func (c *Connection) doPipelining() {
	c.writeAllRequst()
	c.readAllReply()
	c.reqsPending.Init()
}

func (c *Connection) writeAllRequst() {
	c.handleRequetList(func(req *Request) {
		switch req.reqtype {
		case type_normal:
			fallthrough
		case type_only_send:
			writeReqToBuf(c.writeBuffer, req)
		case type_only_wait:
		default:

		}
	})
	c.writeBuffer.Flush()
}

func (c *Connection) readAllReply() {
	c.handleRequetList(func(req *Request) {
		switch req.reqtype {
		case type_normal:
			fallthrough
		case type_only_wait:
			reply := readReply(c.readBuffer)
			req.reply = reply
		case type_only_send:
		default:

		}
	})
	c.handleRequetList(func(req *Request) {
		req.done()
	})
}

func NewConnection(spec *ConnectionSpec) (conn *Connection) {
	conn = &Connection{
		stop:        false,
		connected:   false,
		connSpec:    spec,
		reqsPending: list.New(),
		waitingChan: make(chan *Request, spec.WaitingChanSize),
		pingTick:    time.Tick(spec.PingInterval),
		lastConnect: time.Now(),
	}

	conn.connect()

	go conn.process()

	return
}
