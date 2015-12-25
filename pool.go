package asredis

import (
//	"time"
)
// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	clients		[]*Client
}

func NewPool(network, addr string, n int) *Pool {
	clients := make([]*Client, n)
	for i := 0; i < n; i++ {
		clients[i] = NewClient(network, addr)
	}

	return &Pool{clients: clients}
}
