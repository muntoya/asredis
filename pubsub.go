package asredis

import (
//	"fmt"
	"time"
	"log"
)

const (
	subTimeout time.Duration = time.Second * 1
)

type SubMsg struct {
	Channel string
	Value   string
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
		messageChan: make(chan *SubMsg, 100),
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
		if err != nil {
			log.Printf("read sub reply error: %v", err)
		} else {

			if len(reply.Array) == 3 && reply.Array[0].(string) == "message" {
				msg := SubMsg{Channel: reply.Array[1].(string), Value: reply.Array[2].(string)}
				this.messageChan <- &msg
			}
		}
	}
}

func (this *PubsubClient) GetMessage(timeout time.Duration) *SubMsg {
	var tick <-chan time.Time
	if timeout == 0 {
		tick = this.subTick
	} else {
		tick = time.Tick(timeout)
	}

	select {
	case msg := <-this.messageChan:
		return msg
	case <-tick:
		return nil
	}
}
