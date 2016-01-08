package asredis
import (
	"log"
	//"fmt"
)

//sentinel connetcion,工具类,用来实现集群的连接

type ConnProperty struct {
	name, ip, port, runid, flags string
}

type SConnection struct {
	*Connection
	commandChan chan *Request
}

func (this *SConnection) GetMasters() (ppArray []*ConnProperty, err error) {
	var reply *Reply
	reply, err = this.Call(this.commandChan, "sentinel", "masters")
	if err != nil {
		return
	}

	var m interface{}
	for _, m = range reply.Array {
		array, e := m.([]interface{})
		if !e {
			log.Panicln("can't parse masters")
		}

		var property ConnProperty
		for i := 0; i < len(array); i += 2 {

			key := array[i].(string)
			value := array[i+1].(string)
			switch key {
			case "name":
				property.name = value
			case "ip":
				property.ip = value
			case "port":
				property.runid = value
			case "flags":
				property.flags = value
			default:
				//log.Println("no key", key)
			}
		}
		ppArray = append(ppArray, &property)
	}

	return ppArray, nil
}

func NewSConnection(addr string) *SConnection {
	return &SConnection{
		Connection: NewConnection(addr),
		commandChan: make(chan *Request, 1),
	}
}
