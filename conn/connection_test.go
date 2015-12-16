package conn

import (
	"testing"
	"time"
)

func TestCon(t *testing.T) {
	conn, err := DialTimeout("tcp", "127.0.0.1:6379", time.Second * 10)

	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(conn)
	}
}
