package actor

import (
	"time"

	"github.com/saveio/themis/common/log"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/common"
)

type VersionReq struct{}

type VersionResp struct {
	version string
}

type SetHostAddrReq struct {
	addr common.Address
	netAddr string
}

type SetHostAddrResp struct {
}

type GetHostAddrReq struct {
	addr common.Address
}

type GetHostAddrResp struct {
	addr common.Address
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
	tokenAddress common.TokenAddress
	partnerAddress common.Address
	totalDeposit common.TokenAmount
}

type SetTotalChannelDepositResp struct {
	err error
}

type DirectTransferAsyncReq struct{
	amount common.TokenAmount
	target common.Address
	identifier common.PaymentID
}

type DirectTransferAsyncResp struct {
	ret bool
	err error
}

type MediaTransferReq struct {
	registryAddress common.PaymentNetworkID
	tokenAddress    common.TokenAddress
	amount common.TokenAmount
	target common.Address
	identifier common.PaymentID
}

type MediaTransferResp struct {
	ret bool
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
	tokenAddress common.TokenAddress
	partnerAddress common.Address
	totalWithdraw common.TokenAmount
}

type WithdrawResp struct {
	ret bool
	err error
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
	partnerAddress,totalDeposit}
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
		return directTransferAsyncResp.ret, nil
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
		return mediaTransferResp.ret, nil
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
		return withDrawResp.ret, nil
	}
}