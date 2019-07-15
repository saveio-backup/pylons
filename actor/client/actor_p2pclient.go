package client

import (
	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
)

var P2pServerPid *actor.PID

func SetP2pPid(p2pPid *actor.PID) {
	P2pServerPid = p2pPid
}

//------------------------------------------------------------------------------------

type ConnectRet struct {
	Done chan bool
	Err  error
}

type ConnectReq struct {
	Address string
	Ret *ConnectRet
}

type CloseRet struct {
	Done chan bool
	Err  error
}

type CloseReq struct {
	Address string
	Ret *CloseRet
}

type SendRet struct {
	Done chan bool
	Err  error
}

type SendReq struct {
	Address string
	Data    proto.Message
	Ret *SendRet
}

type RecvMsgRet struct {
	Done chan bool
	Err  error
}

type RecvMsg struct {
	From    string
	Message proto.Message
	Ret *RecvMsgRet
}


func P2pConnect(address string) error {
	ret := &ConnectRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	conRet := &ConnectReq{Address:address, Ret:ret}
	P2pServerPid.Tell(conRet)
	<-conRet.Ret.Done
	return conRet.Ret.Err
}

func P2pClose(address string) error {
	ret := &CloseRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	chReq := &CloseReq{Address:address, Ret:ret}
	P2pServerPid.Tell(chReq)
	<-chReq.Ret.Done
	return chReq.Ret.Err
}

func P2pSend(address string, data proto.Message) error {
	ret := &SendRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	chReq := &SendReq{Address:address, Data:data, Ret:ret}
	P2pServerPid.Tell(chReq)
	<-chReq.Ret.Done
	return chReq.Ret.Err
}
