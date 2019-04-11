package actor

import (
	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common/log"
	oc "github.com/oniio/oniChannel"
	"github.com/ontio/ontology-eventbus/actor"
)

var ChannelServerPid *actor.PID

type ChannelActorServer struct {
	props    *actor.Props
	localPID *actor.PID
	chSrv    *oc.Channel
}

func NewChannelActor(config *oc.ChannelConfig, account *account.Account) (*ChannelActorServer, error) {
	var err error
	channelActorServer := &ChannelActorServer{}
	if channelActorServer.chSrv, err = oc.NewChannelService(config, account); err != nil {
		return nil, err
	}
	return channelActorServer, nil
}

func (this *ChannelActorServer) Start() error {
	var err error
	this.props = actor.FromProducer(func() actor.Actor { return this })
	this.localPID, err = actor.SpawnNamed(this.props, "p2p_server")
	ChannelServerPid = this.localPID

	err = this.chSrv.StartService()
	if err != nil {
		log.Error("[ChannelActorServer] ChannelService Start error: ", err.Error())
	}
	return err
}

func (this *ChannelActorServer) Stop() {
	this.chSrv.Stop()
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

	case *VersionReq:
		version := this.chSrv.GetVersion()
		ctx.Sender().Request(VersionResp{version}, ctx.Self())
	case *SetHostAddrReq:
		this.chSrv.Service.SetHostAddr(msg.addr, msg.netAddr)
		ctx.Sender().Request(SetHostAddrResp{}, ctx.Self())
	case *GetHostAddrReq:
		netAddr, err := this.chSrv.Service.GetHostAddr(msg.addr)
		if err != nil && netAddr != ""{
			ctx.Sender().Request(GetHostAddrResp{msg.addr, netAddr}, ctx.Self())
		} else {
			ctx.Sender().Request(GetHostAddrResp{msg.addr, ""}, ctx.Self())
		}
	case *OpenChannelReq:
		chanId := this.chSrv.Service.OpenChannel(msg.tokenAddress, msg.target)
		ctx.Sender().Request(OpenChannelResp{chanId}, ctx.Self())
	case *SetTotalChannelDepositReq:
		err := this.chSrv.Service.SetTotalChannelDeposit(msg.tokenAddress, msg.partnerAddress, msg.totalDeposit)
		ctx.Sender().Request(SetTotalChannelDepositResp{err}, ctx.Self())
	case *DirectTransferAsyncReq:
		var ret chan bool
		ret, err := this.chSrv.Service.DirectTransferAsync(msg.amount, msg.target, msg.identifier)
		d := <- ret
		ctx.Sender().Request(DirectTransferAsyncResp{d, err}, ctx.Self())
	case *MediaTransferReq:
		var ret chan bool
		ret, err := this.chSrv.Service.MediaTransfer(msg.registryAddress, msg.tokenAddress, msg.amount,
			msg.target, msg.identifier)
		d := <- ret
		ctx.Sender().Request(MediaTransferResp{d, err}, ctx.Self())
	case *CanTransferReq:
		ret := this.chSrv.Service.CanTransfer(msg.target, msg.amount)
		ctx.Sender().Request(CanTransferResp{ret}, ctx.Self())
	case *WithdrawReq:
		var ret chan bool
		ret, err := this.chSrv.Service.Withdraw(msg.tokenAddress, msg.partnerAddress, msg.totalWithdraw)
		d := <- ret
		ctx.Sender().Request(WithdrawResp{d, err}, ctx.Self())
	default:
		log.Error("[P2plActorServer] receive unknown message type!")
	}
}

func (this *ChannelActorServer) SetLocalPID(pid *actor.PID) {
	this.localPID = pid
}

func (this *ChannelActorServer) GetLocalPID() *actor.PID {
	return this.localPID
}
