package asredis

import (
	"errors"
)

var (
	ErrClusterNoService = errors.New("can't connect to any server of cluster")
)

const numSlots = 16384

type mapping [numSlots]string

type CPool struct {
	*Pool
	slot    []int
}

type Cluster struct {
	pools	map [string]Pool
	addrs   []string
}

func (c *Cluster) connect() (error) {
	var conn *Connection
	for _, addr := range c.addrs {
		conn = NewConnection(addr)
		if conn.IsConnected() {
			break
		}
	}

	if conn == nil || !conn.IsConnected() {
		return ErrClusterNoService
	}

	return nil
}

func (c *Cluster) getNodes() {

}

func (c *Cluster) checkCluster() {

}

func NewCluster(addrs []string) (cluster *Cluster, err error) {
	cluster = &Cluster {
		addrs: addrs,
	}

	err = cluster.connect()

	if err != nil {
		return nil, err
	}
	
	return
}
