package conn

import (
	"bufio"
	"strconv"
	"bytes"
	"fmt"
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

type ResponseType byte

const (
	NIL ResponseType = iota
	STRING
	ERROR
	INTEGER
	BULK
	ARRAY
)

func readToCRLF(io *bufio.Reader) []byte {
	buf, err := io.ReadBytes(cr_byte)
	if err != nil {
		panic(err)
	}

	var b byte
	b, err = io.ReadByte()
	if err != nil {
		panic(err)
	}

	if b != lf_byte {
		panic(common.ErrExpectingLinefeed)
	}

	return buf[0 : len(buf)-1]
}

func readReply(io *bufio.Reader, reply *Reply)  error {
	defer func() {
		if err := recover(); err != nil {

		}
	}()

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
			return err
		} else {
			reply.Value = i
		}

	case size_byte:
		reply.Type = INTEGER
		len, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		s := readToCRLF(io)
		reply.Value = string(s[0:len])

	case array_byte:
		len, err :=  strconv.Atoi(v)
		if err != nil {
			return err
		}

		reply.Type = ARRAY

		reply.Array = make([]interface{}, len)
		for i := 0; i < len; i++ {
			s := readToCRLF(io)
			switch s[0] {
			case num_byte:
				ele, err := strconv.Atoi(string(s[1:]))
				if err != nil {
					return err
				}
				reply.Array[i] = ele
			case size_byte:
				ele := readToCRLF(io)
				reply.Array[i] = ele

			default:
				return err
			}
		}

	default:
		return common.ErrUnexpectedReplyType
	}

	return nil
}

func writeReqToBuf(buf *bytes.Buffer, req *RequestInfo) (b []byte, err error) {
	buf.Reset()

	//写入参数个数
	argsCnt := len(req.args) + 1
	buf.WriteByte(array_byte)
	buf.WriteString(strconv.Itoa(argsCnt))
	buf.Write(cr_lf)

	//写入命令
	buf.WriteByte(size_byte)
	buf.WriteString(strconv.Itoa(len(req.cmd)))
	buf.Write(cr_lf)
	buf.WriteString(req.cmd)
	buf.Write(cr_lf)

	//写入参数
	for _, arg := range req.args {
		v := fmt.Sprint(arg)
		buf.WriteByte(size_byte)
		buf.WriteString(strconv.Itoa(len(v)))
		buf.Write(cr_lf)

		buf.WriteString(v)
		buf.Write(cr_lf)
	}

	return buf.Bytes(), nil
}
