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
	connectTimeout     time.Duration = time.Second * 1
	intervalReconnect  time.Duration = time.Second * 1
	intervalPing       time.Duration = time.Second * 1
	waitingChanLen     int           = 20
	ctrlChanLen        int           = 10
	defaultPPLen       int           = 10
	defaultSendTimeout time.Duration = time.Millisecond
)

type ctrlType byte

const (
	ctrlReconnect ctrlType = iota
	ctrlShutdown
)

type Connection struct {
	net.Conn
	timeout     time.Duration
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer
	addr        string
	stop        bool

	//等待送入发送线程的请求
	waitingChan chan *Request

	//等待发送的请求
	ppLen       int
	reqsPending *list.List

	//发送超时信号的channel
	sendTime    <-chan time.Time

	ctrlChan    chan ctrlType
	connected   bool
	err         error
	pingTick    <-chan time.Time

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
	c.sendShutdownCtrl()
	c.stop = true
	c.connected = false
}

func (c *Connection) ping() {
	c.call(nil, "PING")
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
	req := newRequst(NORMAL, done, cmd, args...)
	c.waitingChan <- req
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
		c.sendTime = time.After(c.timeout)
	}
	c.reqsPending.PushBack(req)

	if c.isShutDown() {
		panic(ErrNotRunning)
	}

	if !c.isConnected() {
		panic(ErrNotConnected)
	}

	fmt.Println("send")
	if c.reqsPending.Len() < c.ppLen {
		return
	}
	fmt.Println("send2")

	c.doPipelining()
}

func (c *Connection) pubsubWait(done chan *Request) (*Reply, error) {
	req := newRequst(ONLY_WAIT, done, "")
	c.waitingChan <- req
	return req.GetReply()
}

func (c *Connection) pubsubSend(cmd string, args ...interface{}) error {
	req := newRequst(ONLY_SEND, nil, cmd, args...)
	c.waitingChan <- req
	return req.err
}

func (c *Connection) recover(err error) {
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
		case <- c.sendTime:
			c.doPipelining()
		case req := <-c.waitingChan:
			c.sendRequest(req)
		case <-c.pingTick:
			c.ping()
		}
	}
}

func (c *Connection) doPipelining() {
	fmt.Println("pipelining")
	c.writeAllRequst()
	c.readAllReply()
	c.reqsPending.Init()
}

func (c *Connection) writeAllRequst() {
	c.handleRequetList(func(req *Request) {
		writeReqToBuf(c.writeBuffer, req)
	})
}

func (c *Connection) readAllReply() {
	c.handleRequetList(func(req *Request) {
		reply := readReply(c.readBuffer)
		req.reply = reply
		req.done()
	})
}

func NewConnection(addr string, plLengh int, timeout time.Duration) (client *Connection) {
	client = &Connection{
		addr:        addr,
		stop:        false,
		connected:   false,
		ppLen:       plLengh,
		timeout:     timeout,
		reqsPending: list.New(),
		waitingChan: make(chan *Request, waitingChanLen),
		ctrlChan:    make(chan ctrlType, ctrlChanLen),
		pingTick:    time.Tick(intervalPing),
		lastConnect: time.Now(),
	}

	client.connect()

	go client.process()

	return
}

func newRequst(reqtype requestType, done chan *Request, cmd string, args ...interface{}) *Request {
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
	return req
}
