package conn

import (
	"time"
//	"runtime/debug"
	"fmt"
)


func ExampleClient() {
	client, _ := NewClient("tcp", "127.0.0.1:6379", time.Second * 10)

	req := client.Go(nil, "SET", "int", 1)
	reply := req.GetReply()
	fmt.Println(reply.Type, reply.Value)
	// Output: 0 OK
}