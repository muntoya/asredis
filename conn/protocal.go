package conn

import (
	"io"
	"github.com/muntoya/asredis/common"
)

const (
	cr_byte    byte = byte('\r')
	lf_byte         = byte('\n')
	space_byte      = byte(' ')
	err_byte        = byte('-')
	ok_byte         = byte('+')
	count_byte      = byte('*')
	size_byte       = byte('$')
	num_byte        = byte(':')
	true_byte       = byte('1')
)

func sendRequest(w io.Writer, data []byte) {
	loginfo := "sendRequest"
	if w == nil {
		panic(common.NewRedisError("<BUG> %s() - nil Writer"))
	}

	n, e := w.Write(data)
	if e != nil {
		panic(common.NewRedisErrorf("%s() - connection Write wrote %d bytes only.", loginfo, n))
	}

	if n < len(data) {
		panic(common.NewRedisErrorf("%s() - connection Write wrote %d bytes only.", loginfo, n))
	}
}


