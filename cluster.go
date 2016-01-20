package asredis

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

func (c *Cluster) connect() {

}

func (c *Cluster) checkCluster() {

}

func NewCluster(addrs []string) (cluster *Cluster) {
	cluster = &Cluster {
		addrs: addrs,
	}

	cluster.connect()
	
	return
}
