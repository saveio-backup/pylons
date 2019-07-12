package server

import (
	"errors"
	"time"

	"github.com/gogo/protobuf/proto"
	p2p_act "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

type ChannelInfo struct {
	ChannelId     uint32
	Balance       uint64
	BalanceFormat string
	Address       string
	HostAddr      string
	TokenAddr     string
}

type ChannelInfosResp struct {
	Balance       uint64
	BalanceFormat string
	Channels      []*ChannelInfo
}

type VersionReq struct{}

type VersionResp struct {
	version string
}

type SetHostAddrReq struct {
	addr    common.Address
	netAddr string
}

type SetHostAddrResp struct {
}

type GetHostAddrReq struct {
	addr common.Address
}

type GetHostAddrResp struct {
	addr    common.Address
	netAddr string
}

type GetHostAddrCallbackType func(address common.Address) (string, error)

type SetGetHostAddrCallbackReq struct {
	GetHostAddrCallback GetHostAddrCallbackType
}

type SetGetHostAddrCallbackResp struct {
	Err error
}

type OpenChannelRet struct {
	ChannelID common.ChannelID
	Done      chan bool
	Err       error
}

type OpenChannelReq struct {
	TokenAddress common.TokenAddress
	Target       common.Address
	Ret          *OpenChannelRet
}

type SetTotalChannelDepositRet struct {
	Done chan bool
	Err  error
}

type SetTotalChannelDepositReq struct {
	TokenAddress common.TokenAddress
	PartnerAdder common.Address
	TotalDeposit common.TokenAmount
	Ret          *SetTotalChannelDepositRet
}

type DirectTransferRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type DirectTransferReq struct {
	Target     common.Address
	Amount     common.TokenAmount
	Identifier common.PaymentID
	Ret        *DirectTransferRet
}

type MediaTransferRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type MediaTransferReq struct {
	RegisterAddress common.PaymentNetworkID
	TokenAddress    common.TokenAddress
	Target          common.Address
	Amount          common.TokenAmount
	Identifier      common.PaymentID
	Ret             *MediaTransferRet
}

type CanTransferReq struct {
	target common.Address
	amount common.TokenAmount
}

type CanTransferResp struct {
	ret bool
}

type WithdrawRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type WithdrawReq struct {
	TokenAddress   common.TokenAddress
	PartnerAddress common.Address
	TotalWithdraw  common.TokenAmount
	Ret            *WithdrawRet
}

type ChannelReachableReq struct {
	target common.Address
}

type ChannelReachableResp struct {
	ret bool
	err error
}

type CloseChannelReq struct {
	target common.Address
}

type CloseChannelResp struct {
	ret bool
	err error
}

type GetTotalDepositBalanceReq struct {
	target common.Address
}

type GetTotalDepositBalanceResp struct {
	ret uint64
	err error
}

type GetTotalWithdrawReq struct {
	target common.Address
}

type GetTotalWithdrawResp struct {
	ret uint64
	err error
}

type GetAvaliableBalanceReq struct {
	partnerAddress common.Address
}

type GetAvaliableBalanceResp struct {
	ret uint64
	err error
}

type GetCurrentBalanceReq struct {
	partnerAddress common.Address
}

type GetCurrentBalanceResp struct {
	ret uint64
	err error
}

type CooperativeSettleReq struct {
	partnerAddress common.Address
}

type CooperativeSettleResp struct {
	err error
}

type GetUnitPricesReq struct {
	asset int32
}

type GetUnitPricesResp struct {
	ret uint64
	err error
}

type SetUnitPricesReq struct {
	asset int32
	price uint64
}

type SetUnitPricesResp struct {
	ret bool
}

type GetAllChannelsReq struct {
}

type GetAllChannelsResp struct {
	ret *ChannelInfosResp
}

type NodeStateChangeReq struct {
	Address string
	State   string
}

type NodeStateChangeResp struct{}

type HealthyCheckNodeReq struct {
	Address common.Address
}

type HealthyCheckNodeResp struct{}

type ProcessResp struct{}

type RegisterReceiveNotificationReq struct{}

type RegisterRecieveNotificationResp struct {
	notificationChannel chan *transfer.EventPaymentReceivedSuccess
}

type LastFilterBlockHeightReq struct{}
type LastFilterBlockHeightResp struct {
	Height uint32
}

func GetVersion() (string, error) {
	chReq := &VersionReq{}
	future := ChannelServerPid.RequestFuture(chReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetVersion] error: ", err)
		return "", err
	} else {
		version := ret.(string)
		return version, nil
	}
}

func SetHostAddr(addr common.Address, netAddr string) error {
	setHostAddrReq := &SetHostAddrReq{addr, netAddr}
	future := ChannelServerPid.RequestFuture(setHostAddrReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[SetHostAddr] error: ", err)
		return err
	}
	return nil
}

