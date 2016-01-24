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
		var i int
		i, err = strconv.Atoi(info[0].(string))
		if err != nil {
			return
		}
		slots.begin = uint16(i)

		i, err = strconv.Atoi(info[1].(string))
		if err != nil {
			return
		}
		slots.end = uint16(i)

		for _, c := range info[2].([]interface{}) {
			append(slots.addrs, strings.Join(c.([]string), ":"))
		}

		append(slotsArray, slots)
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
