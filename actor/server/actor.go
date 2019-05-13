package server

import (
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

type OpenChannelReq struct {
	tokenAddress common.TokenAddress
	target       common.Address
}

type OpenChannelResp struct {
	channelID common.ChannelID
}

type SetTotalChannelDepositReq struct {
	tokenAddress   common.TokenAddress
	partnerAddress common.Address
	totalDeposit   common.TokenAmount
}

type SetTotalChannelDepositResp struct {
	err error
}

type DirectTransferAsyncReq struct {
	amount     common.TokenAmount
	target     common.Address
	identifier common.PaymentID
}

type DirectTransferAsyncResp struct {
	ret chan bool
	err error
}

type MediaTransferReq struct {
	registryAddress common.PaymentNetworkID
	tokenAddress    common.TokenAddress
	amount          common.TokenAmount
	target          common.Address
	identifier      common.PaymentID
}

type MediaTransferResp struct {
	ret chan bool
	err error
}

type CanTransferReq struct {
	target common.Address
	amount common.TokenAmount
}

type CanTransferResp struct {
	ret bool
}

type WithdrawReq struct {
	tokenAddress   common.TokenAddress
	partnerAddress common.Address
	totalWithdraw  common.TokenAmount
}

type WithdrawResp struct {
	ret chan bool
	err error
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

type RegisterRecieveNotificationReq struct{}

type RegisterRecieveNotificationResp struct {
	notificationChannel chan *transfer.EventPaymentReceivedSuccess
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

func OpenChannel(tokenAddress common.TokenAddress, target common.Address) (common.ChannelID, error) {
	openChannelReq := &OpenChannelReq{tokenAddress, target}
	future := ChannelServerPid.RequestFuture(openChannelReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[OpenChannel] error: ", err)
		return 0, err
	} else {
		openChannelResp := ret.(OpenChannelResp)
		return openChannelResp.channelID, nil
	}
}

func SetTotalChannelDeposit(tokenAddress common.TokenAddress, partnerAddress common.Address,
	totalDeposit common.TokenAmount) error {
	setTotalChannelDepositReq := &SetTotalChannelDepositReq{tokenAddress,
		partnerAddress, totalDeposit}
	future := ChannelServerPid.RequestFuture(setTotalChannelDepositReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[SetTotalChannelDeposit] error: ", err)
		return err
	} else {
		setTotalChannelDepositResp := ret.(SetTotalChannelDepositResp)
		return setTotalChannelDepositResp.err
	}
}

func DirectTransferAsync(amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) (bool, error) {
	directTransferAsyncReq := &DirectTransferAsyncReq{amount, target, identifier}
	future := ChannelServerPid.RequestFuture(directTransferAsyncReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[DirectTransferAsync] error: ", err)
		return false, err
	} else {
		directTransferAsyncResp := ret.(DirectTransferAsyncResp)
		d := <-directTransferAsyncResp.ret
		return d, directTransferAsyncResp.err
	}
}

func MediaTransfer(registryAddress common.PaymentNetworkID, tokenAddress common.TokenAddress,
	amount common.TokenAmount, target common.Address, identifier common.PaymentID) (bool, error) {
	mediaTransferReq := &MediaTransferReq{registryAddress, tokenAddress,
		amount, target, identifier}
	future := ChannelServerPid.RequestFuture(mediaTransferReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[MediaTransfer] error: ", err)
		return false, err
	} else {
		mediaTransferResp := ret.(MediaTransferResp)
		d := <-mediaTransferResp.ret
		return d, mediaTransferResp.err
	}
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
	withDrawReq := &WithdrawReq{tokenAddress, partnerAddress, totalWithdraw}
	future := ChannelServerPid.RequestFuture(withDrawReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[WithDraw] error: ", err)
		return false, err
	} else {
		withDrawResp := ret.(WithdrawResp)
		d := <-withDrawResp.ret
		return d, withDrawResp.err
	}
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
	registerRecieveNotificationReq := &RegisterRecieveNotificationReq{}
	future := ChannelServerPid.RequestFuture(registerRecieveNotificationReq, constants.REQ_TIMEOUT*time.Second)
	if ret, err := future.Result(); err != nil {
		log.Error("[RegisterReceiveNotification] error: ", err)
		return nil, err
	} else {
		registerRecieveNotificationResp := ret.(RegisterRecieveNotificationResp)
		return registerRecieveNotificationResp.notificationChannel, nil
	}
}
