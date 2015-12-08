package conn

import "github.com/muntoya/asredis/common"

type status_code byte

type requestInfo struct {
	id      int64
	stat    status_code
	cmd     string
	outbuff *[]byte
	future  interface{}
	error   common.Error
}
type reqPtr *requestInfo

type Client struct {
	conn		*Connection

	reqsToWrite	chan reqPtr
	reqsToRead	chan reqPtr
}

func (this *Client) QueueRequest(req *reqPtr) {
	this.reqsToWrite <- req
}