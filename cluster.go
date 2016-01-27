package asredis

import (
	"errors"
	"fmt"
	"log"
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
	slots   []*ClusterSlots

	configChan  chan *Request
}

type ClusterSlots struct {
	begin	uint16
	end		uint16
	addrs	[]string
}

type ClusterInfo struct {

}

func (c *Cluster) connect() (error) {
	c.updateSlots()

	return nil
}

func (c *Cluster) updateSlots() (err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Panicf("update slots info error: %v", err)
		}
	}()

	var conn *Connection
	conn, err = c.getConn()
	if err != nil {
		return
	}

	var slotsArray []*ClusterSlots
	slotsArray, err = getSlots(conn, c.configChan)
	fmt.Println("slots:", slotsArray)

	return
}

func (c *Cluster) getConn() (conn *Connection, err error) {
	for _, addr := range c.addrs {
		conn = NewConnection(addr)
		if conn.isConnected() {
			break
		}
	}

	if conn == nil || !conn.isConnected() {
		err = ErrClusterNoService
	}

	return conn, err
}

func getSlots(conn *Connection, ch chan *Request) (slotsArray []*ClusterSlots, err error) {

	var r *Reply
	r, err = conn.call(ch, "CLUSTER", "slots")
	if err != nil {
		return
	}

	if (r.Type != ARRAY) {
		err = ErrSlotsInfo
		return
	}

	for _, s := range r.Array {
		info := s.([]interface{})
		slots := &ClusterSlots{}

		slots.begin = uint16(info[0].(int))
		slots.end = uint16(info[1].(int))

		addrs := info[2].([]interface{})

		for i := 0; i < len(addrs); i+=2 {
			addr := fmt.Sprintf("%s:%d", addrs[0].(string), addrs[1].(int))
			slots.addrs = append(slots.addrs, addr)
			fmt.Println("slots", slots)
		}

		slotsArray = append(slotsArray, slots)
	}

	return
}

func getInfo(conn *Connection, ch chan *Request) {
	r, err := conn.call(ch, "CLUSTER", "info")
	fmt.Println(r, err)
}

func getNodes(conn *Connection, ch chan *Request) {
	r, err := conn.call(ch, "CLUSTER", "nodes")
	fmt.Println(r, err)
}

func (c *Cluster) checkCluster() {

}

func NewCluster(addrs []string) (cluster *Cluster, err error) {
	cluster = &Cluster {
		addrs: addrs,
		configChan: make(chan *Request, 1),
	}

	err = cluster.connect()

	if err != nil {
		return nil, err
	}
	
	return
}
