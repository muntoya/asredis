package conn

import (
	"testing"
	"time"
//	"runtime/debug"
//	"fmt"
)

func TestClient(t *testing.T) {
//	defer func() {
//		if p := recover(); p != nil {
//			t.Error(fmt.Sprint("%s  %s", p, string(debug.Stack())))
//		}
//	}()

	client, err := NewClient("tcp", "127.0.0.1:6379", time.Second * 10)

	if err != nil {
		t.Error(err)
	}

	req := client.Go("SET", []string{"int", "1"}, nil)
	<- req.Done
}