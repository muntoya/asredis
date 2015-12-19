package conn

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

type Client struct {
	net.Conn
	readBuffer  *bufio.Reader
	timeout     time.Duration
	writeBuffer *bufio.Writer
	Network     string
	Addr        string

	mutex		sync.Mutex

	cmdBuf		bytes.Buffer
	reqsPending	chan *RequestInfo
}

func (this *Client) String() string {
	return fmt.Sprintf("%s %s", this.Network, this.Addr)
}

func (this *Client) Connect() (err error) {
	this.Conn, err =  net.DialTimeout(this.Network, this.Addr, this.timeout)
	return
}

func (this *Client) Close() {
	if this.Conn != nil {
		this.Conn.Close()
	}
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

	this.Send(req)

	return req
}

func (this *Client) Send(req *RequestInfo) {
	this.mutex.Lock()
	defer func() {
		if err := recover(); err != nil {
			req.err = err.(error)
		}
		this.mutex.Unlock()
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
	close(this.reqsPending)
	for req := range this.reqsPending {
		req.err = err
	}

}

func (this *Client) input() {
	for {
		req := <-this.reqsPending
		err := readReply(this.readBuffer, &req.reply)
		req.err = err
		req.done()
	}
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	client = &Client{
		Conn:        connection,
		readBuffer:  bufio.NewReader(connection),
		timeout:     timeout,
		writeBuffer: bufio.NewWriter(connection),
		Network:     network,
		Addr:        addr,
		reqsPending: make(chan *RequestInfo, 100),
	}

	go client.input()

	return client, err
}
