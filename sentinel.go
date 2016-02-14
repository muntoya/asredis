package asredis
import (
	"log"
	//"fmt"
)

//sentinel connetcion,工具类,用来实现集群的连接

type ConnProp struct {
	name, ip, port, runid, flags string
}

type SConnection struct {
	*Connection
	commandChan chan *Request
}

func (s *SConnection) GetMasters() (ppArray []*ConnProp, err error) {
	var reply *Reply
	reply, err = s.call(s.commandChan, "SENTINEL", "masters")
	if err != nil {
		return
	}

	ppArray, err = readConnPropArray(reply.Array)
	return
}

func (s *SConnection) GetMaster(master string) (pp *ConnProp, err error) {
	var reply *Reply
	reply ,err = s.call(s.commandChan, "SENTINEL", "master", master)
	if err != nil {
		return
	}
	pp, err = readConnProp(reply.Array)
	return
}

func (s *SConnection) GetMasterAddr(master string) (ip, port string, err error) {
	var reply *Reply
	reply, err = s.call(s.commandChan, "SENTINEL", "get-master-addr-by-name", master)
	if err != nil {
		return
	}
	array := reply.Array
	if len(array) != 2 {
		log.Panic("can't parse master's ip and port")
	}
	ip = array[0].(string)
	port = array[1].(string)
	return
}

func (s *SConnection) GetSlaves(master string) (ppArray []*ConnProp, err error) {
	var reply *Reply
	reply ,err = s.call(s.commandChan, "SENTINEL", "slaves", master)
	if err != nil {
		return
	}

	ppArray, err = readConnPropArray(reply.Array)
	return
}

func NewSConnection(spec *ConnectionSpec) *SConnection {
	return &SConnection{
		Connection: NewConnection(spec),
		commandChan: make(chan *Request, 1),
	}
}

func readConnPropArray(itemArray []interface{}) (ppArray []*ConnProp, err error) {
	for _, m := range itemArray {
		array, ok := m.([]interface{})
		if !ok {
			log.Panicln("can't parse masters")
		}
		var pp *ConnProp
		pp, err = readConnProp(array)
		if err != nil {
			return
		}

		ppArray = append(ppArray, pp)
	}
	return
}

func readConnProp (array []interface{}) (pp *ConnProp, err error) {
	pp = new(ConnProp)
	for i := 0; i < len(array); i += 2 {
		key := array[i].(string)
		value := array[i+1].(string)
		switch key {
		case "name":
			pp.name = value
		case "ip":
			pp.ip = value
		case "port":
			pp.runid = value
		case "flags":
			pp.flags = value
		default:
			//log.Println("no key", key)
		}
	}
	return
}
