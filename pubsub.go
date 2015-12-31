package asredis

import (
	"fmt"
	"time"
)

const (
	subTimeout time.Duration = time.Second * 1
)

type SubMsg struct {
	Channel string
	Value   interface{}
}

//订阅
type PubsubClient struct {
	redisClient *Client
	replyChan   chan *Request
	subTick     <-chan time.Time
	messageChan chan *SubMsg
}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		redisClient: NewClient(network, addr),
		replyChan:   make(chan *Request, 1),
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
		reply, err := this.redisClient.PubsubWait(this.replyChan)
		fmt.Println(*reply, err)
	}
}
