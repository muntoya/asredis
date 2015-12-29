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
	connectTimeout    time.Duration = time.Second * 1
	intervalReconnect time.Duration = time.Second * 1
	intervalPing      time.Duration = time.Second * 1
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
	if this.Done != nil {
		this.Done <- this
	}
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
	network     string
	addr        string
	stop        bool

	conMutex    sync.Mutex
	reqMutex    sync.Mutex

	//等待接收回复的请求
	reqsPending chan *RequestInfo

	ctrlChan    chan ctrlType
	connected   bool
	err         error
	pingTick    <-chan time.Time

	lastConnect time.Time
}

func (this *Client) String() string {
	return fmt.Sprintf("%s %s", this.network, this.addr)
}

func (this *Client) Connect() {
	this.connected = false

	var err error
	this.Conn, err = net.DialTimeout(this.network, this.addr, connectTimeout)
	if err != nil {
		this.err = err
		log.Printf("can't connect to redis %v:%v, error:%v", this.network, this.addr, err)
		return
	}

	this.readBuffer = bufio.NewReader(this.Conn)
	this.writeBuffer = bufio.NewWriter(this.Conn)

	this.connected = true
	this.err = nil
}

func (this *Client) Shutdown() {
	this.conMutex.Lock()
	defer this.conMutex.Unlock()

	this.sendShutdownCtrl()
	this.stop = true
	this.connected = false
}

func (this *Client) Ping() {
	this.Go(nil, "PING")
}

func (this *Client) IsShutDown() bool {
	return this.stop
}

func (this *Client) IsConnected() bool {
	return this.connected
}

func (this *Client) sendReconnectCtrl() {
	this.ctrlChan <- ctrlReconnect
}

func (this *Client) sendShutdownCtrl() {
	this.ctrlChan <- ctrlShutdown
}

func (this *Client) send(req *RequestInfo) {
	writeReqToBuf(this.writeBuffer, req)
}

func (this *Client) Go(done chan *RequestInfo, cmd string, args ...interface{}) *RequestInfo {
	req := newRequst(done, cmd, args...)
	this.sendRequest(req, false)
	return req
}

func (this *Client) sendRequest(req *RequestInfo, onlyWait bool) {

	this.conMutex.Lock()
	defer func() {
		if err := recover(); err != nil {
			req.err = err.(error)
			req.done()
			this.connected = false
			this.sendReconnectCtrl()
		}
		this.conMutex.Unlock()
	}()

	if this.IsShutDown() {
		panic(ErrNotRunning)
	}

	if !this.IsConnected() {
		panic(ErrNotConnected)
	}

	if !onlyWait {
		this.send(req)
	}
	this.reqsPending <- req
}

func (this *Client) PubSubWait(done chan *RequestInfo) {
	req := new(RequestInfo)

	if done != nil {
		if cap(done) == 0 {
			log.Panic("redis client: done channel is unbuffered")
		}
		req.Done = done
	}

	this.reqsPending <- req

	return req
}

func (this *Client) PubSubSend(done chan *RequestInfo, cmd string, args ...interface{}) *RequestInfo {
	req := newRequst(done, cmd, args...)
	this.sendRequest(req, true)
	return req
}

func (this *Client) recover(err error) {
	this.conMutex.Lock()
	defer this.conMutex.Unlock()

	//一定时间段内只尝试重连一次
	if this.lastConnect.Add(intervalReconnect).After(time.Now()) {
		return
	}

	this.lastConnect = time.Now()
	this.clear(err)
	this.Connect()
}

//清空等待的请求
func (this *Client) clear(err error) {
	close(this.reqsPending)
	for req := range this.reqsPending {
		req.err = err
		req.done()
	}
	this.reqsPending = make(chan *RequestInfo, 100)
}

func (this *Client) control(ctrl ctrlType) {
	switch ctrl {
	case ctrlReconnect:
		this.recover(ErrNotConnected)
	case ctrlShutdown:
		this.stop = true
		if this.Conn != nil {
			this.Conn.Close()
		}
		this.clear(ErrNotRunning)
	default:
		log.Panic(ErrUnexpectedCtrlType)
	}
}

//处理读请求和控制请求
func (this *Client) process() {
	for {
		if this.stop {
			break
		}

		select {
		case ctrl := <-this.ctrlChan:
			this.control(ctrl)
		case req := <-this.reqsPending:
			this.read(req)
		case <-this.pingTick:
			this.Ping()
		}
	}
}

func (this *Client) read(req *RequestInfo) {
	if this.IsShutDown() {
		return
	}

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
		network:     network,
		addr:        addr,
		stop:		false,
		connected:   false,
		reqsPending: make(chan *RequestInfo, 100),
		ctrlChan:    make(chan ctrlType, 10),
		pingTick:    time.Tick(intervalPing),
		lastConnect: time.Now(),
	}

	client.Connect()

	go client.process()

	return
}

func newRequst(done chan *RequestInfo, cmd string, args ...interface{}) (*RequestInfo) {
	req := new(RequestInfo)

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
