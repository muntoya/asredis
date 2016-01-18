package asredis

import (
	//"fmt"
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
func (p *MSPool) Connect() {
	p.checkSentinel()
	p.checkMaster()
	p.checkSlaves()
}

func (p *MSPool) checkSentinel() {
	if p.sentinel != nil && p.sentinel.IsConnected() {
		return
	}

	if p.sentinel != nil {
		p.sentinel.Close()
		p.sentinel = nil
	}

	for _, addr := range p.sentinelAddrs {
		p.sentinel = NewSConnection(addr)
		if !p.sentinel.IsConnected() {
			p.sentinel.Close()
		}
	}
}

func (p *MSPool) checkMaster() {
	ppMaster, err := p.sentinel.GetMaster(p.masterName)
	if err != nil {
		return
	}

	address := strings.Join([]string{ppMaster.ip, ppMaster.port}, ":")
	if p.master != nil && address == p.master.addr {
		return
	}

	if p.master != nil {
		p.master.Close()
		p.master = nil
	}

	p.master = NewPool(address, 10, 10)
}

func (p *MSPool) checkSlaves() {
	_, err := p.sentinel.GetSlaves(p.masterName)
	if err != nil {
		return
	}

	//TODO: 比较全部slave连接和配置
}

func (p *MSPool) Close() {
	p.master.Close()
	for _, s := range p.slaves {
		s.Close()
	}
}

func NewMSPool(master string, sentinelAddrs []string) *MSPool {
	mspool := &MSPool{
		sentinelAddrs: sentinelAddrs,
		masterName:    master,
	}

	mspool.Connect()
	return mspool
}
