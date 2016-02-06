package asredis

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"
	"errors"
	"runtime/debug"
	//"fmt"
	"fmt"
)

var (
	ErrNotConnected = errors.New("redis: not connected")
	ErrNotRunning = errors.New("redis: shutdown and can't use any more")
	ErrUnexpectedCtrlType = errors.New("redis: can't process control command")
)

const (
	connectTimeout    time.Duration = time.Second * 1
	intervalReconnect time.Duration = time.Second * 1
	intervalPing      time.Duration = time.Second * 1
)

type Reply struct {
	Type  ResponseType
	Value interface{}
	Array []interface{}
}

type Request struct {
	cmd   string
	args  []interface{}
	err   error
	reply *Reply
	Done  chan *Request
}

func (r *Request) done() {
	if r.Done != nil {
		r.Done <- r
	}
}

func (r *Request) GetReply() (*Reply, error) {
	<-r.Done
	return r.reply, r.err
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
	addr        string
	stop        bool

	conMutex sync.Mutex
	reqMutex sync.Mutex

	//等待接收回复的请求
	reqsPending chan *Request

	ctrlChan  chan ctrlType
	connected bool
	err       error
	pingTick  <-chan time.Time

	lastConnect time.Time
}

func (c *Connection) connect() {
	c.connected = false

	var err error
	c.Conn, err = net.DialTimeout("tcp", c.addr, connectTimeout)
	if err != nil {
		c.err = err
		log.Printf("can't connect to redis %v, error:%v", c.addr, err)
		return
	}

	c.readBuffer = bufio.NewReader(c.Conn)
	c.writeBuffer = bufio.NewWriter(c.Conn)

	c.connected = true
	c.err = nil
}

func (c *Connection) close() {
	c.conMutex.Lock()
	defer c.conMutex.Unlock()

	c.sendShutdownCtrl()
	c.stop = true
	c.connected = false
}

func (c *Connection) ping() {
	c.asyncCall(nil, "PING")
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

func (c *Connection) send(req *Request) {
	writeReqToBuf(c.writeBuffer, req)
}

func (c *Connection) asyncCall(done chan *Request, cmd string, args ...interface{}) *Request {
	req := newRequst(done, cmd, args...)
	c.sendRequest(req, false)
	return req
}

func (c *Connection) call(done chan *Request, cmd string, args ...interface{}) (*Reply, error) {
	req := c.asyncCall(done, cmd, args...)
	return req.GetReply()
}

func (c *Connection) sendRequest(req *Request, onlySend bool) {

	c.conMutex.Lock()
	defer func() {
		if err := recover(); err != nil {
			req.err = err.(error)
			req.done()
			c.connected = false
			c.sendReconnectCtrl()
			fmt.Println(string(debug.Stack()))
		}
		c.conMutex.Unlock()
	}()

	if c.isShutDown() {
		panic(ErrNotRunning)
	}

	if !c.isConnected() {
		panic(ErrNotConnected)
	}

	c.send(req)
	if !onlySend {
		c.reqsPending <- req
	}
}

func (c *Connection) pubsubWait(done chan *Request) (*Reply, error) {
	req := newRequst(done, "")
	c.reqsPending <- req
	return req.GetReply()
}

func (c *Connection) pubsubSend(cmd string, args ...interface{}) error {
	req := newRequst(nil, cmd, args...)
	c.sendRequest(req, true)
	return req.err
}

func (c *Connection) recover(err error) {
	c.conMutex.Lock()
	defer c.conMutex.Unlock()

	//一定时间段内只尝试重连一次
	if c.lastConnect.Add(intervalReconnect).After(time.Now()) {
		return
	}

	c.lastConnect = time.Now()
	c.clear(err)
	c.connect()
}

//清空等待的请求
func (c *Connection) clear(err error) {
	close(c.reqsPending)
	for req := range c.reqsPending {
		req.err = err
		req.done()
	}
	c.reqsPending = make(chan *Request, 100)
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
		case req := <-c.reqsPending:
			c.read(req)
		case <-c.pingTick:
			c.ping()
		}
	}
}

func (c *Connection) read(req *Request) {
	if c.isShutDown() {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			req.err = e
			c.recover(e)
			fmt.Println(string(debug.Stack()))
		}

		req.done()
	}()

	req.reply = readReply(c.readBuffer)
}

func NewConnection(addr string) (client *Connection) {
	client = &Connection{
		addr:        addr,
		stop:        false,
		connected:   false,
		//FIXME: 测试chan的个数
		reqsPending: make(chan *Request, 200),
		ctrlChan:    make(chan ctrlType, 10),
		pingTick:    time.Tick(intervalPing),
		lastConnect: time.Now(),
	}

	client.connect()

	go client.process()

	return
}

func newRequst(done chan *Request, cmd string, args ...interface{}) *Request {
	req := new(Request)

	if done != nil {
		if cap(done) == 0 {
			log.Panic("redis client: done channel is unbuffered")
		}
		req.Done = done
	}

	req.cmd = cmd
	req.args = args
	return req
}
