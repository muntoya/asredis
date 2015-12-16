package conn

import (
	"io"
	"bufio"
	"strconv"
//	"fmt"
	"github.com/muntoya/asredis/common"
)

const (
	cr_byte    byte = byte('\r')
	lf_byte         = byte('\n')
	space_byte      = byte(' ')
	err_byte        = byte('-')
	ok_byte         = byte('+')
	array_byte 		= byte('*')
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
	NIL
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

	switch  v := string(b[1:]); b[0] {
	case ok_byte:
		reply.Type = STRING
		reply.Value = string(v)

	case err_byte:
		reply.Type = STRING
		reply.Value = string(v)

	case num_byte:
		reply.Type = INTEGER
		i, err := strconv.Atoi(string(v))
		if err != nil {
			panic(common.NewRedisErrorf("readReply: atoi can't convert %s", v))
		} else {
			reply.Value = i
		}

	case size_byte:
		len, err := strconv.Atoi(v)
		if err != nil {
			panic(common.NewRedisErrorf("readReply: atoi can't convert %s", v))
		}
		if len == -1 {
			reply.Type = NIL
		} else {
			s := readToCRLF(io)
			reply.Value = string(s[1:])
		}

	case array_byte:
		len, err :=  strconv.Atoi(v)
		if err != nil {
			panic(common.NewRedisErrorf("readReply: atoi can't convert %s", v))
		}

		reply.Array = make([]interface{}, len)
		for i := 0; i < len; i++ {
			s := readToCRLF(io)
			switch s[0] {
			case num_byte:
				ele, err := strconv.Atoi(string(s[1:]))
				if err != nil {
					panic(common.NewRedisErrorf("readReply: atoi can't convert %s", v))
				}
				reply.Array[i] = ele
			case size_byte:
				ele := readToCRLF(io)
				reply.Array[i] = ele

			default:
				panic(common.NewRedisErrorf("readReply: atoi can't convert %s", s[0]))
			}
		}

	default:
		panic(common.NewRedisErrorf("readReply: can parse type %s", b))
	}
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


