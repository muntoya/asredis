package conn

import (
	"time"
//	"runtime/debug"

	"testing"
	"github.com/stretchr/testify/assert"
)

var client *Client

func TestClient(t *testing.T) {
	var err error
	client, err = NewClient("tcp", "127.0.0.1:6379", time.Second * 10)
	if err != nil {
		t.Fatal(err)
	}

	req := client.Go(nil, "SET", "int", 1)
	reply := req.GetReply()

	assert.Equal(t, reply.Type, STRING)
	assert.Equal(t, reply.Value, "OK")
}
