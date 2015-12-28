package asredis

//订阅
type PubsubClient struct {
	*Client
	replyChan chan *RequestInfo
}

func NewPubsubClient(network, addr string) (pubsubClient *PubsubClient) {
	pubsubClient = &PubsubClient{
		Client: NewClient(network, addr),
		replyChan:  make(chan *RequestInfo, 1),
	}

	go pubsubClient.process()

	return pubsubClient
}

func (this *PubsubClient) Sub(channel ...string) (err error) {
	req := this.Go(this.replyChan, "SUBSCRIBE", channel...)
	_, err = req.GetReply()
	return
}

func (this *PubsubClient) UnSub(channel ...string) (err error) {
	req := this.Go(this.replyChan, "UBSUBSCRIBE", channel...)
	_, err = req.GetReply()
	return
}

func (this *PubsubClient) process() {
	for {

	}
}
