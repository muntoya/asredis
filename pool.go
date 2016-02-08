package asredis

import (
//	"time"
	"sync/atomic"
	//"fmt"
	"time"
)

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	clients     []*Connection
	addr        string
	replyChan   chan chan *Request
	nConn       int32
	nChan       int32
	msgID       int32
}

func (p *Pool) ConnsCreate() int32 {
	return p.nConn
}

func (p *Pool) ConnsFail() int32 {
	var i int32 = 0
	for _, c := range p.clients {
		if !c.isConnected() {
			i++
		}
	}
	return i
}

func (p *Pool) Exec(cmd string, args ...interface{}) (reply *Reply, err error) {
	msgID := atomic.AddInt32(&p.msgID, 1)
	connID := msgID % p.nConn
	conn := p.clients[connID]
	c := <-p.replyChan
	reply, err = conn.call(c, cmd, args...)
	p.replyChan <- c
	return
}

func (p *Pool) Close() {
	for _, c := range p.clients {
		c.close()
	}

	close(p.replyChan)
	for c := range p.replyChan {
		close(c)
	}
}

func (p *Pool) Eval(l *LuaEval, args ...interface{}) (reply *Reply, err error) {
	reply, err = p.Exec("EVALSHA", joinArgs(l.hash, args)...)

	if reply.Type == ERROR {
		var content string
		content, err = l.readFile()
		if err != nil {
			return
		}
		reply, err = p.Exec("EVAL", joinArgs(content, args)...)
	}
	return
}

func NewPool(addr string, nConn, nChan int32, plLen int, sendTimeout time.Duration) *Pool {
	clients := make([]*Connection, nConn)
	var i int32 = 0
	for ; i < nConn; i++ {
		clients[i] = NewConnection(addr, plLen, sendTimeout)
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

func joinArgs(s interface{}, args []interface{}) []interface{} {
	return append([]interface{}{s}, args...)
}