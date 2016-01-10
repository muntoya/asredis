package asredis
import "fmt"

//master/slave pool 用来存储主从服务器的全部连接,自动与所有集群中的服务器连接

type MSPool struct {
	master        Pool
	slaves        []Pool
	sentinelAddrs []string
	masterName    string
}

//遍历连接sentinel,一旦成功就获取集群全部信息并连接
func (this *MSPool) Connect() {
	var conn *SConnection
	for _, addr := range this.sentinelAddrs {
		conn = NewSConnection(addr)
		if !conn.IsConnected() {
			conn.Shutdown()
			conn = nil
		}
	}

	if conn == nil {
		return
	}

	pp, err := conn.GetMaster(this.masterName)
	conn.Shutdown()

	if err != nil {
		return
	}
	fmt.Println(pp)
}

func NewMSPool(master string, sentinelAddrs []string) *MSPool {
	mspool := &MSPool{
		sentinelAddrs: sentinelAddrs,
		masterName: master,
	}

	mspool.Connect()
	return mspool
}
