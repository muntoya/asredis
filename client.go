package asredis

import (
	"time"
	"log"
	"bytes"
	"sync"
	"net"
	"bufio"
	"fmt"
)


type Reply struct {
	Type	ResponseType
	Value	interface{}
	Array	[]interface{}
}

type RequestInfo struct {
	cmd     string
	args 	[]interface{}
	err		error
	reply	Reply
	Done	chan *RequestInfo
}

func (this *RequestInfo) done() {
	this.Done <- this
}

func (this *RequestInfo) GetReply() (*Reply, error) {
	<- this.Done
	return &this.reply, this.err
}


type commandType byte

const (
	cmdReconnect commandType = iota
	cmdShutdown
)

type Client struct {
	net.Conn
	readBuffer  *bufio.Reader
	timeout     time.Duration
	writeBuffer *bufio.Writer
	Network     string
	Addr        string

	conMutex    sync.Mutex
	reqMutex	sync.Mutex

	cmdBuf      bytes.Buffer
	reqsPending chan *RequestInfo

	cmdChan		chan commandType
	connected	bool
}

func (this *Client) String() string {
	return fmt.Sprintf("%s %s", this.Network, this.Addr)
}

func (this *Client) Connect() (err error) {
	this.Conn, err =  net.DialTimeout(this.Network, this.Addr, this.timeout)
	if err != nil {
		return
	}

	this.readBuffer = bufio.NewReader(this.Conn)
	this.writeBuffer = bufio.NewWriter(this.Conn)

	this.connected = true

	return
}

func (this *Client) Close() {
	if this.Conn != nil {
		this.Conn.Close()
	}
	this.connected = false
}

func (this *Client) IsConnected() bool {
	return this.connected
}

func (this *Client) send(b []byte) {
	this.writeBuffer.Write(b)
	this.writeBuffer.Flush()
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
		if err := recover(); err != nil {
			req.err = err.(error)
			this.cmdChan <- cmdReconnect
			this.connected = false
		}
		this.conMutex.Unlock()
	}()


	b, err := writeReqToBuf(&this.cmdBuf, req)
	if err == nil {
		this.send(b)
		this.reqsPending <- req
	} else {
		req.err = err
		req.done()
	}
}

func (this *Client) recover(err error) {
	this.conMutex.Lock()
	close(this.reqsPending)
	for req := range this.reqsPending {
		req.err = err
		req.done()
	}

	this.reqsPending = make(chan *RequestInfo, 100)
	this.Connect()

	this.conMutex.Unlock()
}

func (this *Client) input() {
	for {
		select {
		case req := <-this.reqsPending:
			err := readReply(this.readBuffer, &req.reply)
			req.err = err
			req.done()

			if err != nil {
				this.recover(err)
			}
		case cmd := <-this.cmdChan:
			switch cmd {
			case cmdReconnect:
				this.recover(ErrNotConnected)
			default:

			}
		}
	}
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	client = &Client{
		timeout:     timeout,
		Network:     network,
		Addr:        addr,
		connected:   false,
		reqsPending: make(chan *RequestInfo, 100),
		cmdChan:	 make(chan commandType, 10),
	}

	err = client.Connect()

	go client.input()

	return client, err
}
