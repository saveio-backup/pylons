package client

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/themis/common/log"
)

var P2pServerPid *actor.PID

func SetP2pPid(p2pPid *actor.PID) {
	P2pServerPid = p2pPid
}

//------------------------------------------------------------------------------------

type ConnectReq struct {
	Address string
}

type CloseReq struct {
	Address string
}

type SendReq struct {
	Address string
	Data    proto.Message
}
type P2pResp struct {
	Error error
}

type RecvMsg struct {
	From    string
	Message proto.Message
}

type NodeActiveReq struct {
	Address string
}

type NodeActiveResp struct {
	Active bool
}

func P2pConnect(address string) error {
	chReq := &ConnectReq{address}
	future := P2pServerPid.RequestFuture(chReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[P2pConnect] error: ", err)
		return err
	}
	return nil
}

func P2pClose(address string) error {
	chReq := &CloseReq{address}
	future := P2pServerPid.RequestFuture(chReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[P2pClose] error: ", err)
		return err
	}
	return nil
}

func P2pSend(address string, data proto.Message) error {
	chReq := &SendReq{address, data}
	future := P2pServerPid.RequestFuture(chReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[P2pSend] error: ", err)
		return err
	}
	return nil
}
