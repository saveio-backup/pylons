package server

import (
	"github.com/gogo/protobuf/proto"
	p2p_act "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

func GetVersion() (string, error) {
	ret := &VersionRet{
		Version: "",
		Done: make(chan bool, 1),
		Err:  nil,
	}
	versionReq := &VersionReq{Ret:ret}
	ChannelServerPid.Tell(versionReq)
	<-versionReq.Ret.Done
	return versionReq.Ret.Version, versionReq.Ret.Err
}

func SetHostAddr(walletAddr common.Address, netAddr string) error {
	ret := &SetHostAddrRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}
	setHostAddrReq := &SetHostAddrReq{WalletAddr:walletAddr, NetAddr:netAddr, Ret:ret}
	ChannelServerPid.Tell(setHostAddrReq)
	<-setHostAddrReq.Ret.Done
	return setHostAddrReq.Ret.Err
}

func GetHostAddr(walletAddr common.Address) (string, error) {
	ret := &GetHostAddrRet{
		WalletAddr: common.EmptyAddress,
		NetAddr: "",
		Done: make(chan bool, 1),
		Err:  nil,
	}
	getHostAddrReq := &GetHostAddrReq{WalletAddr:walletAddr, Ret:ret}
	ChannelServerPid.Tell(getHostAddrReq)
	<-getHostAddrReq.Ret.Done
	return getHostAddrReq.Ret.NetAddr, getHostAddrReq.Ret.Err
}

