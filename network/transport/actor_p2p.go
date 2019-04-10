package transport

import (
	"github.com/oniio/oniChain/common/log"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/gogo/protobuf/proto"
	tranCrypto "github.com/oniio/oniP2p/crypto"
	"time"
	"github.com/oniio/oniChannel/common/constants"
)

var P2pServerPid *actor.PID

type P2plActorServer struct {
	props    *actor.Props
	localPID *actor.PID
	p2p      *P2pNetwork
}

func NewP2pActor(config map[string]string, keys *tranCrypto.KeyPair) (*P2plActorServer, error) {
	p2plActorServer := &P2plActorServer{}
	p2plActorServer.p2p = NewNetWork()
	p2plActorServer.p2p.SetKeys(keys)
	p2plActorServer.p2p.SetProtocol(config["protocol"])
	p2plActorServer.p2p.SetAddress(config["listenAddr"])
	p2plActorServer.p2p.SetMappingAddress(config["mappingAddr"])
	return p2plActorServer, nil
}

func (this *P2plActorServer) Start() error {
	var err error
	this.props = actor.FromProducer(func() actor.Actor { return this })
	this.localPID, err = actor.SpawnNamed(this.props, "p2p_server")
	P2pServerPid = this.localPID
	err = this.p2p.Start()
	if err != nil {
		log.Error("[P2plActorServer] P2plActorServer Start error: ", err.Error())
	}
	return err
}

func (this *P2plActorServer) Stop() {
	this.p2p.Stop()
}

func (this *P2plActorServer) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Restarting:
		log.Warn("[P2plActorServer] Actor restarting")
	case *actor.Stopping:
		log.Warn("[P2plActorServer] Actor stopping")
	case *actor.Stopped:
		log.Warn("[P2plActorServer] Actor stopped")
	case *actor.Started:
		log.Debug("[P2plActorServer] Actor started")
	case *actor.Restart:
		log.Warn("[P2plActorServer] Actor restart")

	case *ConnectReq:
		this.p2p.Connect(msg.address)
		ctx.Sender().Request(ProcessResp{}, ctx.Self())
	case *CloseReq:
		this.p2p.Close(msg.address)
		ctx.Sender().Request(ProcessResp{}, ctx.Self())
	case *SendReq:
		this.p2p.Send(msg.data, msg.address)
		ctx.Sender().Request(ProcessResp{}, ctx.Self())
	default:
		log.Error("[P2plActorServer] receive unknown message type!")
	}
}

func (this *P2plActorServer) SetLocalPID(pid *actor.PID) {
	this.localPID = pid
}

func (this *P2plActorServer) GetLocalPID() *actor.PID {
	return this.localPID
}

//------------------------------------------------------------------------------------

type ConnectReq struct {
	address string
}

type CloseReq struct {
	address string
}

type SendReq struct {
	address string
	data    proto.Message
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
		log.Error("[P2pConnect] error: ", err)
		return err
	}
	return nil
}

func P2pSend(address string, data proto.Message) error {
	chReq := &SendReq{address, data}
	future := P2pServerPid.RequestFuture(chReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[P2pConnect] error: ", err)
		return err
	}
	return nil
}