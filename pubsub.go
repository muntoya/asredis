package asredis

import (
	//	"fmt"
	"log"
	"time"
)

const (
	subTimeout     time.Duration = time.Second * 1
	messageChanLen int           = 100
	subChanLen     int           = 10
	unSubChanLen   int           = 10
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
	subChan     chan error
	unSubChan   chan error
}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		redisClient: NewClient(network, addr),
		replyChan:   make(chan *Request, 1),
		subTick:     time.Tick(subTimeout),
		messageChan: make(chan *SubMsg, messageChanLen),
		subChan:     make(chan error, subChanLen),
		unSubChan:   make(chan error, unSubChanLen),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (this *PubsubClient) Sub(channel ...interface{}) (err error) {
	err = this.redisClient.PubsubSend("SUBSCRIBE", channel...)
	return
}

func (this *PubsubClient) UnSub(channel ...interface{}) (err error) {
	err = this.redisClient.PubsubSend("UBSUBSCRIBE", channel...)
	return
}

func (this *PubsubClient) process() {
	for {
		reply, err := this.redisClient.PubsubWait(this.replyChan)
		if err != nil {
			log.Printf("read sub reply error: %v\n", err)
		} else {

			if len(reply.Array) != 3 {
				continue
			}

			t, ok := reply.Array[0].(string)
			if !ok {
				continue
			}

			switch t {
			case "message":
				msg := SubMsg{Channel: reply.Array[1].(string), Value: reply.Array[2].(string)}
				this.messageChan <- &msg
			case "subscribe":
			//	this.subChan <- err
			case "unsubscribe":
			//	this.unSubChan <- err
			default:
				log.Printf("error pubsub reply type: %v", t)
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
