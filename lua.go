package asredis

import (
	"crypto/sha1"
	"io"
	"io/ioutil"
	"os"
)

//lua脚本支持
type LuaEval struct {
	fileName    string
	hash        string
}

func (this *LuaEval) readFile() (content []byte, err error) {
	var f *os.File
	f, err = os.Open(this.fileName)
	if err != nil {
		return
	}

	content, err = ioutil.ReadAll(f)
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
		hash: string(h.Sum(nil)),
	}
	return
}