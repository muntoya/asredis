package asredis

import (
//	"time"
	"sync/atomic"
	//"fmt"
)

const (
	defaultPoolSize int32 = 3
	defaultChanSize int32 = 500
)

type PoolSpec struct {
	*ConnectionSpec

	//连接个数
	PoolSize	int32

	ChanSize	int32
}

func DefaultPoolSpec() *PoolSpec {
	return &PoolSpec{
		ConnectionSpec: DefaultConnectionSpec(),
		PoolSize:	defaultPoolSize,
		ChanSize:	defaultChanSize,
	}
}

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	conns     []*Connection
	poolSpec  *PoolSpec
	replyChan chan chan *Request
	msgID     int32
}

func (p *Pool) ConnsCreate() int32 {
	return p.poolSpec.PoolSize
}

func (p *Pool) ConnsFail() int32 {
	var i int32 = 0
	for _, c := range p.conns {
		if !c.isConnected() {
			i++
		}
	}
	return i
}

func (p *Pool) Exec(cmd string, args ...interface{}) (reply *Reply, err error) {
	msgID := atomic.AddInt32(&p.msgID, 1)
	connID := msgID % p.poolSpec.PoolSize
	conn := p.conns[connID]
	c := <-p.replyChan
	reply, err = conn.call(c, cmd, args...)
	p.replyChan <- c
	return
}

func (p *Pool) Close() {
	for _, c := range p.conns {
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

func NewPool(spec *PoolSpec) *Pool {
	conns := make([]*Connection, spec.PoolSize)
	var i int32 = 0
	for ; i < spec.PoolSize; i++ {
		conns[i] = NewConnection(spec.ConnectionSpec)
	}

	pool := &Pool{
		conns:    conns,
		poolSpec:   spec,
		replyChan:  make(chan chan *Request, spec.ChanSize),
	}

	for i = 0; i < spec.ChanSize; i++ {
		pool.replyChan <- make(chan *Request, 1)
	}

	return pool
}

func joinArgs(s interface{}, args []interface{}) []interface{} {
	return append([]interface{}{s}, args...)
}