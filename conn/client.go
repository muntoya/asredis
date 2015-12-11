package conn

import (
	"time"
	"github.com/muntoya/asredis/common"
)

type status_code byte

type requestInfo struct {
	stat    status_code
	cmd     string
	outbuff *[]byte
	error   common.RedisError
	done	chan *requestInfo
}

type Client struct {
	conn		*Connection

	reqsSend	chan *requestInfo
	reqsRecv	chan *requestInfo
}


func (this *Client) QueueRequest(cmd string, args []string) {
	req := &requestInfo{cmd: cmd, }
	this.reqsSend <- req
}


func (this *Client) input() {

}

func NewClient(network, addr string, timeout time.Duration) (client *Client, err error) {
	connection, err := DialTimeout(network, addr, timeout)
	if err == nil {
		return
	}

	client = &Client{conn: connection}
	client.reqsRecv = make(chan *requestInfo, 100)
	client.reqsRecv = make(chan *requestInfo, 100)

	go client.input()

	return client, err
}

