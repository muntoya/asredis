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
	defaultSendTimeout     time.Duration = time.Millisecond
	defaultHost            string        = "127.0.0.1"
	defaultPort            int           = 6379
	defaultPassword        string        = ""
	defaultDB              int           = 0
	defaultTCPConnectTimeout time.Duration = time.Second
	defaultTCPReadBufSize  int           = 1024 * 256
	defaultTCPWriteBufSize int           = 1024 * 256
	defaultTCPReadTimeout  time.Duration = time.Millisecond
	defaultTCPWriteTimeout time.Duration = time.Millisecond
	defaultTCPLinger                     = 0 // -n: finish io; 0: discard, +n: wait for n secs to finish
	defaultTCPKeepalive                  = true
	defaultIOReadBufSize   int           = 1024 * 256
	defaultIOWriteBufSize  int           = 1024 * 256
	defaultPipeliningSize	int		= 40
	defaultCommandTimeout	time.Duration = time.Millisecond
	defaultReconnectInterval	time.Duration = time.Second
	defaultPingInterval	time.Duration  = time.Second
	defaultWaitingChanSize int = 100
	defaultControlChanSize	int = 10
)

type ConnectionSpec struct {
	host            string
	port            int
	password        string
	db              int
	tcpConnectTimeout time.Duration
	tcpReadBufSize  int
	tcpWritBufSize  int
	tcpReadTimeout  time.Duration
	tcpWriteTimeout time.Duration
	tcpLinger		int
	tcpKeepalive	bool
	ioReadBufSize   int
	ioWriteBufSize  int
	pipeliningSize	int
	commandTimeout		time.Duration
	reconnectInterval time.Duration
	pingInterval	time.Duration
	waitingChanSize	int
	controlChanSize int
}

func DefaultSpec() *ConnectionSpec {
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
		defaultControlChanSize,
	}
}

type ctrlType byte

const (
	ctrlReconnect ctrlType = iota
	ctrlShutdown
)

type Connection struct {
	net.Conn
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer
	connSpec    *ConnectionSpec
	stop        bool

	//等待送入发送线程的请求
	waitingChan chan *Request

	//等待发送的请求
	reqsPending *list.List

	//发送超时信号的channel
	sendTime <-chan time.Time

	ctrlChan  chan ctrlType
	connected bool
	err       error
	pingTick  <-chan time.Time

	lastConnect time.Time
}

func createTCPConnection(s *ConnectionSpec) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	conn, err := net.DialTimeout(s.tcpConnectTimeout, addr, s.tcpConnectTimeout)
	if err != nil {
		return
	}

	c := conn.(net.TCPConn)
	c.SetKeepAlive(s.tcpKeepalive)
	c.SetReadBuffer(s.tcpReadBufSize)
	c.SetWriteBuffer(s.tcpWritBufSize)
	c.SetLinger(s.tcpLinger)

	return conn, nil
}

func (c *Connection) connect() {
	c.connected = false

	var err error
	c.Conn, err = createTCPConnection(c.connSpec)
	if err != nil {
		c.err = err
		log.Printf("can't connect to redis %v, error:%v", c.Conn.RemoteAddr().String(), err)
		return
	}

	c.readBuffer = bufio.NewReaderSize(c.Conn, c.connSpec.ioReadBufSize)
	c.writeBuffer = bufio.NewWriterSize(c.Conn, c.connSpec.ioWriteBufSize)

	c.connected = true
	c.err = nil
}

func (c *Connection) close() {
	c.sendShutdownCtrl()
	c.stop = true
	c.connected = false
}

func (c *Connection) ping() {
	c.newRequst(type_normal, nil, "PING")
}

func (c *Connection) isShutDown() bool {
	return c.stop
}

func (c *Connection) isConnected() bool {
	return c.connected
}

func (c *Connection) sendReconnectCtrl() {
	c.ctrlChan <- ctrlReconnect
}

func (c *Connection) sendShutdownCtrl() {
	c.ctrlChan <- ctrlShutdown
}

func (c *Connection) handleRequetList(f func(*Request)) {
	for e := c.reqsPending.Front(); e != nil; e = e.Next() {
		req := e.Value.(*Request)
		f(req)
	}
}

func (c *Connection) call(done chan *Request, cmd string, args ...interface{}) (*Reply, error) {
	req := c.newRequst(type_normal, done, cmd, args...)
	return req.GetReply()
}

func (c *Connection) sendRequest(req *Request) {
	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			req.err = e
			c.recover(e)
			fmt.Println(string(debug.Stack()))
		}
	}()

	if c.reqsPending.Len() == 0 {
		c.sendTime = time.After(c.connSpec.commandTimeout)
	}
	c.reqsPending.PushBack(req)

	if c.isShutDown() {
		panic(ErrNotRunning)
	}

	if !c.isConnected() {
		panic(ErrNotConnected)
	}

	if c.reqsPending.Len() < c.connSpec.pipeliningSize {
		return
	}

	c.doPipelining()
}

func (c *Connection) pubsubWait(done chan *Request) (*Reply, error) {
	req := c.newRequst(type_only_wait, done, "")
	return req.GetReply()
}

func (c *Connection) pubsubSend(cmd string, args ...interface{}) error {
	req := c.newRequst(type_only_send, nil, cmd, args...)
	return req.err
}

func (c *Connection) recover(err error) {
	//一定时间段内只尝试重连一次
	if c.lastConnect.Add(c.connSpec.reconnectInterval).After(time.Now()) {
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

func (c *Connection) control(ctrl ctrlType) {
	switch ctrl {
	case ctrlReconnect:
		c.recover(ErrNotConnected)
	case ctrlShutdown:
		c.stop = true
		if c.Conn != nil {
			c.Conn.Close()
		}
		c.clear(ErrNotRunning)
	default:
		log.Panic(ErrUnexpectedCtrlType)
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
			c.control(ctrl)
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
		case type_ctrl_reconnect:
		case type_ctrl_shutdown:
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
		case type_ctrl_reconnect:
		case type_ctrl_shutdown:
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
		connSpec:	spec,
		reqsPending: list.New(),
		waitingChan: make(chan *Request, spec.waitingChanSize),
		ctrlChan:    make(chan ctrlType, spec.controlChanSize),
		pingTick:    time.Tick(spec.pingInterval),
		lastConnect: time.Now(),
	}

	conn.connect()

	go conn.process()

	return
}

func (c *Connection) newRequst(reqtype requestType, done chan *Request, cmd string, args ...interface{}) *Request {
	req := new(Request)
	req.reqtype = reqtype

	if done != nil {
		if cap(done) == 0 {
			log.Panic("redis client: done channel is unbuffered")
		}
		req.Done = done
	}

	req.cmd = cmd
	req.args = args
	c.waitingChan <- req
	return req
}
