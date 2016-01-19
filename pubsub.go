package asredis

import (
	//	"fmt"
	"log"
	"time"
	"errors"
)

var (
	ErrWaitReplyTimeout = errors.New("redis: wait reply time out")
)

const (
	getReplyTimeout time.Duration = time.Second * 1
	commandTimeout  time.Duration = time.Millisecond * 100
	messageChanLen  int           = 100
)

type SubMsg struct {
	Channel string
	Value   string
}

//订阅
type PubsubClient struct {
	redisClient *Connection
	replyChan   chan *Request
	messageChan chan *SubMsg
	subChan     chan error
	unsubChan   chan error
}

func NewPubsubClient(addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		redisClient: NewConnection(addr),
		replyChan:   make(chan *Request, 1),
		messageChan: make(chan *SubMsg, messageChanLen),
		subChan:     make(chan error),
		unsubChan:   make(chan error),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (p *PubsubClient) Sub(channel ...interface{}) (err error) {
	err = p.redisClient.PubsubSend("SUBSCRIBE", channel...)
	if err != nil {
		return
	}

	subTick := time.After(commandTimeout)
	select {
	case err = <-p.subChan:
	case <-subTick:
		err = ErrWaitReplyTimeout
	}

	return
}

func (p *PubsubClient) UnSub(channel ...interface{}) (err error) {
	err = p.redisClient.PubsubSend("UBSUBSCRIBE", channel...)

	subTick := time.After(commandTimeout)
	select {
	case err = <-p.unsubChan:
	case <-subTick:
		err = ErrWaitReplyTimeout
	}

	return
}

func (p *PubsubClient) process() {
	for {
		reply, err := p.redisClient.PubsubWait(p.replyChan)
		if err != nil {
			log.Panic("read sub reply error: %v\n", err)
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
				p.messageChan <- &msg
			case "subscribe":
				select {
				case p.subChan <- err:
				default:
				}
			case "unsubscribe":
				select {
				case p.subChan <- err:
				default:
				}
			default:
				log.Printf("error pubsub reply type: %v\n", t)
			}
		}
	}
}

func (p *PubsubClient) GetMessage(timeout time.Duration) *SubMsg {
	var tick <-chan time.Time
	if timeout == 0 {
		tick = time.After(getReplyTimeout)
	} else {
		tick = time.After(timeout)
	}

	select {
	case msg := <-p.messageChan:
		return msg
	case <-tick:
		return nil
	}
}
