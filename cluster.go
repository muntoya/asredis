package asredis

import (
	"errors"
	"fmt"
	//"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

var (
	ErrClusterNoService = errors.New("can't connect to any server of cluster")
	ErrSlotsInfo        = errors.New("can't get infomation of slots")
	ErrNoSlot           = errors.New("can't find slot")
)

const (
	numSlots = 16384
	numConn  = 10
	numChan  = 20
)

type nodeAddr struct {
	host	string
	port	int16
}

type mapping [numSlots][]*Pool

type Cluster struct {
	mutex sync.RWMutex

	spec *PoolSpec

	slotsMap mapping
	pools    map[nodeAddr]*Pool
	info     *ClusterInfo
}

type ClusterSlots struct {
	begin uint16
	end   uint16
	nodes []nodeAddr
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

func (c *Cluster) Call(req... *Request) (reply *Reply, err error) {
	key := fmt.Sprint(args[0])
	pools, err := c.getPools(key)
	if err != nil {
		return
	}

	return pools[0].Call(req...)
}

//不使用hash
func (c *Cluster) ExecN(cmd string, args ...interface{}) (reply *Reply, err error) {
	key := fmt.Sprint(args[0])
	pools, err := c.getPools(key)
	if err != nil {
		return
	}

	return pools[0].Call(cmd, args...)
}

func (c *Cluster) getPools(key string) (pools []*Pool, err error) {
	v := CRC16([]byte(key)) % numSlots
	pools = c.slotsMap[v]
	if len(pools) == 0 {
		err = ErrNoSlot
	}
	return
}

func (c *Cluster) connect() error {
	c.updateSlots()

	return nil
}

func (c *Cluster) updateSlots() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
			//fmt.Println(string(debug.Stack()))
		}
	}()

	var pool *Pool
	pool, err = c.getPoolsInfo()
	if err != nil {
		return
	}

	info := getClusterInfo(pool)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.info != nil && c.info.currentEpoch >= info.currentEpoch {
		return
	}

	c.info = info

	slotsArray := getSlotsInfo(pool)
	c.slotsMap = mapping{}
	for _, slots := range slotsArray {

		var pools []*Pool
		for _, node := range slots.nodes {
			pool, ok := c.pools[node]
			if !ok {
				spec := *c.spec
				spec.Host = node.host
				spec.Port = node.port
				pool = NewPool(&spec)
				c.pools[node] = pool
			}
			pools = append(pools, pool)
		}

		for i := slots.begin; i <= slots.end; i++ {
			c.slotsMap[i] = pools
		}
	}
	fmt.Println("pools", c.pools)
	//fmt.Println("slotsmap", c.slotsMap)

	return
}

func (c *Cluster) getPoolsInfo() (pool *Pool, err error) {
	pool = NewPool(c.spec)

	if pool == nil || pool.ConnsFail() > 0 {
		err = ErrClusterNoService
	}

	return pool, err
}

func getSlotsInfo(pool *Pool) (slotsArray []*ClusterSlots) {
	r, err := pool.Call("CLUSTER", "slots")
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
			node := nodeAddr{addrs[i].(string), int16(addrs[i + 1].(int))}
			slots.nodes = append(slots.nodes, node)
		}

		slotsArray = append(slotsArray, slots)
	}

	return
}

func getClusterInfo(pool *Pool) (info *ClusterInfo) {
	r, err := pool.Call("CLUSTER", "info")
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

func getClusterNodes(pool *Pool) {
	r, err := pool.Call("CLUSTER", "nodes")
	fmt.Println(r, err)
}

func (c *Cluster) checkCluster() {

}

func NewCluster(spec *PoolSpec) (cluster *Cluster, err error) {
	cluster = &Cluster{
		spec: spec,
		pools: make(map[nodeAddr]*Pool),
	}

	err = cluster.connect()

	if err != nil {
		return nil, err
	}

	return
}
