package protobuf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/golang/protobuf/proto"
)

// -------------------------
// | id | protobuf message |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[uint16]*MsgInfo
	msgID        map[reflect.Type]uint16
	defaltHandle MsgHandler
}

type MsgInfo struct {
	msgType reflect.Type
	//	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
}

type MsgHandler func([]interface{})

type MsgRaw struct {
	msgID      uint16
	msgRawData []byte
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgID = make(map[reflect.Type]uint16)
	p.msgInfo = make(map[uint16]*MsgInfo)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

func (p *Processor) RegisterWithId(msg proto.Message, id uint16) uint16 {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		fmt.Printf("protobuf message pointer required")
	}
	if _, ok := p.msgID[msgType]; ok {
		fmt.Printf("message %s is already registered", msgType)
	}
	if len(p.msgInfo) >= math.MaxUint16 {
		fmt.Printf("too many protobuf messages (max = %v)", math.MaxUint16)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[id] = i
	p.msgID[msgType] = id
	return id
}

//modify end--------------------------------------
// It's dangerous to call the method on routing or marshaling (unmarshaling)
/*
func (p *Processor) SetRouter(msg proto.Message, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		fmt.Printf("message %s not registered", msgType)
	}

	p.msgInfo[id].msgRouter = msgRouter
}
*/
// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetDefaultHandle(msgHandler MsgHandler) {

	if msgHandler == nil {

		fmt.Printf("msgHandle is nil")
		return
	}
	p.defaltHandle = msgHandler
}
func (p *Processor) SetHandler(msg proto.Message, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		fmt.Printf("message %s not registered", msgType)
	}

	p.msgInfo[id].msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
//设置原始数据回调, 发送原始数据
func (p *Processor) SetRawHandler(id uint16, msgRawHandler MsgHandler) {

	p.msgInfo[id].msgRawHandler = msgRawHandler
}

// goroutine safe
func (p *Processor) Route(msg interface{}, userData interface{}) error {
	// raw
	if msgRaw, ok := msg.(MsgRaw); ok {

		i := p.msgInfo[msgRaw.msgID]
		if i.msgRawHandler != nil {
			i.msgRawHandler([]interface{}{msgRaw.msgID, msgRaw.msgRawData, userData})
		}
		return nil
	}

	// protobuf
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		return fmt.Errorf("message %s not registered", msgType)
	}
	i := p.msgInfo[id]
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}

	return nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	if len(data) < 2 {
		return nil, errors.New("protobuf data too short")
	}

	// id
	var id uint16
	if p.littleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}

	if p.msgInfo[id] == nil {
		return nil, fmt.Errorf("message id %v not registered", id)
	}
	//modify end----------------------------------------
	// msg
	i := p.msgInfo[id]
	if i.msgRawHandler != nil {
		return MsgRaw{id, data[2:]}, nil
	} else {
		msg := reflect.New(i.msgType.Elem()).Interface()
		return msg, proto.UnmarshalMerge(data[2:], msg.(proto.Message))
	}
}

// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)

	// id
	_id, ok := p.msgID[msgType]
	if !ok {
		err := fmt.Errorf("message %s not registered", msgType)
		return nil, err
	}

	id := make([]byte, 2)
	if p.littleEndian {
		binary.LittleEndian.PutUint16(id, _id)
	} else {
		binary.BigEndian.PutUint16(id, _id)
	}

	// data
	data, err := proto.Marshal(msg.(proto.Message))
	return [][]byte{id, data}, err
}

// goroutine safe
func (p *Processor) Range(f func(id uint16, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(uint16(id), i.msgType)
	}
}
