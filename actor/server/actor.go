package server

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	p2p_act "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

const (
	defaultMinTimeOut = 1500
	defaultMaxTimeOut = 24000
)

func GetVersion() (string, error) {
	ret := &VersionRet{
		Version: "",
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	versionReq := &VersionReq{Ret: ret}
	ChannelServerPid.Tell(versionReq)
	if err := waitForCallDone(versionReq.Ret.Done, "SetHostAddr", defaultMinTimeOut); err != nil {
		return "", err
	} else {
		return versionReq.Ret.Version, versionReq.Ret.Err
	}
}

func GetHostAddr(walletAddr common.Address) (string, error) {
	ret := &GetHostAddrRet{
		WalletAddr: common.EmptyAddress,
		NetAddr:    "",
		Done:       make(chan bool, 1),
		Err:        nil,
	}
	getHostAddrReq := &GetHostAddrReq{WalletAddr: walletAddr, Ret: ret}
	ChannelServerPid.Tell(getHostAddrReq)

	if err := waitForCallDone(getHostAddrReq.Ret.Done, "GetHostAddr", defaultMinTimeOut); err != nil {
		return "", err
	} else {
		return getHostAddrReq.Ret.NetAddr, getHostAddrReq.Ret.Err
	}
}

func SetGetHostAddrCallback(getHostAddrCallback GetHostAddrCallbackType) error {
	ret := &SetGetHostAddrCallbackRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}

	setGetHostAddrCallbackReq := &SetGetHostAddrCallbackReq{GetHostAddrCallback: getHostAddrCallback, Ret: ret}
	ChannelServerPid.Tell(setGetHostAddrCallbackReq)

	if err := waitForCallDone(setGetHostAddrCallbackReq.Ret.Done, "SetGetHostAddrCallback", defaultMinTimeOut); err != nil {
		return err
	} else {
		return setGetHostAddrCallbackReq.Ret.Err
	}
}

func OpenChannel(tokenAddress common.TokenAddress, target common.Address) (common.ChannelID, error) {
	ret := &OpenChannelRet{
		ChannelID: 0,
		Done:      make(chan bool, 1),
		Err:       nil,
	}
	openChannelReq := &OpenChannelReq{TokenAddress: tokenAddress, Target: target, Ret: ret}
	ChannelServerPid.Tell(openChannelReq)

	if err := waitForCallDone(openChannelReq.Ret.Done, "OpenChannel", defaultMaxTimeOut); err != nil {
		return 0, err
	} else {
		return openChannelReq.Ret.ChannelID, openChannelReq.Ret.Err
	}
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

	if err := waitForCallDone(setTotalChannelDepositReq.Ret.Done, "SetTotalChannelDeposit", defaultMaxTimeOut); err != nil {
		return err
	} else {
		return setTotalChannelDepositReq.Ret.Err
	}
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

	if err := waitForCallDone(directTransferReq.Ret.Done, "DirectTransferAsync", defaultMaxTimeOut); err != nil {
		return false, err
	} else {
		return directTransferReq.Ret.Success, directTransferReq.Ret.Err
	}
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

	if err := waitForCallDone(mediaTransferReq.Ret.Done, "MediaTransfer", defaultMaxTimeOut); err != nil {
		return false, err
	} else {
		return mediaTransferReq.Ret.Success, mediaTransferReq.Ret.Err
	}
}

