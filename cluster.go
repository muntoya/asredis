package asredis

import (
	"errors"
	"fmt"
	"log"
)

var (
	ErrClusterNoService = errors.New("can't connect to any server of cluster")
	ErrSlotsInfo        = errors.New("can't get infomation of slots")
)

const numSlots = 16384

type mapping [numSlots]*CPool

type CPool struct {
	*Pool
}

type Cluster struct {
	mapping
	pools []*CPool
	addrs []string
	slots []*ClusterSlots
}

type ClusterSlots struct {
	begin uint16
	end   uint16
	addrs []string
}

type ClusterInfo struct {
	state         bool
	slotsAssinged int
	slotsOK       bool
	slotsPFial    int
	slotsFail     int
	knownNodes    int
	size          int
	currentEpoch  int
	myEpoch       int
}

func (c *Cluster) connect() error {
	c.updateSlots()

	return nil
}

func (c *Cluster) updateSlots() (err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Panicf("update slots info error: %v", err)
		}
	}()

	var pool *Pool
	pool, err = c.getPool()
	if err != nil {
		return
	}

	getInfo(pool)

	var slotsArray []*ClusterSlots
	slotsArray, err = getSlots(pool)
	fmt.Println("slots:", slotsArray)

	return
}

func (c *Cluster) getPool() (pool *Pool, err error) {
	for _, addr := range c.addrs {
		pool = NewPool(addr, 1, 1)
		if pool.ConnsFail() > 0 {
			continue
		}
	}

	if pool == nil || pool.ConnsFail() > 0 {
		err = ErrClusterNoService
	}

	return pool, err
}

func getSlots(pool *Pool) (slotsArray []*ClusterSlots, err error) {

	var r *Reply
	r, err = pool.Exec("CLUSTER", "slots")
	if err != nil {
		return
	}

	if r.Type != ARRAY {
		err = ErrSlotsInfo
		return
	}

	for _, s := range r.Array {
		info := s.([]interface{})
		slots := &ClusterSlots{}

		slots.begin = uint16(info[0].(int))
		slots.end = uint16(info[1].(int))

		addrs := info[2].([]interface{})

		for i := 0; i < len(addrs); i += 2 {
			addr := fmt.Sprintf("%s:%d", addrs[0].(string), addrs[1].(int))
			slots.addrs = append(slots.addrs, addr)
			fmt.Println("slots", slots)
		}

		slotsArray = append(slotsArray, slots)
	}

	return
}

func getInfo(pool *Pool) (info *ClusterInfo, err error) {
	var r *Reply
	r, err = pool.Exec("CLUSTER", "info")
	fmt.Println(r, err)
	return
}

func getNodes(pool *Pool) {
	r, err := pool.Exec("CLUSTER", "nodes")
	fmt.Println(r, err)
}

func (c *Cluster) checkCluster() {

}

func NewCluster(addrs []string) (cluster *Cluster, err error) {
	cluster = &Cluster{
		addrs:      addrs,
	}

	err = cluster.connect()

	if err != nil {
		return nil, err
	}

	return
}