func SetGetHostAddrCallback(getHostAddrCallback GetHostAddrCallbackType) error {
	ret := &SetGetHostAddrCallbackRet{
		Done: make(chan bool, 1),
		Err:  nil,
	}

	setGetHostAddrCallbackReq := &SetGetHostAddrCallbackReq{GetHostAddrCallback: getHostAddrCallback, Ret: ret}
	ChannelServerPid.Tell(setGetHostAddrCallbackReq)
	<-setGetHostAddrCallbackReq.Ret.Done
	return setGetHostAddrCallbackReq.Ret.Err
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
	ret := &CanTransferRet{
		Result: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	canTransferReq := &CanTransferReq{Target:target, Amount:amount, Ret:ret}
	ChannelServerPid.Tell(canTransferReq)
	<-canTransferReq.Ret.Done
	return canTransferReq.Ret.Result, canTransferReq.Ret.Err
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
	ret := &ChannelReachableRet{
		Result: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	reachableReq := &ChannelReachableReq{Target:target, Ret:ret}
	ChannelServerPid.Tell(reachableReq)
	<-reachableReq.Ret.Done
	return reachableReq.Ret.Result, reachableReq.Ret.Err
}

func CloseChannel(target common.Address) (bool, error) {
	ret := &CloseChannelRet{
		Result: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	closeChannelReq := &CloseChannelReq{Target:target, Ret:ret}
	ChannelServerPid.Tell(closeChannelReq)
	<-closeChannelReq.Ret.Done
	return closeChannelReq.Ret.Result, closeChannelReq.Ret.Err
}

func GetTotalDepositBalance(target common.Address) (uint64, error) {
	ret := &GetTotalDepositBalanceRet{
		Ret: 0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getTotalDepositBalanceReq := &GetTotalDepositBalanceReq{Target:target, Ret:ret}
	ChannelServerPid.Tell(getTotalDepositBalanceReq)
	<-getTotalDepositBalanceReq.Ret.Done
	return getTotalDepositBalanceReq.Ret.Ret, getTotalDepositBalanceReq.Ret.Err
}

func GetTotalWithdraw(target common.Address) (uint64, error) {
	ret := &GetTotalWithdrawRet{
		Ret: 0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getTotalWithdrawReq := &GetTotalWithdrawReq{Target:target, Ret:ret}
	ChannelServerPid.Tell(getTotalWithdrawReq)
	<-getTotalWithdrawReq.Ret.Done
	return getTotalWithdrawReq.Ret.Ret, getTotalWithdrawReq.Ret.Err
}

func GetAvailableBalance(partnerAddr common.Address) (uint64, error) {
	ret := &GetAvailableBalanceRet{
		Ret: 0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getAvailableBalanceReq := &GetAvailableBalanceReq{PartnerAddress:partnerAddr, Ret:ret}
	ChannelServerPid.Tell(getAvailableBalanceReq)
	<-getAvailableBalanceReq.Ret.Done
	return getAvailableBalanceReq.Ret.Ret, getAvailableBalanceReq.Ret.Err
}

func GetCurrentBalance(partnerAddress common.Address) (uint64, error) {
	ret := &GetCurrentBalanceRet{
		Ret: 0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getCurrentBalanceReq := &GetCurrentBalanceReq{PartnerAddress:partnerAddress, Ret:ret}
	ChannelServerPid.Tell(getCurrentBalanceReq)
	<-getCurrentBalanceReq.Ret.Done
	return getCurrentBalanceReq.Ret.Ret, getCurrentBalanceReq.Ret.Err
}

func CooperativeSettle(partnerAddress common.Address) error {
	ret := &CooperativeSettleRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	cooperativeSettleReq := &CooperativeSettleReq{PartnerAddress:partnerAddress, Ret:ret}
	ChannelServerPid.Tell(cooperativeSettleReq)
	<-cooperativeSettleReq.Ret.Done
	return cooperativeSettleReq.Ret.Err
}

func GetUnitPrices(asset int32) (uint64, error) {
	ret := &GetUnitPricesRet{
		Ret: 0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getUnitPricesReq := &GetUnitPricesReq{Asset:asset, Ret:ret}
	ChannelServerPid.Tell(getUnitPricesReq)
	<-getUnitPricesReq.Ret.Done
	return getUnitPricesReq.Ret.Ret, getUnitPricesReq.Ret.Err
}

func SetUnitPrices(asset int32, price uint64) error {
	ret := &SetUnitPricesRet{
		Ret: false,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	setUnitPricesReq := &SetUnitPricesReq{Asset:asset, Price: price, Ret:ret}
	ChannelServerPid.Tell(setUnitPricesReq)
	<-setUnitPricesReq.Ret.Done
	return setUnitPricesReq.Ret.Err
}

func GetAllChannels() *ChannelsInfoResp {
	ret := &GetAllChannelsRet{
		Ret:     nil,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	getAllChannelsReq := &GetAllChannelsReq{Ret:ret}
	ChannelServerPid.Tell(getAllChannelsReq)
	<-getAllChannelsReq.Ret.Done
	return getAllChannelsReq.Ret.Ret
}

func OnBusinessMessage(message proto.Message, from string) error {
	ret := &p2p_act.RecvMsgRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	recvMsg := &p2p_act.RecvMsg{From: from, Message: message, Ret: ret}
	ChannelServerPid.Tell(recvMsg)
	<-recvMsg.Ret.Done
	return recvMsg.Ret.Err
}

func SetNodeNetworkState(address string, state string) error {
	ret := &NodeStateChangeRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	nodeStateChangeReq := &NodeStateChangeReq{Address: address, State: state, Ret:ret}
	ChannelServerPid.Tell(nodeStateChangeReq)
	<-nodeStateChangeReq.Ret.Done
	return nodeStateChangeReq.Ret.Err
}

func HealthyCheckNodeState(address common.Address) error {
	ret := &HealthyCheckNodeRet{
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	healthyCheckNodeReq := &HealthyCheckNodeReq{Address: address, Ret:ret}
	ChannelServerPid.Tell(healthyCheckNodeReq)
	<-healthyCheckNodeReq.Ret.Done
	return healthyCheckNodeReq.Ret.Err
}

func RegisterReceiveNotification() (chan *transfer.EventPaymentReceivedSuccess, error) {
	ret := &RegisterReceiveNotificationRet{
		NotificationChannel:nil,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	registerReceiveNotificationReq := &RegisterReceiveNotificationReq{Ret:ret}
	ChannelServerPid.Tell(registerReceiveNotificationReq)
	<-registerReceiveNotificationReq.Ret.Done
	return registerReceiveNotificationReq.Ret.NotificationChannel, registerReceiveNotificationReq.Ret.Err
}

func GetLastFilterBlockHeight() (uint32, error) {
	ret := &LastFilterBlockHeightRet{
		Height:  0,
		Done:    make(chan bool, 1),
		Err:     nil,
	}
	lastFilterBlockHeightReq := &LastFilterBlockHeightReq{Ret:ret}
	ChannelServerPid.Tell(lastFilterBlockHeightReq)
	<-lastFilterBlockHeightReq.Ret.Done
	return lastFilterBlockHeightReq.Ret.Height, lastFilterBlockHeightReq.Ret.Err
}
