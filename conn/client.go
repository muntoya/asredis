package conn

import (
	"time"
	"log"
	"bytes"
	"sync"
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

type Client struct {
	*Connection

	mutex		sync.Mutex

	cmdBuf		bytes.Buffer
	reqsPending	chan *RequestInfo
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

	this.Send(req)

	return req
}

func (this *Client) Send(req *RequestInfo) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	b, err := writeReqToBuf(&this.cmdBuf, req)
	if err == nil {
		this.send(b)
		this.reqsPending <- req
	} else {
		req.err = err
		req.done()
	}
}


func (this *Client) input() {
	req := <- this.reqsPending
	readReply(this.readBuffer, &req.reply)
	req.done()
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	client = &Client{Connection: connection}
	client.reqsPending = make(chan *RequestInfo, 100)

	go client.input()

	return client, err
}
