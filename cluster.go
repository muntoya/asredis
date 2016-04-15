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
)

type nodeAddr struct {
	host	string
	port	int16
}

type mapping [numSlots]*Pool


type Cluster struct {
	mutex sync.RWMutex
	Spec

	slotsMap mapping
	pools    map[nodeAddr]*Pool
	info     *ClusterInfo
}

type ClusterSlots struct {
	begin uint16
	end   uint16
	node  nodeAddr
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

type requestListMap map[*Pool] []*Request

func (c *Cluster) Call(cmd string, args...interface{}) (interface{}, error) {
	pool, err := c.getPools(args[0])
	if err != nil {
		return nil, err
	}
	return pool.Call(cmd, args...)
}

//不使用hash
func (c *Cluster) Pipelining(reqPkg *RequestsPkg) error {
	if len(reqPkg.requests) == 0 {
		return ErrEmptyReqests
	}
	pool, err := c.getPools(reqPkg.requests[0])
	if err != nil {
		return err
	}
	pool.Pipelining(reqPkg)
	return nil
}

func (c *Cluster) getPools(key interface{}) (pool *Pool, err error) {
	k := fmt.Sprint(key)
	v := CRC16([]byte(k)) % numSlots
	pool = c.slotsMap[v]
	if pool == nil {
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
	pool, err = c.getTempPool()
	checkError(err)

	info := getClusterInfo(pool)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.info != nil && c.info.currentEpoch >= info.currentEpoch {
		return ErrSlotsInfo
	}

	c.info = info

	slotsArray := getSlotsInfo(pool)
	c.slotsMap = mapping{}
	for _, slots := range slotsArray {

		var pool *Pool
		pool, ok := c.pools[slots.node]
		if !ok {
			spec := c.Spec
			spec.Host = slots.node.host
			spec.Port = slots.node.port
			pool = NewPool(&spec)
			c.pools[slots.node] = pool
		}

		for i := slots.begin; i <= slots.end; i++ {
			c.slotsMap[i] = pool
		}
	}
	fmt.Println("pools", c.pools)
	//fmt.Println("slotsmap", c.slotsMap)

	return
}

func (c *Cluster) getTempPool() (pool *Pool, err error) {
	pool = NewPool(&c.PoolSpec)

	if pool == nil || pool.ConnsFail() > 0 {
		err = ErrClusterNoService
	}

	return pool, err
}

func getSlotsInfo(pool *Pool) (slotsArray []*ClusterSlots) {
	reply, err := pool.Call("CLUSTER", "slots")
	checkError(err)

	array, ok := reply.([]interface{})
	if !ok {
		panic(ErrSlotsInfo)
	}

	for _, s := range array {
		info := s.([]interface{})
		slots := &ClusterSlots{}

		slots.begin = uint16(info[0].(int))
		slots.end = uint16(info[1].(int))

		//只取master的slots,方便实现
		addrs := info[2].([]interface{})
		slots.node = nodeAddr{addrs[0].(string), int16(addrs[1].(int))}

		slotsArray = append(slotsArray, slots)
	}

	return
}

func getClusterInfo(pool *Pool) (info *ClusterInfo) {
	reply, err := pool.Call("CLUSTER", "info")
	checkError(err)

	infoStr := string(reply.([]byte))
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
	reply, err := pool.Call("CLUSTER", "nodes")
	fmt.Println(reply, err)
}

func (c *Cluster) checkCluster() {

}

func NewCluster(spec *ClusterSpec) (cluster *Cluster, err error) {
	cluster = &Cluster{
		ClusterSpec: *spec,
		pools: make(map[nodeAddr]*Pool),
	}

	err = cluster.connect()

	if err != nil {
		return nil, err
	}

	return
}