func GetHostAddr(addr common.Address) (string, error) {
	getHostAddrReq := &GetHostAddrReq{addr}
	future := ChannelServerPid.RequestFuture(getHostAddrReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetHostAddr] error: ", err)
		return "", err
	} else {
		hostAddr := ret.(GetHostAddrResp)
		return hostAddr.netAddr, nil
	}
}

func SetGetHostAddrCallback(getHostAddrCallback GetHostAddrCallbackType) error {
	setGetHostAddrReq := &SetGetHostAddrCallbackReq{getHostAddrCallback}
	future := ChannelServerPid.RequestFuture(setGetHostAddrReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[SetGetHostAddrCallback] error: ", err)
		return err
	}
	return nil

}

func OpenChannel(tokenAddress common.TokenAddress, target common.Address) (common.ChannelID, error) {
	ret := &OpenChannelRet{
		ChannelID: 0,
		Done:      make(chan bool, 1),
		Err:       nil,
	}
	openChannelReq := &OpenChannelReq{TokenAddress: tokenAddress, Target: target, Ret: ret}
	ChannelServerPid.Tell(openChannelReq)
	<-openChannelReq.Ret.Done
	return openChannelReq.Ret.ChannelID, openChannelReq.Ret.Err
}

func SetTotalChannelDeposit(tokenAddress common.TokenAddress, partnerAddress common.Address,
	totalDeposit common.TokenAmount) error {
	ret := &SetTotalChannelDepositRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}
	setTotalChannelDepositReq := &SetTotalChannelDepositReq{
		TokenAddress: tokenAddress,
		PartnerAdder: partnerAddress,
		TotalDeposit: totalDeposit,
		Ret:          ret,
	}
	ChannelServerPid.Tell(setTotalChannelDepositReq)
	<-setTotalChannelDepositReq.Ret.Done
	return setTotalChannelDepositReq.Ret.Err
}

func DirectTransferAsync(amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) (bool, error) {
	ret := &DirectTransferRet{
		Success: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	directTransferReq := &DirectTransferReq{
		Target:     target,
		Amount:     amount,
		Identifier: identifier,
		Ret:        ret,
	}
	ChannelServerPid.Tell(directTransferReq)
	<-directTransferReq.Ret.Done
	return directTransferReq.Ret.Success, directTransferReq.Ret.Err
}

func MediaTransfer(registryAddress common.PaymentNetworkID, tokenAddress common.TokenAddress,
	amount common.TokenAmount, target common.Address, identifier common.PaymentID) (bool, error) {
	ret := &MediaTransferRet{
		Success: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	mediaTransferReq := &MediaTransferReq{
		RegisterAddress: registryAddress,
		TokenAddress:    tokenAddress,
		Target:          target,
		Amount:          amount,
		Identifier:      identifier,
		Ret:             ret,
	}
	ChannelServerPid.Tell(mediaTransferReq)
	<-mediaTransferReq.Ret.Done
	return mediaTransferReq.Ret.Success, mediaTransferReq.Ret.Err
}

func CanTransfer(target common.Address, amount common.TokenAmount) (bool, error) {
	canTransferReq := &CanTransferReq{target, amount}
	future := ChannelServerPid.RequestFuture(canTransferReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[CanTransfer] error: ", err)
		return false, err
	} else {
		canTransferResp := ret.(CanTransferResp)
		return canTransferResp.ret, nil
	}
}

func WithDraw(tokenAddress common.TokenAddress, partnerAddress common.Address,
	totalWithdraw common.TokenAmount) (bool, error) {
	ret := &WithdrawRet{
		Success: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	withdrawReq := &WithdrawReq{
		TokenAddress:   tokenAddress,
		PartnerAddress: partnerAddress,
		TotalWithdraw:  totalWithdraw,
		Ret:            ret,
	}
	ChannelServerPid.Tell(withdrawReq)
	<-withdrawReq.Ret.Done
	return withdrawReq.Ret.Success, withdrawReq.Ret.Err
}

func ChannelReachable(target common.Address) (bool, error) {
	reachableReq := &ChannelReachableReq{target}
	future := ChannelServerPid.RequestFuture(reachableReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[ChannelReachable] error: ", err)
		return false, err
	} else {
		channelReachableResp := ret.(ChannelReachableResp)
		return channelReachableResp.ret, channelReachableResp.err
	}
}

func CloseChannel(target common.Address) (bool, error) {
	closeChannelReq := &CloseChannelReq{target}
	future := ChannelServerPid.RequestFuture(closeChannelReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[CloseChannel] error: ", err)
		return false, err
	} else {
		closeChannelResp := ret.(CloseChannelResp)
		return closeChannelResp.ret, closeChannelResp.err
	}
}

func GetTotalDepositBalance(target common.Address) (uint64, error) {
	getTotalDepositBalanceReq := &GetTotalDepositBalanceReq{target}
	future := ChannelServerPid.RequestFuture(getTotalDepositBalanceReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetTotalDepositBalance] error: ", err)
		return 0, err
	} else {
		getTotalDepositBalanceResp := ret.(GetTotalDepositBalanceResp)
		return getTotalDepositBalanceResp.ret, getTotalDepositBalanceResp.err
	}
}

func GetTotalWithdraw(target common.Address) (uint64, error) {
	getTotalWithdrawReq := &GetTotalWithdrawReq{target}
	future := ChannelServerPid.RequestFuture(getTotalWithdrawReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetTotalDepositBalance] error: ", err)
		return 0, err
	} else {
		getTotalWithdrawResp := ret.(GetTotalWithdrawResp)
		return getTotalWithdrawResp.ret, getTotalWithdrawResp.err
	}
}

func GetAvaliableBalance(partnerAddress common.Address) (uint64, error) {
	getAvaliableBalanceReq := &GetAvaliableBalanceReq{partnerAddress}
	future := ChannelServerPid.RequestFuture(getAvaliableBalanceReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetAvaliableBalance] error: ", err)
		return 0, err
	} else {
		getAvaliableBalanceResp := ret.(GetAvaliableBalanceResp)
		return getAvaliableBalanceResp.ret, getAvaliableBalanceResp.err
	}
}

