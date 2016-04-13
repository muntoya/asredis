package asredis

import (
//	"time"
//"fmt"
)

const (
	defaultPoolSize  int32 = 5
	defaultQueueSize int32 = 1024 * 8
)

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	Spec

	queueChan chan *RequestsPkg

	//连接个数
	PoolSize int32

	//发送请求到连接的队列长度
	QueueSize int32

	stopChan chan struct{}
}

func (p *Pool) Call(cmd string, args ...interface{}) (interface{}, error) {
	reqPkg := NewRequstPkg()
	reqPkg.Add(cmd, args...)
	p.Pipelining(reqPkg)
	req := reqPkg.requests[0]
	return req.Reply, req.Err
}

func (p *Pool) Pipelining(reqPkg *RequestsPkg) {
	p.queueChan <- reqPkg
	reqPkg.wait()
}

func (p *Pool) Start() {
	if p.stopChan != nil {
		panic("asredis: the given client is already started. Call Stop() before calling Start() ")
	}
	p.queueChan = make(chan *RequestsPkg, p.QueueSize)
}

func (p *Pool) Stop() {

}

func (p *Pool) Eval(l *LuaEval, args ...interface{}) (reply interface{}, err error) {
	reply, err = p.Call("EVALSHA", JoinArgs(l.hash, args)...)

	if err != nil {
		var content string
		content, err = l.readFile()
		if err != nil {
			return
		}
		reply, err = p.Call("EVAL", JoinArgs(content, args)...)
	}
	return
}

func NewPool() *Pool {
	pool := &Pool{
		Spec: *DefaultSpec(),
		PoolSize:       defaultPoolSize,
		QueueSize:      defaultQueueSize,
	}

	return pool
}
