package conn

import (
	"time"
	"log"
	"fmt"
	"bytes"
	"github.com/muntoya/asredis/common"
)

type status_code byte

type requestInfo struct {
	stat    status_code
	cmd     string
	args 	[]string
	error   common.RedisError
	Done	chan *requestInfo
}

type Client struct {
	conn		*Connection

	cmdBuf		bytes.Buffer
	reqsPending	chan *requestInfo
}

func (this *Client) Go(cmd string, args []string, done chan *requestInfo) *requestInfo {
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
	this.cmdBuf.WriteString(req.cmd)

	for _, arg := range req.args {
		this.cmdBuf.WriteByte(' ')
		this.cmdBuf.WriteString(arg)
	}

	this.cmdBuf.Write([]byte{cr_byte, lf_byte})
	str := this.cmdBuf.String()
	fmt.Println(str)
	this.conn.send(str)
	this.cmdBuf.Reset()
	this.reqsPending <- req
}


func (this *Client) input() {
	fmt.Println(string(this.conn.readToCRLF()))
	req := <- this.reqsPending
	req.Done <- req
}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := DialTimeout(network, addr, timeout)
	if err != nil {
		return
	}

	client = &Client{conn: connection}
	client.reqsPending = make(chan *requestInfo, 100)

	go client.input()

	return client, err
}

