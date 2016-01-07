package asredis

import (
	"testing"
//	"fmt"
	"github.com/stretchr/testify/assert"
)

func TestPubsub(t *testing.T) {
	clientSubpub := NewPubsubClient("127.0.0.1:6379")
	clientSubpub.Sub("c1", "c2")

	client := NewConnection("127.0.0.1:6379")
	c := make(chan *Request, 1)
	_, err := client.Call(c, "PUBLISH", "c1", "haha")
	assert.Exactly(t, nil, err)
	_, err = client.Call(c, "PUBLISH", "c2", "heihei")
	assert.Exactly(t, nil, err)

	msg := clientSubpub.GetMessage(0)
	assert.Exactly(t, "c1", msg.Channel)
	assert.Exactly(t, "haha", msg.Value)
	msg = clientSubpub.GetMessage(0)
	assert.Exactly(t, "c2", msg.Channel)
	assert.Exactly(t, "heihei", msg.Value)
}
