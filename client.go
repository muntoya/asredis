package asredis

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	//	"runtime/debug"
)

const (
	intervalReconnect time.Duration = time.Second * 2
)

type Reply struct {
	Type  ResponseType
	Value interface{}
	Array []interface{}
}

type RequestInfo struct {
	cmd   string
	args  []interface{}
	err   error
	reply *Reply
	Done  chan *RequestInfo
}

func (this *RequestInfo) done() {
	this.Done <- this
}

func (this *RequestInfo) GetReply() (*Reply, error) {
	<-this.Done
	return this.reply, this.err
}

type ctrlType byte

const (
	ctrlReconnect ctrlType = iota
	ctrlShutdown
)

type Client struct {
	net.Conn
	readBuffer  *bufio.Reader
	writeBuffer *bufio.Writer
	Network     string
	Addr        string

	conMutex sync.Mutex
	reqMutex sync.Mutex

	//等待接收回复的请求
	reqsPending chan *RequestInfo

	ctrlChan  chan ctrlType
	connected bool
	err       error

	lastConnect time.Time
}

func (this *Client) String() string {
	return fmt.Sprintf("%s %s", this.Network, this.Addr)
}

func (this *Client) Connect() {
	var err error
	this.Conn, err = net.DialTimeout(this.Network, this.Addr, time.Second*1)
	if err != nil {
		this.connected = false
		this.err = err
		log.Fatalf("can't connect to redis %v:%v, error:%v", this.Network, this.Addr, err)
		return
	}

	this.readBuffer = bufio.NewReader(this.Conn)
	this.writeBuffer = bufio.NewWriter(this.Conn)

	this.connected = true
	this.err = nil
}

func (this *Client) Close() {
	if this.Conn != nil {
		this.Conn.Close()
	}
	this.connected = false
}

func (this *Client) Ping()

func (this *Client) IsConnected() bool {
	return this.connected
}

func (this *Client) sendReconnectCtrl() {
	this.ctrlChan <- ctrlReconnect
}

func (this *Client) send(req *RequestInfo) {
	writeReqToBuf(this.writeBuffer, req)
}

func (this *Client) Go(done chan *RequestInfo, cmd string, args ...interface{}) *RequestInfo {
	req := new(RequestInfo)

	if done == nil {
		done = make(chan *RequestInfo, 1)
	} else {
		if cap(done) == 0 {
			log.Panic("redis client: done channel is unbuffered")
		}
	}

	req.cmd = cmd
	req.args = args
	req.Done = done

	this.SendRequest(req)

	return req
}

func (this *Client) SendRequest(req *RequestInfo) {
	this.conMutex.Lock()
	defer func() {
		// FIXME: 可能需要将恢复放到pool中
		if err := recover(); err != nil {
			req.err = err.(error)
			req.done()
			this.connected = false
			this.sendReconnectCtrl()
		}
		this.conMutex.Unlock()
	}()

	if !this.IsConnected() {
		panic(ErrNotConnected)
	}

	this.send(req)
	this.reqsPending <- req
}

func (this *Client) recover(err error) {
	this.conMutex.Lock()
	defer this.conMutex.Unlock()

	//一定时间段内只尝试重连一次
	if this.lastConnect.Add(intervalReconnect).After(time.Now()) {
		return
	}

	this.lastConnect = time.Now()

	//清空等待的请求
	close(this.reqsPending)
	for req := range this.reqsPending {
		req.err = err
		req.done()
	}

	this.reqsPending = make(chan *RequestInfo, 100)
	this.Connect()
}

func (this *Client) control(ctrl ctrlType) {
	switch ctrl {
	case ctrlReconnect:
		this.recover(ErrNotConnected)
	default:
		log.Panic(ErrUnexpectedCtrlType)
	}
}

//处理读请求和控制请求
func (this *Client) process() {
	for {
		select {
		case ctrl := <-this.ctrlChan:
			this.control(ctrl)
		case req := <-this.reqsPending:
			this.read(req)
		}
	}
}

func (this *Client) read(req *RequestInfo) {
	defer func() {
		if err := recover(); err != nil {
			e := err.(error)
			req.err = e
			this.recover(e)
		}

		req.done()
	}()

	req.reply = readReply(this.readBuffer)
}

func NewClient(network, addr string) (client *Client) {
	client = &Client{
		Network:     network,
		Addr:        addr,
		connected:   false,
		reqsPending: make(chan *RequestInfo, 100),
		ctrlChan:    make(chan ctrlType, 10),
		lastConnect: time.Now(),
	}

	client.Connect()

	go client.process()

	return client
}
