package asredis

import (
//	"time"
	//"fmt"
)

const (
	defaultPoolSize int32 = 5
	defaultQueueSize int32 = 40
)

type PoolSpec struct {
	*ConnectionSpec

	//连接个数
	PoolSize	int32

	//发送请求到连接的队列长度
	QueueSize	int32
}

func DefaultPoolSpec() *PoolSpec {
	return &PoolSpec{
		ConnectionSpec: DefaultConnectionSpec(),
		PoolSize:	defaultPoolSize,
		QueueSize:	defaultQueueSize,
	}
}

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	conns     []*Connection
	PoolSpec
	queueChan chan *RequestsPkg
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

func (p *Pool) Call(cmd string, args...interface{}) (*Reply, error) {
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

func (p *Pool) Close() {
	for _, c := range p.conns {
		c.Close()
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
	queueChan := make(chan *RequestsPkg, spec.QueueSize)

	for i := int32(0); i < spec.PoolSize; i++ {
		conns[i] = NewConnection(*spec.ConnectionSpec, queueChan)
	}

	pool := &Pool{
		conns:    conns,
		PoolSpec:   *spec,
		queueChan:	queueChan,
	}

	return pool
}

func joinArgs(s interface{}, args []interface{}) []interface{} {
	return append([]interface{}{s}, args...)
}