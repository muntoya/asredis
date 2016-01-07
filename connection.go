package asredis

import (
	"bufio"
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

type Request struct {
	cmd   string
	args  []interface{}
	err   error
	reply *Reply
	Done  chan *Request
}

func (this *Request) done() {
	if this.Done != nil {
		this.Done <- this
	}
}

func (this *Request) GetReply() (*Reply, error) {
	<-this.Done
	return this.reply, this.err
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

	conMutex    sync.Mutex
	reqMutex    sync.Mutex

	//等待接收回复的请求
	reqsPending chan *Request

	ctrlChan    chan ctrlType
	connected   bool
	err         error
	pingTick    <-chan time.Time

	lastConnect time.Time
}

func (this *Connection) String() string {
	return this.addr
}

func (this *Connection) Connect() {
	this.connected = false

	var err error
	this.Conn, err = net.DialTimeout("tcp", this.addr, connectTimeout)
	if err != nil {
		this.err = err
		log.Printf("can't connect to redis %v, error:%v", this.addr, err)
		return
	}

	this.readBuffer = bufio.NewReader(this.Conn)
	this.writeBuffer = bufio.NewWriter(this.Conn)

	this.connected = true
	this.err = nil
}

func (this *Connection) Shutdown() {
	this.conMutex.Lock()
	defer this.conMutex.Unlock()

	this.sendShutdownCtrl()
	this.stop = true
	this.connected = false
}

func (this *Connection) Ping() {
	this.Go(nil, "PING")
}

func (this *Connection) IsShutDown() bool {
	return this.stop
}

func (this *Connection) IsConnected() bool {
	return this.connected
}

func (this *Connection) sendReconnectCtrl() {
	this.ctrlChan <- ctrlReconnect
}

func (this *Connection) sendShutdownCtrl() {
	this.ctrlChan <- ctrlShutdown
}

func (this *Connection) send(req *Request) {
	writeReqToBuf(this.writeBuffer, req)
}

func (this *Connection) Go(done chan *Request, cmd string, args ...interface{}) *Request {
	req := newRequst(done, cmd, args...)
	this.sendRequest(req, false)
	return req
}

func (this *Connection) sendRequest(req *Request, onlySend bool) {

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

	this.send(req)
	if !onlySend {
		this.reqsPending <- req
	}
}

func (this *Connection) PubsubWait(done chan *Request) (*Reply, error) {
	req := newRequst(done, "")
	this.reqsPending <- req
	return req.GetReply()
}

func (this *Connection) PubsubSend(cmd string, args ...interface{}) error {
	req := newRequst(nil, cmd, args...)
	this.sendRequest(req, true)
	return req.err
}

func (this *Connection) recover(err error) {
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
func (this *Connection) clear(err error) {
	close(this.reqsPending)
	for req := range this.reqsPending {
		req.err = err
		req.done()
	}
	this.reqsPending = make(chan *Request, 100)
}

func (this *Connection) control(ctrl ctrlType) {
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
func (this *Connection) process() {
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

func (this *Connection) read(req *Request) {
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

func NewConnection(addr string) (client *Connection) {
	client = &Connection{
		addr:        addr,
		stop:		false,
		connected:   false,
		reqsPending: make(chan *Request, 100),
		ctrlChan:    make(chan ctrlType, 10),
		pingTick:    time.Tick(intervalPing),
		lastConnect: time.Now(),
	}

	client.Connect()

	go client.process()

	return
}

func newRequst(done chan *Request, cmd string, args ...interface{}) (*Request) {
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