func GetCurrentBalance(partnerAddress common.Address) (uint64, error) {
	getCurrentBalanceReq := &GetCurrentBalanceReq{partnerAddress}
	future := ChannelServerPid.RequestFuture(getCurrentBalanceReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetCurrentBalance] error: ", err)
		return 0, err
	} else {
		getCurrentBalanceResp := ret.(GetCurrentBalanceResp)
		return getCurrentBalanceResp.ret, getCurrentBalanceResp.err
	}
}

func CooperativeSettle(partnerAddress common.Address) error {
	cooperativeSettleReq := &CooperativeSettleReq{partnerAddress}
	future := ChannelServerPid.RequestFuture(cooperativeSettleReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[CooperativeSettle] error: ", err)
		return err
	} else {
		cooperativeSettleResp := ret.(CooperativeSettleResp)
		return cooperativeSettleResp.err
	}
}

func GetUnitPrices(asset int32) (uint64, error) {
	getUnitPricesReq := &GetUnitPricesReq{asset}
	future := ChannelServerPid.RequestFuture(getUnitPricesReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetUnitPrices] error: ", err)
		return 0, err
	} else {
		getUnitPricesResp := ret.(GetUnitPricesResp)
		return getUnitPricesResp.ret, getUnitPricesResp.err
	}
}

func SetUnitPrices(asset int32, price uint64) error {
	setUnitPricesReq := &SetUnitPricesReq{asset, price}
	future := ChannelServerPid.RequestFuture(setUnitPricesReq, constants.REQ_TIMEOUT*time.Second)
	if _, err := future.Result(); err != nil {
		log.Error("[SetUnitPrices] error: ", err)
		return err
	} else {
		return nil
	}
}

func GetAllChannels() *ChannelInfosResp {
	getAllChannelsReq := &GetAllChannelsReq{}
	future := ChannelServerPid.RequestFuture(getAllChannelsReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[GetAllChannels] error: ", err)
		return nil
	} else {
		getAllChannelsResp := ret.(GetAllChannelsResp)
		return getAllChannelsResp.ret
	}
}

func OnBusinessMessage(message proto.Message, from string) error {
	future := ChannelServerPid.RequestFuture(&p2p_act.RecvMsg{From: from, Message: message},
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

func HealthyCheckNodeState(address common.Address) error {
	ChannelServerPid.Tell(&HealthyCheckNodeReq{Address: address})
	return nil
}

func RegisterReceiveNotification() (chan *transfer.EventPaymentReceivedSuccess, error) {
	registerReceiveNotificationReq := &RegisterReceiveNotificationReq{}
	future := ChannelServerPid.RequestFuture(registerReceiveNotificationReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[RegisterReceiveNotification] error: ", err)
		return nil, err
	} else {
		registerRecieveNotificationResp := ret.(RegisterRecieveNotificationResp)
		return registerRecieveNotificationResp.notificationChannel, nil
	}
}

func GetLastFilterBlockHeight() (uint32, error) {
	req := &LastFilterBlockHeightReq{}
	future := ChannelServerPid.RequestFuture(req, constants.REQ_TIMEOUT*time.Second)
	ret, err := future.Result()
	if err != nil {
		return 0, err
	}
	result, ok := ret.(LastFilterBlockHeightResp)
	if !ok {
		return 0, errors.New("invalid resp")
	}
	return result.Height, nil
}
