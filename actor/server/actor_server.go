package server

import (
	"github.com/ontio/ontology-eventbus/actor"
	oc "github.com/saveio/pylons"
	p2p_act "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/channelservice"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/cmd/utils"
	cmdutils "github.com/saveio/themis/cmd/utils"
	com "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/pkg/errors"
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
	case *SetGetHostAddrCallbackReq:
		this.chSrv.Service.SetHostAddrCallBack(msg.GetHostAddrCallback)
		ctx.Sender().Request(SetGetHostAddrCallbackResp{}, ctx.Self())
	case *GetHostAddrReq:
		netAddr, err := this.chSrv.Service.GetHostAddr(msg.addr)
		if err == nil && netAddr != "" {
			ctx.Sender().Request(GetHostAddrResp{msg.addr, netAddr}, ctx.Self())
		} else {
			ctx.Sender().Request(GetHostAddrResp{msg.addr, ""}, ctx.Self())
		}
	case *OpenChannelReq:
		go func() {
			channelId := this.chSrv.Service.OpenChannel(msg.TokenAddress, msg.Target)
			if channelId >100 {
				msg.Ret.ChannelID = channelId
				msg.Ret.Err = nil
			} else {
				msg.Ret.ChannelID = 0
				msg.Ret.Err = errors.New("OpenChannel failed.")
			}
			msg.Ret.Done <- true
		}()
	case *SetTotalChannelDepositReq:
		go func() {
			msg.Ret.Err = this.chSrv.Service.SetTotalChannelDeposit(msg.TokenAddress, msg.PartnerAdder, msg.TotalDeposit)
			msg.Ret.Done <- true
		}()
	case *DirectTransferReq:
		go func() {
			ret, err := this.chSrv.Service.DirectTransferAsync(msg.Amount, msg.Target, msg.Identifier)
			if err == nil {
				msg.Ret.Success = <- ret
				msg.Ret.Err = nil
			} else {
				msg.Ret.Success = false
				msg.Ret.Err = err
			}
			msg.Ret.Done <- true
		}()
	case *MediaTransferReq:
		go func() {
			ret, err := this.chSrv.Service.MediaTransfer(msg.RegisterAddress, msg.TokenAddress,
				msg.Amount, msg.Target, msg.Identifier)
			if err == nil {
				msg.Ret.Success = <- ret
				msg.Ret.Err = nil
			} else {
				msg.Ret.Success = false
				msg.Ret.Err = err
			}
			msg.Ret.Done <- true
		}()
	case *CanTransferReq:
		ret := this.chSrv.Service.CanTransfer(msg.target, msg.amount)
		ctx.Sender().Request(CanTransferResp{ret}, ctx.Self())
	case *WithdrawReq:
		go func() {
			ret, err := this.chSrv.Service.Withdraw(msg.TokenAddress, msg.PartnerAddress, msg.TotalWithdraw)
			if err == nil {
				msg.Ret.Success = <- ret
				msg.Ret.Err = nil
			} else {
				msg.Ret.Success = false
				msg.Ret.Err = err
			}
			msg.Ret.Done <- true
		}()
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
	case *GetTotalDepositBalanceReq:
		amount, err := this.chSrv.Service.GetTotalDepositBalance(msg.target)
		ctx.Sender().Request(GetTotalDepositBalanceResp{uint64(amount), err}, ctx.Self())
	case *GetTotalWithdrawReq:
		amount, err := this.chSrv.Service.GetTotalWithdraw(msg.target)
		ctx.Sender().Request(GetTotalWithdrawResp{uint64(amount), err}, ctx.Self())
	case *GetAvaliableBalanceReq:
		amount, err := this.chSrv.Service.GetAvaliableBalance(msg.partnerAddress)
		ctx.Sender().Request(GetAvaliableBalanceResp{uint64(amount), err}, ctx.Self())
	case *GetCurrentBalanceReq:
		amount, err := this.chSrv.Service.GetCurrentBalance(msg.partnerAddress)
		ctx.Sender().Request(GetCurrentBalanceResp{uint64(amount), err}, ctx.Self())
	case *CooperativeSettleReq:
		tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
		err := this.chSrv.Service.ChannelCooperativeSettle(tokenAddress, msg.partnerAddress)
		ctx.Sender().Request(CooperativeSettleResp{err}, ctx.Self())
	case *GetUnitPricesReq:
		ctx.Sender().Request(GetUnitPricesResp{0, nil}, ctx.Self())
	case *SetUnitPricesReq:
		ctx.Sender().Request(SetUnitPricesResp{true}, ctx.Self())
	case *GetAllChannelsReq:
		channelInfos := this.chSrv.Service.GetAllChannelInfo()
		ctx.Sender().Request(GetAllChannelsResp{getChannelInfosRespFromChannelInfos(channelInfos)}, ctx.Self())
	case *RegisterReceiveNotificationReq:
		notificationChannel := make(chan *transfer.EventPaymentReceivedSuccess)
		this.chSrv.RegisterReceiveNotification(notificationChannel)
		ctx.Sender().Request(RegisterRecieveNotificationResp{notificationChannel}, ctx.Self())
	case *p2p_act.RecvMsg:
		this.chSrv.Service.Transport.Receive(msg.Message, msg.From)
		ctx.Sender().Request(p2p_act.P2pResp{nil}, ctx.Self())
	case *LastFilterBlockHeightReq:
		height := this.chSrv.Service.GetLastFilterBlock()
		ctx.Sender().Request(LastFilterBlockHeightResp{Height: uint32(height)}, ctx.Self())
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
func getChannelInfosRespFromChannelInfos(channelInfos []*channelservice.ChannelInfo) *ChannelInfosResp {
	resp := &ChannelInfosResp{}
	totalBalance := uint64(0)
	infos := make([]*ChannelInfo, 0)
	for _, info := range channelInfos {
		balance := uint64(info.Balance)

		addr := com.Address(info.Address)
		tokenAddr := com.Address(info.TokenAddr)
		info := &ChannelInfo{
			ChannelId:     uint32(info.ChannelId),
			Address:       (&addr).ToBase58(),
			Balance:       uint64(info.Balance),
			BalanceFormat: utils.FormatUsdt(uint64(info.Balance)),
			HostAddr:      info.HostAddr,
			TokenAddr:     (&tokenAddr).ToBase58(),
		}
		totalBalance += balance
		infos = append(infos, info)
	}

	resp.Balance = totalBalance
	resp.BalanceFormat = cmdutils.FormatUsdt(totalBalance)
	resp.Channels = infos
	return resp
}
