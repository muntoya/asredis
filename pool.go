package asredis

// 用来保存连接至单个redis进程的多个连接
type Pool struct {
	clients		[]*Client
}
