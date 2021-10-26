package server

import (
	"fmt"

	"github.com/ontio/ontology-eventbus/actor"
	oc "github.com/saveio/pylons"
	p2pAct "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/service"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/cmd/utils"
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
		ChannelServerPid = nil
		return nil, err
	}
	if channelActorServer.chSrv, err = oc.NewChannelService(config, account); err != nil {
		ChannelServerPid = nil
		return nil, err
	}
	ChannelServerPid = channelActorServer.localPID
	return channelActorServer, nil
}

func (this *ChannelActorServer) SyncBlockData() error {
	err := this.chSrv.Service.SyncBlockData()
	if err != nil {
		log.Error("[ChannelActorServer] SyncBlockData error: ", err.Error())
	}
	return err
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
	case *StartPylonsReq:
		go func() {
			msg.Ret.Err = this.chSrv.StartPylons()
			msg.Ret.Done <- true
		}()
	case *StopPylonsReq:
		go func() {
			this.chSrv.StopPylons()
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *VersionReq:
		go func() {
			msg.Ret.Version = this.chSrv.GetVersion()
			msg.Ret.Err = nil
			msg.Ret.Done <- true
		}()
	case *OpenChannelReq:
		go func() {
			msg.Ret.ChannelID, msg.Ret.Err = this.chSrv.Service.OpenChannel(msg.TokenAddress, msg.Target)
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
				msg.Media, msg.Target, msg.Amount, msg.Identifier)
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
			msg.Ret.Err = this.chSrv.Service.Transport.StartHealthCheck(msg.Address)
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

	case *p2pAct.RecvMsg:
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
	case *GetFeeReq:
		go func() {
			fee, err := this.chSrv.Service.GetFee(msg.ChannelId)
			msg.Ret.Fee = fee
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *SetFeeReq:
		go func() {
			err := this.chSrv.Service.SetFee(msg.ChannelId, msg.Flat)
			msg.Ret.Err = err
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
func getChannelsInfoRespFromChannelsInfo(channelInfos []*service.ChannelInfo) *ChannelsInfoResp {
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
			TokenAddr:     (&tokenAddr).ToBase58(),
		}
		totalBalance += balance
		infos = append(infos, info)
	}

	resp.Balance = totalBalance
	resp.BalanceFormat = utils.FormatUsdt(totalBalance)
	resp.Channels = infos
	return resp
}
