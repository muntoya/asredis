package asredis

//master/slave pool 用来存储主从服务器的全部连接,自动与所有集群中的服务器连接

type MSPool struct {
	master  Pool
	slaves  []Pool
	sentinels   []string
}
