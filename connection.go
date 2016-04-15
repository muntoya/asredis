package asredis

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	//"runtime/debug"
	"time"
)

var (
	ErrNotConnected       = errors.New("redis: not connected")
	ErrNotRunning         = errors.New("redis: shutdown and can't use any more")
	ErrUnexpectedCtrlType = errors.New("redis: can't process control command")
	ErrEmptyReqests       = errors.New("redis: package has no request")
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

type Spec struct {
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

func DefaultSpec() *Spec {
	return &Spec{
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
	addr string
	net.Conn
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer

	//等待送入发送线程的请求
	waitingChan chan *RequestsPkg

	pingTick <-chan time.Time
}

func createTCPConnection(s *Spec, addr string) (net.Conn, error) {
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

func (c *Connection) connect(spec *Spec) (err error) {
	c.Conn, err = createTCPConnection(spec, c.addr)
	if err != nil {
		log.Printf("can't connect to redis %v, error:%v", c.addr, err)
		return
	}
	c.readBuffer = bufio.NewReaderSize(c.Conn, spec.IOReadBufSize)
	c.writeBuffer = bufio.NewWriterSize(c.Conn, spec.IOWriteBufSize)
	return
}

func (c *Connection) ping() {
	r := NewRequstPkg()
	r.Add("PING")
}

func (c *Connection) sendRequest(reqsPkg *RequestsPkg) {
	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			for _, req := range reqsPkg.requests {
				req.Err = e
			}
			reqsPkg.done()
			//fmt.Println(string(debug.Stack()))
		}
	}()

	c.doPipelining(reqsPkg)
}

//处理读请求和控制请求
func (c *Connection) process(p *Pool) {
	defer p.stopWg.Done()

	c.pingTick = time.Tick(p.PingInterval)
	for {
		dialChan := make(chan struct{})
		c.connect(p.Spec)

		select {
		case p.stopChan:
			return
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
		req.Reply, req.Err = readReply(c.readBuffer)
	}
	reqs.done()
}
