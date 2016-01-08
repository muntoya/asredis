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
}
