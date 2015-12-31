package asredis

import (
	"testing"
//	"fmt"
	"github.com/stretchr/testify/assert"
//	"fmt"
)

func TestPubsub(t *testing.T) {
	clientSubpub := NewPubsubClient("tcp", "127.0.0.1:6379")
	clientSubpub.Sub("c1", "c2")

	client := NewClient("tcp", "127.0.0.1:6379")
	c := make(chan *RequestInfo, 1)
	_, err := client.Go(c, "PUBLISH", "c1", "haha").GetReply()
	assert.Exactly(t, nil, err)
	_, err = client.Go(c, "PUBLISH", "c2", "heihei").GetReply()
	assert.Exactly(t, nil, err)
}
