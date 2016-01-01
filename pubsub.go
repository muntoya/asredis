package asredis

import (
	//	"fmt"
	"log"
	"time"
)

const (
	getReplyTimeout     time.Duration = time.Second * 1
	subTimeout	time.Duration = time.Millisecond * 100
	messageChanLen int           = 100
)

type SubMsg struct {
	Channel string
	Value   string
}

//订阅
type PubsubClient struct {
	redisClient *Client
	replyChan   chan *Request
	replyTick   <-chan time.Time
	subTick     time.Ticker
	messageChan chan *SubMsg
	subChan     chan error
	unsubChan   chan error
}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		redisClient: NewClient(network, addr),
		replyChan:   make(chan *Request, 1),
		messageChan: make(chan *SubMsg, messageChanLen),
		subChan:     make(chan error),
		unsubChan:   make(chan error),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (this *PubsubClient) Sub(channel ...interface{}) (err error) {
	err = this.redisClient.PubsubSend("SUBSCRIBE", channel...)
	if err != nil {
		return
	}

	subTick := time.Tick(subTimeout)
	select {
	case err = <-this.subChan:
	case <-subTick:
		err = ErrWaitReplyTimeout
	}

	return
}

func (this *PubsubClient) UnSub(channel ...interface{}) (err error) {
	err = this.redisClient.PubsubSend("UBSUBSCRIBE", channel...)

	subTick := time.Tick(subTimeout)
	select {
	case err = <-this.unsubChan:
	case <-subTick:
		err = ErrWaitReplyTimeout
	}

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
				select {
				case this.subChan <- err:
				default:
				}
			case "unsubscribe":
				select {
				case this.subChan <- err:
				default:
				}
			default:
				log.Printf("error pubsub reply type: %v", t)
			}
		}
	}
}

func (this *PubsubClient) GetMessage(timeout time.Duration) *SubMsg {
	var tick <-chan time.Time
	if timeout == 0 {
		tick = this.replyTick
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
