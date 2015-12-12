package conn

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	client, err := NewClient("tcp", "127.0.0.1", time.Second * 10)
	if err != nil {
		t.Error(err)
	}
	client.Go("SET", []string{"int", "1"}, nil)
}