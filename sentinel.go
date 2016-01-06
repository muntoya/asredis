package asredis

//sentinel connetcion,工具类,用来实现集群的连接

type SConnection struct {
	*Connection
}

func NewSConnection(addr string) *SConnection {
	return &SConnection{
		Connection: NewConnection(addr),
	}
}
