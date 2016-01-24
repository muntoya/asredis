package asredis

import (
	"errors"
	"fmt"
	"strings"
	"strconv"
)

var (
	ErrClusterNoService = errors.New("can't connect to any server of cluster")
	ErrSlotsInfo = errors.New("can't get infomation of slots")
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

type Slots struct {
	begin	uint16
	end		uint16
	addrs	[]string
}
func (c *Cluster) connect() (error) {
	var conn *Connection
	for _, addr := range c.addrs {
		conn = NewConnection(addr)
		if conn.isConnected() {
			break
		}
	}

	if conn == nil || !conn.isConnected() {
		return ErrClusterNoService
	}

	getNodes(conn)
	getSlots(conn)
	return nil
}

func getSlots(conn *Connection) (slotsArray []*Slots, err error) {
	c := make(chan *Request, 1)

	var r *Reply
	r, err = conn.call(c, "CLUSTER", "slots")
	if err != nil {
		return
	}

	if (r.Type != ARRAY) {
		err = ErrSlotsInfo
		return
	}

	for _, s := range r.Array {
		info := s.([]interface{})
		slots := &Slots{}

		slots.begin = uint16(info[0].(int))
		slots.end = uint16(info[1].(int))

		addrs := info[2].([]interface{})

		for i := 0; i < len(addrs); i++ {
			addr := fmt.Sprintf("%s:%d", addrs[0].(string), addrs[1].(int))
			slots.addrs = append(slots.addrs, addr)
			fmt.Println("slots", slots)
		}

		slotsArray = append(slotsArray, slots)
	}

	fmt.Println("slots:", slotsArray)

	return
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
