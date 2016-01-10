package asredis

import (
//	"time"
//	"runtime/debug"
	"testing"
//	"github.com/stretchr/testify/assert"
	"fmt"
)

func TestSConnection(t *testing.T) {
	sconn := NewSConnection("127.0.0.1:26379")
	ps, _ := sconn.GetMasters()
	fmt.Println(ps[0])

	p,err := sconn.GetMaster("mymaster")
	fmt.Println(p, err)

	ps, err = sconn.GetSlaves("mymaster")
	fmt.Println(ps[0], err)

	ip, port ,err := sconn.GetMasterAddr("mymaster")
	fmt.Println(ip, port)
}
