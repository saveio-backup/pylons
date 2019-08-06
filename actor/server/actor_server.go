package server

import (
	"fmt"

	"github.com/ontio/ontology-eventbus/actor"
	"github.com/pkg/errors"
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
		go func() {
			msg.Ret.Version = this.chSrv.GetVersion()
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *SetHostAddrReq:
		go func() {
			this.chSrv.Service.SetHostAddr(msg.WalletAddr, msg.NetAddr)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *GetHostAddrReq:
		go func() {
			netAddr, err := this.chSrv.Service.GetHostAddr(msg.WalletAddr)
			if err != nil {
				msg.Ret.Err = err
			} else if netAddr == "" {
				msg.Ret.Err = fmt.Errorf("NetAddr is nil")
			} else {
				msg.Ret.Err = nil
			}
			msg.Ret.Done <- true
		}()
	case *SetGetHostAddrCallbackReq:
		go func() {
			this.chSrv.Service.SetHostAddrCallBack(msg.GetHostAddrCallback)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *OpenChannelReq:
		go func() {
			channelId := this.chSrv.Service.OpenChannel(msg.TokenAddress, msg.Target)
			if channelId > 100 {
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
			msg.Ret.Err = this.chSrv.Service.SetTotalChannelDeposit(msg.TokenAddress, msg.PartnerAdder,
				msg.TotalDeposit)
			msg.Ret.Done <- true
		}()
	case *DirectTransferReq:
		go func() {
			ret, err := this.chSrv.Service.DirectTransferAsync(msg.Amount, msg.Target, msg.Identifier)
			if err == nil {
				msg.Ret.Success = <-ret
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
				msg.Ret.Success = <-ret
				msg.Ret.Err = nil
			} else {
				msg.Ret.Success = false
				msg.Ret.Err = err
			}
			msg.Ret.Done <- true
		}()
	case *CanTransferReq:
		go func() {
			msg.Ret.Result = this.chSrv.Service.CanTransfer(msg.Target, msg.Amount)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *WithdrawReq:
		go func() {
			ret, err := this.chSrv.Service.Withdraw(msg.TokenAddress, msg.PartnerAddress, msg.TotalWithdraw)
			msg.Ret.Success = <-ret
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *ChannelReachableReq:
		go func() {
			result := this.chSrv.Service.GetNodeNetworkState(msg.Target)
			if result == transfer.NetworkReachable {
				msg.Ret.Result = true
				msg.Ret.Err = nil
			} else if result == transfer.NetworkUnreachable {
				msg.Ret.Result = false
				msg.Ret.Err = fmt.Errorf("Node is unreacheable.")
			} else if result == transfer.NetworkUnknown {
				msg.Ret.Result = false
				msg.Ret.Err = fmt.Errorf("Node is unknown.")
			}
			msg.Ret.Done <- true
		}()
	case *CloseChannelReq:
		go func() {
			tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
			this.chSrv.Service.ChannelClose(tokenAddress, msg.Target, 3)
			msg.Ret.Result = true
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *HealthyCheckNodeReq:
		go func() {
			this.chSrv.Service.Transport.StartHealthCheck(msg.Address)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *GetTotalDepositBalanceReq:
		go func() {
			amount, err := this.chSrv.Service.GetTotalDepositBalance(msg.Target)
			msg.Ret.Ret = uint64(amount)
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *GetTotalWithdrawReq:
		go func() {
			amount, err := this.chSrv.Service.GetTotalWithdraw(msg.Target)
			msg.Ret.Ret = uint64(amount)
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *GetAvailableBalanceReq:
		go func() {
			amount, err := this.chSrv.Service.GetAvailableBalance(msg.PartnerAddress)
			msg.Ret.Ret = uint64(amount)
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *GetCurrentBalanceReq:
		go func() {
			amount, err := this.chSrv.Service.GetCurrentBalance(msg.PartnerAddress)
			msg.Ret.Ret = uint64(amount)
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *CooperativeSettleReq:
		go func() {
			tokenAddr := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
			msg.Ret.Err = this.chSrv.Service.ChannelCooperativeSettle(tokenAddr, msg.PartnerAddress)
			msg.Ret.Done <- true
		}()
	case *GetUnitPricesReq:
		msg.Ret.Ret = 0
		msg.Ret.Err = nil
		msg.Ret.Done <- true
	case *SetUnitPricesReq:
		msg.Ret.Ret = true
		msg.Ret.Err = nil
		msg.Ret.Done <- true
	case *GetAllChannelsReq:
		go func() {
			channelsInfo := this.chSrv.Service.GetAllChannelInfo()
			msg.Ret.Ret = getChannelsInfoRespFromChannelsInfo(channelsInfo)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *RegisterReceiveNotificationReq:
		go func() {
			notificationChannel := make(chan *transfer.EventPaymentReceivedSuccess)
			this.chSrv.RegisterReceiveNotification(notificationChannel)
			msg.Ret.NotificationChannel = notificationChannel
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()

	case *p2p_act.RecvMsg:
		go func() {
			this.chSrv.Service.Transport.Receive(msg.Message, msg.From)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *LastFilterBlockHeightReq:
		go func() {
			height := this.chSrv.Service.GetLastFilterBlock()
			msg.Ret.Height = uint32(height)
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *GetPaymentResultReq:
		go func() {
			paymentResult := this.chSrv.Service.GetPaymentResult(msg.Target, msg.Identifier)
			if paymentResult != nil {
				msg.Ret.Ret = &PaymentResultResp{
					Result: paymentResult.Result,
					Reason: paymentResult.Reason,
				}
			} else {
				msg.Ret.Ret = nil
			}
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
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
func getChannelsInfoRespFromChannelsInfo(channelInfos []*channelservice.ChannelInfo) *ChannelsInfoResp {
	resp := &ChannelsInfoResp{}
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
