package asredis

import (
	"time"
	"fmt"
)

const (
	subTimeout time.Duration = time.Second * 1
)

//订阅
type PubsubClient struct {
	*Client
	replyChan chan *RequestInfo
	subTick   <-chan time.Time

}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		Client:    NewClient(network, addr),
		replyChan: make(chan *RequestInfo, 1),
		subTick:   time.Tick(subTimeout),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (this *PubsubClient) Sub(channel ...interface{}) (err error) {
	req := this.PubsubSend("SUBSCRIBE", channel...)
	_, err = req.GetReply()
	return
}

func (this *PubsubClient) UnSub(channel ...interface{}) (err error) {
	req := this.PubsubSend("UBSUBSCRIBE", channel...)
	_, err = req.GetReply()
	return
}

func (this *PubsubClient) process() {
	for {
		req := this.PubsubWait(this.replyChan)
		reply, err := req.GetReply()
		fmt.Println(*reply, err)
	}
}