func CanTransfer(target common.Address, amount common.TokenAmount) (bool, error) {
	ret := &CanTransferRet{
		Result: false,
		Done:   make(chan bool, 1),
		Err:    nil,
	}
	canTransferReq := &CanTransferReq{Target: target, Amount: amount, Ret: ret}
	ChannelServerPid.Tell(canTransferReq)

	if err := waitForCallDone(canTransferReq.Ret.Done, "CanTransfer", defaultMinTimeOut); err != nil {
		return false, err
	} else {
		return canTransferReq.Ret.Result, canTransferReq.Ret.Err
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

	// withdraw time is handled as a event, no need actor timeout
	<-withdrawReq.Ret.Done
	close(withdrawReq.Ret.Done)
	return withdrawReq.Ret.Success, withdrawReq.Ret.Err
}

func ChannelReachable(target common.Address) (bool, error) {
	ret := &ChannelReachableRet{
		Result: false,
		Done:   make(chan bool, 1),
		Err:    nil,
	}
	reachableReq := &ChannelReachableReq{Target: target, Ret: ret}
	ChannelServerPid.Tell(reachableReq)

	if err := waitForCallDone(reachableReq.Ret.Done, "ChannelReachable", defaultMinTimeOut); err != nil {
		return false, err
	} else {
		return reachableReq.Ret.Result, reachableReq.Ret.Err
	}
}

func CloseChannel(target common.Address) (bool, error) {
	ret := &CloseChannelRet{
		Result: false,
		Done:   make(chan bool, 1),
		Err:    nil,
	}
	closeChannelReq := &CloseChannelReq{Target: target, Ret: ret}
	ChannelServerPid.Tell(closeChannelReq)

	if err := waitForCallDone(closeChannelReq.Ret.Done, "CloseChannel", defaultMaxTimeOut); err != nil {
		return false, err
	} else {
		return closeChannelReq.Ret.Result, closeChannelReq.Ret.Err
	}
}

func GetTotalDepositBalance(target common.Address) (uint64, error) {
	ret := &GetTotalDepositBalanceRet{
		Ret:  0,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getTotalDepositBalanceReq := &GetTotalDepositBalanceReq{Target: target, Ret: ret}
	ChannelServerPid.Tell(getTotalDepositBalanceReq)

	if err := waitForCallDone(getTotalDepositBalanceReq.Ret.Done, "GetTotalDepositBalance", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return getTotalDepositBalanceReq.Ret.Ret, getTotalDepositBalanceReq.Ret.Err
	}
}

func GetTotalWithdraw(target common.Address) (uint64, error) {
	ret := &GetTotalWithdrawRet{
		Ret:  0,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getTotalWithdrawReq := &GetTotalWithdrawReq{Target: target, Ret: ret}
	ChannelServerPid.Tell(getTotalWithdrawReq)

	if err := waitForCallDone(getTotalWithdrawReq.Ret.Done, "GetTotalWithdraw", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return getTotalWithdrawReq.Ret.Ret, getTotalWithdrawReq.Ret.Err
	}
}

func GetAvailableBalance(partnerAddr common.Address) (uint64, error) {
	ret := &GetAvailableBalanceRet{
		Ret:  0,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getAvailableBalanceReq := &GetAvailableBalanceReq{PartnerAddress: partnerAddr, Ret: ret}
	ChannelServerPid.Tell(getAvailableBalanceReq)

	if err := waitForCallDone(getAvailableBalanceReq.Ret.Done, "GetAvailableBalance", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return getAvailableBalanceReq.Ret.Ret, getAvailableBalanceReq.Ret.Err
	}
}

func GetCurrentBalance(partnerAddress common.Address) (uint64, error) {
	ret := &GetCurrentBalanceRet{
		Ret:  0,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getCurrentBalanceReq := &GetCurrentBalanceReq{PartnerAddress: partnerAddress, Ret: ret}
	ChannelServerPid.Tell(getCurrentBalanceReq)

	if err := waitForCallDone(getCurrentBalanceReq.Ret.Done, "GetCurrentBalance", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return getCurrentBalanceReq.Ret.Ret, getCurrentBalanceReq.Ret.Err
	}
}

func CooperativeSettle(partnerAddress common.Address) error {
	ret := &CooperativeSettleRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}
	cooperativeSettleReq := &CooperativeSettleReq{PartnerAddress: partnerAddress, Ret: ret}
	ChannelServerPid.Tell(cooperativeSettleReq)

	if err := waitForCallDone(cooperativeSettleReq.Ret.Done, "CooperativeSettle", defaultMaxTimeOut); err != nil {
		return err
	} else {
		return cooperativeSettleReq.Ret.Err
	}
}

func GetUnitPrices(asset int32) (uint64, error) {
	ret := &GetUnitPricesRet{
		Ret:  0,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getUnitPricesReq := &GetUnitPricesReq{Asset: asset, Ret: ret}
	ChannelServerPid.Tell(getUnitPricesReq)

	if err := waitForCallDone(getUnitPricesReq.Ret.Done, "GetUnitPrices", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return getUnitPricesReq.Ret.Ret, getUnitPricesReq.Ret.Err
	}
}

func SetUnitPrices(asset int32, price uint64) error {
	ret := &SetUnitPricesRet{
		Ret:  false,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	setUnitPricesReq := &SetUnitPricesReq{Asset: asset, Price: price, Ret: ret}
	ChannelServerPid.Tell(setUnitPricesReq)

	if err := waitForCallDone(setUnitPricesReq.Ret.Done, "SetUnitPrices", defaultMinTimeOut); err != nil {
		return err
	} else {
		return setUnitPricesReq.Ret.Err
	}
}

func GetAllChannels() (*ChannelsInfoResp, error) {
	ret := &GetAllChannelsRet{
		Ret:  nil,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getAllChannelsReq := &GetAllChannelsReq{Ret: ret}
	ChannelServerPid.Tell(getAllChannelsReq)

	if err := waitForCallDone(getAllChannelsReq.Ret.Done, "GetAllChannels", defaultMinTimeOut); err != nil {
		return nil, err
	} else {
		return getAllChannelsReq.Ret.Ret, nil
	}
}

func OnBusinessMessage(message proto.Message, from string) error {
	ret := &p2p_act.RecvMsgRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}
	recvMsg := &p2p_act.RecvMsg{From: from, Message: message, Ret: ret}
	ChannelServerPid.Tell(recvMsg)

	if err := waitForCallDone(recvMsg.Ret.Done, "OnBusinessMessage", defaultMinTimeOut); err != nil {
		return err
	} else {
		return recvMsg.Ret.Err
	}
}

func HealthyCheckNodeState(address common.Address) error {
	ret := &HealthyCheckNodeRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}
	healthyCheckNodeReq := &HealthyCheckNodeReq{Address: address, Ret: ret}
	ChannelServerPid.Tell(healthyCheckNodeReq)

	if err := waitForCallDone(healthyCheckNodeReq.Ret.Done, "HealthyCheckNodeState", defaultMinTimeOut); err != nil {
		return err
	} else {
		return healthyCheckNodeReq.Ret.Err
	}
}

func RegisterReceiveNotification() (chan *transfer.EventPaymentReceivedSuccess, error) {
	ret := &RegisterReceiveNotificationRet{
		NotificationChannel: nil,
		Done:                make(chan bool, 1),
		Err:                 nil,
	}
	registerReceiveNotificationReq := &RegisterReceiveNotificationReq{Ret: ret}
	ChannelServerPid.Tell(registerReceiveNotificationReq)

	if err := waitForCallDone(registerReceiveNotificationReq.Ret.Done, "RegisterReceiveNotification", defaultMinTimeOut); err != nil {
		return nil, err
	} else {
		return registerReceiveNotificationReq.Ret.NotificationChannel, registerReceiveNotificationReq.Ret.Err
	}
}

func GetLastFilterBlockHeight() (uint32, error) {
	ret := &LastFilterBlockHeightRet{
		Height: 0,
		Done:   make(chan bool, 1),
		Err:    nil,
	}
	lastFilterBlockHeightReq := &LastFilterBlockHeightReq{Ret: ret}
	ChannelServerPid.Tell(lastFilterBlockHeightReq)

	if err := waitForCallDone(lastFilterBlockHeightReq.Ret.Done, "GetLastFilterBlockHeight", defaultMinTimeOut); err != nil {
		return 0, err
	} else {
		return lastFilterBlockHeightReq.Ret.Height, lastFilterBlockHeightReq.Ret.Err
	}
}

func GetPaymentResult(target common.Address, identifier common.PaymentID) (*PaymentResultResp, error) {
	ret := &GetPaymentResultRet{
		Ret:  nil,
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getPaymentResultReq := &GetPaymentResultReq{target, identifier, ret}
	ChannelServerPid.Tell(getPaymentResultReq)

	if err := waitForCallDone(getPaymentResultReq.Ret.Done, "GetPaymentResult", defaultMinTimeOut); err != nil {
		return nil, err
	} else {
		return getPaymentResultReq.Ret.Ret, nil
	}
}

func waitForCallDone(c chan bool, funcName string, maxTimeOut int64) error {
	if maxTimeOut < defaultMinTimeOut {
		maxTimeOut = defaultMinTimeOut
	}
	select {
	case <-c:
		close(c)
		return nil
	case <-time.After(time.Duration(maxTimeOut) * time.Millisecond):
		return fmt.Errorf("function:[%s] timeout", funcName)
	}
}
