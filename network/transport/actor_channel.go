package transport

import (
	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChain/common/log"
	"github.com/ontio/ontology-eventbus/actor"
	"time"
	"github.com/oniio/oniChannel/common/constants"
)

var ChannelServerPid *actor.PID

type ChannelActorServer struct {
	props     *actor.Props
	localPID  *actor.PID
	Transport *Transport
}

func NewChannelActor(channelService ChannelServiceInterface) (*ChannelActorServer, error) {
	channelActorServer := &ChannelActorServer{}
	channelActorServer.Transport = NewTransport(channelService)
	return channelActorServer, nil
}

func (this *ChannelActorServer) Start() error {
	var err error
	this.props = actor.FromProducer(func() actor.Actor { return this })
	this.localPID, err = actor.SpawnNamed(this.props, "channel_server")
	if err != nil {
		return err
	}
	ChannelServerPid = this.localPID
	return nil
}

func (this *ChannelActorServer) Stop() error {
	return nil
}

func (this *ChannelActorServer) SetLocalPID(pid *actor.PID) {
	this.localPID = pid
}

func (this *ChannelActorServer) GetLocalPID() *actor.PID {
	return this.localPID
}

func (this *ChannelActorServer) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Restarting:
		log.Warn("[ChannelActorServer] Actor restarting")
	case *actor.Stopping:
		log.Warn("[ChannelActorServer] Actor stopping")
	case *actor.Stopped:
		log.Warn("[ChannelActorServer] Actor stopped")
	case *actor.Started:
		log.Debug("[ChannelActorServer] Actor started")
	case *actor.Restart:
		log.Warn("[ChannelActorServer] Actor restart")

	case *BusinessMessageReq:
		this.Transport.Receive(msg.Message, msg.From)
		ctx.Sender().Request(ProcessResp{}, ctx.Self())
	case *NodeStateChangeReq:
		this.Transport.SetNodeNetworkState(msg.Address, msg.State)
		ctx.Sender().Request(ProcessResp{}, ctx.Self())
	default:
		log.Error("[ChannelActorServer] receive unknown message type!")
	}
}

//--------------------------------------------------------------------------------------------

type BusinessMessageReq struct {
	From    string
	Message proto.Message
}

type NodeStateChangeReq struct {
	Address string
	State   string
}

type ProcessResp struct {}

func OnBusinessMessage(message proto.Message, from string) error {
	future := ChannelServerPid.RequestFuture(&BusinessMessageReq{From: from, Message: message},
		constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[OnBusinessMessage] error: ", err)
		return err
	}
	return nil
}

func SetNodeNetworkState(address string, state string) error {
	future := ChannelServerPid.RequestFuture(&NodeStateChangeReq{Address: address, State: state},
		constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[SetNodeNetworkState] error: ", err)
		return err
	}
	return nil
}
