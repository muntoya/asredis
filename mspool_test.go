package asredis

import (
	"testing"
)

func TestMSPool(t *testing.T) {
	_ = NewMSPool("mymaster", []string{"127.0.0.1:26379"})
}
