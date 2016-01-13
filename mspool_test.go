package asredis

import (
	"testing"
)

func TestMSPool(t *testing.T) {
	t.Skip("skip master slave pool")
	_ = NewMSPool("mymaster", []string{"127.0.0.1:26379"})
}
