package asredis

import (
//	"time"
	"sync/atomic"
)

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	clients     []*Client
	addr        string
	replyChan   chan chan *Request
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
	this.replyChan <- c
	return
}

func (this *Pool) Close() {
	for _, c := range this.clients {
		c.Close()
	}

	close(this.replyChan)
	for c := range this.replyChan {
		close(c)
	}
}

func NewPool(addr string, nConn, nChan int32) *Pool {
	clients := make([]*Client, nConn)
	var i int32 = 0
	for ; i < nConn; i++ {
		clients[i] = NewClient(addr)
	}

	pool := &Pool{
		clients:    clients,
		addr:       addr,
		replyChan:  make(chan chan *Request, nChan),
		nConn:      nConn,
		nChan:      nChan,
	}

	for i = 0; i < nChan; i++ {
		pool.replyChan <- make(chan *Request, 1)
	}

	return pool
}
