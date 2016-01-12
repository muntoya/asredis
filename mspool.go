package asredis
import (
	"fmt"
	"strings"
)

//master/slave pool 用来存储主从服务器的全部连接,自动与所有集群中的服务器连接

type MSPool struct {
	master        *Pool
	slaves        []*Pool
	sentinelAddrs []string
	sentinel      *SConnection
	masterName    string
}

//遍历连接sentinel,一旦成功就获取集群全部信息并连接
func (this *MSPool) Connect() {
	this.checkSentinel()

	ppMaster, ppSlaveArray, err := this.getConnProps()


	if err != nil {
		return
	}
	fmt.Println(ppMaster, ppSlaveArray[0])
}

func (this *MSPool) checkSentinel() {
	if this.sentinel != nil && this.sentinel.IsConnected() {
		return
	}

	if this.sentinel != nil {
		this.sentinel.Close()
		this.sentinel = nil
	}

	for _, addr := range this.sentinelAddrs {
		this.sentinel = NewSConnection(addr)
		if !this.sentinel.IsConnected() {
			this.sentinel.Close()
		}
	}
}

func (this *MSPool) checkMaster() {
	ppMaster, err := this.sentinel.GetMaster(this.masterName)
	if err != nil {
		return
	}

	address := strings.Join([]string{ppMaster.ip, ppMaster.port}, ":")
	if this.master != nil && address == this.master.addr {
		return
	}

	if this.master != nil {
		this.master.Close()
		this.master = nil
	}

	this.master = NewPool(address, 10, 10)
}

func (this *MSPool) checkSlaves() {
	ppSlaveArray, err := this.sentinel.GetSlaves(this.masterName)
	if err != nil {
		return
	}

	var connMap [string]struct{}

	//TODO: 比较全部slave连接和配置
}

func (this *MSPool) Close() {
	this.master.Close()
	for _, s := range this.slaves {
		s.Close()
	}
}

func NewMSPool(master string, sentinelAddrs []string) *MSPool {
	mspool := &MSPool{
		sentinelAddrs: sentinelAddrs,
		masterName: master,
	}

	mspool.Connect()
	return mspool
}
