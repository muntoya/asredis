package conn

import (
	"time"
	"log"
	"fmt"
	"bytes"
	"strconv"
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

func (this *RequestInfo) GetReply() *Reply {
	<- this.Done
	return &this.reply
}

type Client struct {
	conn		*Connection

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

	str, err := writeReqToBuf(&this.cmdBuf, req)
	if err == nil {
		this.conn.send(str)
		this.reqsPending <- req
	} else {
		req.err = err
		req.done()
	}
}


func (this *Client) input() {
	req := <- this.reqsPending
	readReply(this.conn.readBuffer, &req.reply)
	req.done()
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	client = &Client{conn: connection}
	client.reqsPending = make(chan *RequestInfo, 100)

	go client.input()

	return client, err
}

func writeReqToBuf(buf *bytes.Buffer, req *RequestInfo) (str string, err error) {
	buf.Reset()

	//写入参数个数
	argsCnt := len(req.args) + 1
	buf.WriteByte(array_byte)
	buf.WriteString(strconv.Itoa(argsCnt))
	buf.Write(cr_lf)

	//写入命令
	buf.WriteByte(size_byte)
	buf.WriteString(strconv.Itoa(len(req.cmd)))
	buf.Write(cr_lf)
	buf.WriteString(req.cmd)
	buf.Write(cr_lf)

	//写入参数
	for _, arg := range req.args {
		v := fmt.Sprint(arg)
		buf.WriteByte(size_byte)
		buf.WriteString(strconv.Itoa(len(v)))
		buf.Write(cr_lf)

		buf.WriteString(v)
		buf.Write(cr_lf)
	}

	return buf.String(), nil
}
