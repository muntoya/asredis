package asredis

import (
	"crypto/sha1"
	"io"
	"io/ioutil"
	"os"
	"encoding/hex"
	//"fmt"
)

//lua脚本支持
type LuaEval struct {
	fileName    string
	hash        string
}

func (this *LuaEval) readFile() (content string, err error) {
	var f *os.File
	f, err = os.Open(this.fileName)
	if err != nil {
		return
	}

	var c []byte
	c, err = ioutil.ReadAll(f)
	content = string(c)
	return
}

func NewLuaEval(fileName string) (l *LuaEval, err error) {
	var f *os.File
	f, err = os.Open(fileName)
	if err != nil {
		return
	}

	h := sha1.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return
	}

	l = &LuaEval{
		fileName: fileName,
		hash: hex.EncodeToString(h.Sum(nil)),
	}

	return
}