package asredis

import (
//	"time"
	//"fmt"
)

const (
	defaultPoolSize int32 = 5
	defaultChanSize int32 = 80
	defaultQueueSize int32 = 40
)

type PoolSpec struct {
	*ConnectionSpec

	//连接个数
	PoolSize	int32

	//全部预存的channel数量
	ChanSize	int32

	//发送请求到连接的队列长度
	QueueSize	int32
}

func DefaultPoolSpec() *PoolSpec {
	return &PoolSpec{
		ConnectionSpec: DefaultConnectionSpec(),
		PoolSize:	defaultPoolSize,
		ChanSize:	defaultChanSize,
		QueueSize:	defaultQueueSize,
	}
}

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	conns     []*Connection
	PoolSpec
	replyChan chan struct{}
	queueChan chan *requestsPkg
}

func (p *Pool) ConnsCreate() int32 {
	return p.PoolSize
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

func (p *Pool) Call(req... *Request) {
	c := <-p.replyChan
	reqPkg := requestsPkg{req, c}
	p.queueChan <- reqPkg
	reqPkg.wait()
	p.replyChan <- c
}

func (p *Pool) Go(reqPkg *requestsPkg) {
	p.queueChan <- reqPkg
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
	reply, err = p.Call("EVALSHA", joinArgs(l.hash, args)...)

	if reply.Type == ERROR {
		var content string
		content, err = l.readFile()
		if err != nil {
			return
		}
		reply, err = p.Call("EVAL", joinArgs(content, args)...)
	}
	return
}

func NewPool(spec *PoolSpec) *Pool {
	conns := make([]*Connection, spec.PoolSize)
	queueChan := make(chan *requestsPkg, spec.ChanSize)

	for i := int32(0); i < spec.PoolSize; i++ {
		conns[i] = NewConnection(spec.ConnectionSpec, queueChan)
	}

	pool := &Pool{
		conns:    conns,
		PoolSpec:   *spec,
		replyChan:  make(chan struct{}, spec.ChanSize),
		queueChan:	queueChan,
	}

	for i := int32(0); i < spec.ChanSize; i++ {
		pool.replyChan <- make(chan *Request, 1)
	}

	return pool
}

func joinArgs(s interface{}, args []interface{}) []interface{} {
	return append([]interface{}{s}, args...)
}