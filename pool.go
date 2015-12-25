package asredis

import (
	"time"
	"sync/atomic"
)

const (
	intervalPing time.Duration = time.Second * 1
)

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	clients     []*Client
	pingTick    time.Time
	replyChan   chan chan *RequestInfo
	nConn       int32
	nChan       int32
	msgID       int32
}

func (this *Pool) Exec(cmd string, args ...interface{}) (reply *Reply, err error) {
	msgID := atomic.AddInt32(&this.msgID, 1)
	connID := msgID % this.nConn
	conn := this.clients[connID]
	c := <-this.replyChan
	req := conn.Go(c, cmd, args...)
	reply, err = req.GetReply()
	return
}

func NewPool(network, addr string, nConn, nChan int32) *Pool {
	clients := make([]*Client, nConn)
	var i int32 = 0
	for ; i < nConn; i++ {
		clients[i] = NewClient(network, addr)
	}

	pool := &Pool{
		clients:    clients,
		pingTick:   time.Tick(intervalPing),
		replyChan:  make(chan chan *RequestInfo, nChan),
		nConn:      nConn,
		nChan:      nChan,
	}

	for i := 0; i < nChan; i++ {
		pool.replyChan <- make(chan *RequestInfo, 1)
	}

	return pool
}
