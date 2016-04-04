package asredis

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
)

var (
	ErrExpectingLinefeed   = errors.New("redis: expecting a linefeed byte")
	ErrUnexpectedReplyType = errors.New("redis: can't parse reply type")
	ErrNilReply            = errors.New("redis: reply is nil")
)

const (
	okReply   = "OK"
	pongReply = "PONG"
)

type Error string

func (err Error) Error() string { return string(err) }

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


type ctrlType byte

const (
	//control command
	type_ctrl_begin ctrlType = iota //begin标记
	type_ctrl_reconnect
	type_ctrl_shutdown
	type_ctrl_end //end标记
)

type Request struct {
	cmd   string
	args  []interface{}
	Err   error
	Reply interface{}
}

func NewRequstPkg() *RequestsPkg {
	r := &RequestsPkg{
		d: make(chan struct{}, 1),
	}
	return r
}

type RequestsPkg struct {
	requests []*Request
	d        chan struct{}
}

func (r *RequestsPkg) Add(cmd string, args ...interface{}) {
	req := &Request{
		cmd:  cmd,
		args: args,
	}
	r.requests = append(r.requests, req)
}

func (r *RequestsPkg) done() {
	r.d <- struct{}{}
}

func (r *RequestsPkg) wait() {
	if r.d != nil {
		<-r.d
	}
}

//返回第一个请求的错误
func (r *RequestsPkg) err() error {
	if len(r.requests) != 0 {
		return r.requests[0].Err
	}
	return nil
}

//返回第一个请求的回复
func (r *RequestsPkg) reply() interface{} {
	if len(r.requests) != 0 {
		return r.requests[0].Reply
	}
	return nil
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
func readReply(io *bufio.Reader) (reply interface{}, err error) {
	if io == nil {
		panic(ErrNotConnected)
	}

	b := readToCRLF(io)
	switch v := string(b[1:]); b[0] {
	case ok_byte:
		switch {
		case v == okReply:
			reply = okReply
		case v == pongReply:
			reply = pongReply
		default:
			reply = v
		}

	case err_byte:
		err = Error(v)

	case num_byte:
		i, err := strconv.Atoi(v)
		checkError(err)
		reply = i

	case size_byte:
		var size int
		size, err = strconv.Atoi(v)
		checkError(err)

		if size < 0 {
			reply = nil
		}

		s, err := io.Peek(size)
		checkError(err)

		l, err := io.Discard(size)
		checkError(err)

		readToCRLF(io)

		reply = string(s[0:l])

	case array_byte:
		var size int
		size, err = strconv.Atoi(v)
		checkError(err)

		r := make([]interface{}, size)
		for i := 0; i < size; i++ {
			r[i], err = readReply(io)
			if err != nil {
				return nil, err
			}
		}
		reply = r

	default:
		panic(ErrUnexpectedReplyType)
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
		//TODO: 判断参数的类型
		switch arg := arg.(type) {
		case string:
			arg
		}
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
