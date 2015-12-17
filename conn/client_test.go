package conn

import (
	"time"
//	"runtime/debug"
	"fmt"
	"testing"
)

var client *Client

func TestClient(t *testing.T) {
	var err error
	client, err = NewClient("tcp", "127.0.0.1:6379", time.Second * 10)
	if err != nil {
		t.Fatal(err)
	}
}

func ExampleSetInt() {

	req := client.Go(nil, "SET", "int", 1)
	reply := req.GetReply()
	fmt.Println(reply.Type, reply.Value)
	// Output: 0 OK
}