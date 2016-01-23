package asredis

import (
	"errors"
	"fmt"
)

var (
	ErrClusterNoService = errors.New("can't connect to any server of cluster")
)

const numSlots = 16384

type mapping [numSlots]*CPool

type CPool struct {
	*Pool

}

type Cluster struct {
	mapping
	pools	[]*CPool
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

	getNodes(conn)
	getSlots(conn)
	return nil
}

func getSlots(conn *Connection) {
	c := make(chan *Request, 1)
	r, err := conn.call(c, "CLUSTER", "slots")
	fmt.Println(r, err)
}

func getNodes(conn *Connection) {
	c := make(chan *Request, 1)
	r, err := conn.call(c, "CLUSTER", "nodes")
	fmt.Println(r, err)
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
