package asredis
import "fmt"

//sentinel connetcion,工具类,用来实现集群的连接

type ConnProperty struct {
	name, ip, port, runid, flags string
}

type SConnection struct {
	*Connection
	commandChan chan *Request
}

func (this *SConnection) GetMasters() (pp *ConnProperty, err error) {
	var reply *Reply
	reply, err = this.Call(this.commandChan, "sentinel", "masters")
	if err != nil {
		return
	}

	fmt.Println(reply.Array)
	
	return pp, nil
}

func NewSConnection(addr string) *SConnection {
	return &SConnection{
		Connection: NewConnection(addr),
		commandChan: make(chan *Request, 1),
	}
}
