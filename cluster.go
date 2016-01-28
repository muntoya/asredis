package asredis

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"strconv"
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
	info    *ClusterInfo
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
			log.Printf("update slots info error: %v", err)
		}
	}()

	var pool *Pool
	pool, err = c.getPool()
	if err != nil {
		return
	}

	info := getInfo(pool)
	fmt.Println(info)

	slotsArray := getSlots(pool)
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

func getSlots(pool *Pool) (slotsArray []*ClusterSlots) {
	r, err := pool.Exec("CLUSTER", "slots")
	checkError(err)

	if r.Type != ARRAY {
		panic(ErrSlotsInfo)
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
		}

		slotsArray = append(slotsArray, slots)
	}

	return
}

func getInfo(pool *Pool) (info *ClusterInfo) {
	r, err := pool.Exec("CLUSTER", "info")
	checkError(err)

	infoStr := r.Value.(string)
	infoStr = strings.Trim(infoStr, "\r\n ")
	attrs := strings.Split(infoStr, "\r\n")

	info = new(ClusterInfo)
	for _, attr := range attrs {
		attr = strings.Trim(attr, "\r\n ")
		kv := strings.SplitN(attr, ":", 2)
		k, v := kv[0], kv[1]
		switch k {
		case "cluster_state":
			if v == "ok" {
				info.state = true
			} else {
				info.state = false
			}

		case "cluster_slots_assigned":
			i, err := strconv.Atoi(v)
			checkError(err)
			info.slotsAssinged = i

		case "cluster_current_epoch":
			i, err := strconv.Atoi(v)
			checkError(err)
			info.currentEpoch = i
		}
	}
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
