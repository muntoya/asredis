package asredis

import (
	"fmt"
	"time"
)

const (
	subTimeout time.Duration = time.Second * 1
)

//订阅
type PubsubClient struct {
	redisClient *Client
	replyChan   chan *RequestInfo
	subTick     <-chan time.Time
}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		redisClient: NewClient(network, addr),
		replyChan:   make(chan *RequestInfo, 1),
		subTick:     time.Tick(subTimeout),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (this *PubsubClient) Sub(channel ...interface{}) (err error) {
	req := this.redisClient.PubsubSend("SUBSCRIBE", channel...)
	return req.err
}

func (this *PubsubClient) UnSub(channel ...interface{}) (err error) {
	req := this.redisClient.PubsubSend("UBSUBSCRIBE", channel...)
	return req.err
}

func (this *PubsubClient) process() {
	for {
		req := this.redisClient.PubsubWait(this.replyChan)
		reply, err := req.GetReply()
		fmt.Println(*reply, err)
	}
}
