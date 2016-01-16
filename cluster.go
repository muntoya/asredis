package asredis

type Cluster struct {
	pools	map [string]Pool
}

func NewCluster(addrs string) (cluster *Cluster) {
	return
}
