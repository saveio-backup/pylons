package server

import (
	"github.com/ontio/ontology-eventbus/actor"
	oc "github.com/saveio/pylons"
	p2p_act "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
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
	channelActorServer.props = actor.FromProducer(func() actor.Actor { return channelActorServer })
	channelActorServer.localPID, err = actor.SpawnNamed(channelActorServer.props, "channel_server")
	if err != nil {
		return nil, err
	}
	if channelActorServer.chSrv, err = oc.NewChannelService(config, account); err != nil {
		return nil, err
	}
	channelActorServer.chSrv.Service.InitDB()
	ChannelServerPid = channelActorServer.localPID
	return channelActorServer, nil
}

func (this *ChannelActorServer) Start() error {
	var err error
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
		if err != nil && netAddr != "" {
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
		ctx.Sender().Request(DirectTransferAsyncResp{ret, err}, ctx.Self())
	case *MediaTransferReq:
		var ret chan bool
		ret, err := this.chSrv.Service.MediaTransfer(msg.registryAddress, msg.tokenAddress, msg.amount,
			msg.target, msg.identifier)
		ctx.Sender().Request(MediaTransferResp{ret, err}, ctx.Self())
	case *CanTransferReq:
		ret := this.chSrv.Service.CanTransfer(msg.target, msg.amount)
		ctx.Sender().Request(CanTransferResp{ret}, ctx.Self())
	case *WithdrawReq:
		var ret chan bool
		ret, err := this.chSrv.Service.Withdraw(msg.tokenAddress, msg.partnerAddress, msg.totalWithdraw)
		ctx.Sender().Request(WithdrawResp{ret, err}, ctx.Self())
	case *ChannelReachableReq:
		if transfer.NetworkReachable == this.chSrv.Service.GetNodeNetworkState(msg.target) {
			ctx.Sender().Request(ChannelReachableResp{true, nil}, ctx.Self())
		} else {
			ctx.Sender().Request(ChannelReachableResp{false, nil}, ctx.Self())
		}
	case *CloseChannelReq:
		tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
		this.chSrv.Service.ChannelClose(tokenAddress, msg.target, 3)
		ctx.Sender().Request(CloseChannelResp{true, nil}, ctx.Self())
	case *NodeStateChangeReq:
		this.chSrv.Service.Transport.SetNodeNetworkState(msg.Address, msg.State)
		ctx.Sender().Request(NodeStateChangeResp{}, ctx.Self())
	case *HealthyCheckNodeReq:
		this.chSrv.Service.Transport.StartHealthCheck(msg.Address)
	//TBD
	case *GetTotalDepositBalanceReq:
		ctx.Sender().Request(GetTotalDepositBalanceResp{0, nil}, ctx.Self())
	case *GetAvaliableBalanceReq:
		ctx.Sender().Request(GetAvaliableBalanceResp{0, nil}, ctx.Self())
	case *GetCurrentBalanceReq:
		ctx.Sender().Request(GetCurrentBalanceResp{0, nil}, ctx.Self())
	case *CooperativeSettleReq:
		ctx.Sender().Request(CooperativeSettleResp{nil}, ctx.Self())
	case *GetUnitPricesReq:
		ctx.Sender().Request(GetUnitPricesResp{0, nil}, ctx.Self())
	case *SetUnitPricesReq:
		ctx.Sender().Request(SetUnitPricesResp{true}, ctx.Self())
	case *GetAllChannelsReq:
		ctx.Sender().Request(GetAllChannelsResp{nil}, ctx.Self())
	case *p2p_act.RecvMsg:
		this.chSrv.Service.Transport.Receive(msg.Message, msg.From)
		ctx.Sender().Request(p2p_act.P2pResp{nil}, ctx.Self())
	default:
		log.Errorf("[ChannelActorServer] receive unknown message type:%+v", msg)
	}
}

func (this *ChannelActorServer) SetLocalPID(pid *actor.PID) {
	this.localPID = pid
}

func (this *ChannelActorServer) GetLocalPID() *actor.PID {
	return this.localPID
}

func (this *ChannelActorServer) GetChannelService() *oc.Channel {
	return this.chSrv
}
