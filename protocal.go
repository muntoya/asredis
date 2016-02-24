package asredis

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"log"
)

var (
	ErrExpectingLinefeed   = errors.New("redis: expecting a linefeed byte")
	ErrUnexpectedReplyType = errors.New("redis: can't parse reply type")
)

const (
	cr_byte    byte = byte('\r')
	lf_byte         = byte('\n')
	space_byte      = byte(' ')
	err_byte        = byte('-')
	ok_byte         = byte('+')
	array_byte      = byte('*')
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

type Reply struct {
	Type  ResponseType
	Value interface{}
	Array []interface{}
}

type requestType byte

const (
	type_normal requestType = iota
	type_only_send
	type_only_wait

	//control command
	type_ctrl_begin	//begin标记
	type_ctrl_reconnect
	type_ctrl_shutdown
	type_ctrl_end	//end标记
)

type Request struct {
	cmd     string
	args    []interface{}
	Err     error
	Reply   *Reply
	reqtype requestType
}

func NewRequest(cmd string, args ...interface{}) *Request {
	return NewRequestType(type_normal, cmd, args...)
}

func NewRequestType(reqtype requestType, cmd string, args ...interface{}) *Request {
	req := new(Request)
	req.reqtype = reqtype
	req.cmd = cmd
	req.args = args
	return req
}

type requestsPkg struct {
	requests	[]*Request
	Done chan struct{}
}

func (r *requestsPkg) done() {
	if r.Done != nil {
		r.Done <- r
	}
}

func (r *requestsPkg) wait() {
	if r.Done != nil {
		<-r.Done
	}
}

func readToCRLF(io *bufio.Reader) []byte {
	buf, err := io.ReadBytes(cr_byte)
	checkError(err)

	var b byte
	b, err = io.ReadByte()
	checkError(err)

	if b != lf_byte {
		panic(ErrExpectingLinefeed)
	}

	return buf[0 : len(buf)-1]
}

//读取一个完整的回复数据
func readReply(io *bufio.Reader) (reply *Reply) {
	if io == nil {
		panic(ErrNotConnected)
	}

	b := readToCRLF(io)
	reply = new(Reply)

	switch v := string(b[1:]); b[0] {
	case ok_byte:
		reply.Type = STRING
		reply.Value = string(v)

	case err_byte:
		reply.Type = ERROR
		reply.Value = string(v)

	case num_byte:
		reply.Type = INTEGER
		i, err := strconv.Atoi(string(v))
		checkError(err)
		reply.Value = i

	case size_byte:
		reply.Type = BULK
		len, err := strconv.Atoi(v)
		checkError(err)

		s, err := io.Peek(len)
		checkError(err)

		l, err := io.Discard(len)
		checkError(err)

		readToCRLF(io)

		reply.Value = string(s[0:l])

	case array_byte:
		len, err := strconv.Atoi(v)
		checkError(err)

		reply.Type = ARRAY
		reply.Array = readArray(io, len)

	default:
		panic(ErrUnexpectedReplyType)
	}

	return
}

//递归读取数组数据
func readArray(io *bufio.Reader, len int) (array []interface{}) {
	array = make([]interface{}, len)
	for i := 0; i < len; i++ {
		s := readToCRLF(io)
		switch s[0] {
		case num_byte:
			ele, err := strconv.Atoi(string(s[1:]))
			checkError(err)
			array[i] = ele
		case size_byte:
			ele := readToCRLF(io)
			array[i] = string(ele)
		case array_byte:
			l, err := strconv.Atoi(string(s[1:]))
			checkError(err)
			array[i] = readArray(io, l)

		default:
			panic(ErrUnexpectedReplyType)
		}
	}
	return
}

func writeReqToBuf(buf *bufio.Writer, req *Request) {
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
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
