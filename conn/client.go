package conn

import (
	"time"
	"log"
	"fmt"
	"bytes"
	"strconv"
)


type Reply struct {
	Type	ResponseType
	Value	interface{}
}

type requestInfo struct {
	cmd     string
	args 	[]interface{}
	err		error
	reply	Reply
	Done	chan *requestInfo
}

func (this *requestInfo) done() {
	this.Done <- this
}

type Client struct {
	conn		*Connection

	cmdBuf		bytes.Buffer
	reqsPending	chan *requestInfo
}

func (this *Client) Go(done chan *requestInfo, cmd string, args ...interface{}) *requestInfo {
	req := new(requestInfo)

	if done == nil {
		done = make(chan *requestInfo, 1)
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

func (this *Client) Send(req *requestInfo) {
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
	ret := string(this.conn.readToCRLF())
	req := <- this.reqsPending
	req.reply.Value = ret
	req.done()
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}

	client = &Client{conn: connection}
	client.reqsPending = make(chan *requestInfo, 100)

	go client.input()

	return client, err
}

func writeReqToBuf(buf *bytes.Buffer, req *requestInfo) (str string, err error) {
	buf.Reset()

	//写入参数个数
	argsCnt := len(req.args) + 1
	buf.WriteByte(count_byte)
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
