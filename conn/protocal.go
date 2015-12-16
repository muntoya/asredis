package conn

import (
	"io"
	"bufio"
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

var cr_lf = []byte{cr_byte, lf_byte}

type ResponseType int

const (
	STRING ResponseType = iota
	ERROR
	INTEGER
	BULK
	ARRAY
)

func readToCRLF(io *bufio.Reader) []byte {
	buf, e := io.ReadBytes(cr_byte)
	if e != nil {
		panic(common.NewRedisErrorf("readToCRLF - ReadBytes", e))
	}

	var b byte
	b, e = io.ReadByte()
	if e != nil {
		panic(common.NewRedisErrorf("readToCRLF - ReadByte", e))
	}
	if b != lf_byte {
		e = common.NewRedisError("<BUG> Expecting a Linefeed byte here!")
	}
	return buf[0 : len(buf)-1]
}

func readReply(io *bufio.Reader, reply *Reply) {
	b := readToCRLF(io)
	switch b[0] {
	case err_byte:
	case ok_byte:
	case count_byte:
	case size_byte:
	case num_byte:
	default:

	}
	reply.Value = string(b)
}

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


