package channelservice

import (
	"container/list"
	"errors"
	"time"

	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

func WaitForBlock(channel *ChannelService, blockNumber common.BlockHeight,
	retryTimeout float32) {

	var currentBlockHeight common.BlockHeight

	chainState := channel.StateFromChannel()
	currentBlockHeight = transfer.GetBlockHeight(chainState)

	for currentBlockHeight < blockNumber {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		currentBlockHeight = transfer.GetBlockHeight(channel.StateFromChannel())
	}

	return
}

func WaitForNewChannel(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddress common.Address, retryTimeout float32) {

	var channelState *transfer.NettingChannelState

	chainState := channel.StateFromChannel()
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for channelState == nil {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		channelState = transfer.GetChannelStateFor(channel.StateFromChannel(), paymentNetworkId, tokenAddress,
			partnerAddress)
	}

	return
}

func addressEqual(address1 common.Address, address2 common.Address) bool {
	result := true

	for i := 0; i < 20; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}

func WaitForParticipantNewBalance(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddress common.Address, targetAddress common.Address,
	targetBalance common.TokenAmount, retryTimeout float32) error {

	balance := func(channelState *transfer.NettingChannelState) (common.TokenAmount, error) {
		var result common.TokenAmount

		ourState := channelState.GetChannelEndState(0)
		partnerState := channelState.GetChannelEndState(1)

		if addressEqual(targetAddress, channel.address) {
			result = ourState.GetContractBalance()
		} else if addressEqual(partnerAddress, channel.address) {
			result = partnerState.GetContractBalance()
		} else {
			return 0, errors.New("Target Address must be one of the channel participants!")
		}

		return result, nil
	}

	chainState := channel.StateFromChannel()

	var channelState *transfer.NettingChannelState
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for {
		if channelState == nil {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
			channelState = transfer.GetChannelStateFor(channel.StateFromChannel(), paymentNetworkId, tokenAddress,
				partnerAddress)
		} else {
			currentBalance, _ := balance(channelState)
			if currentBalance < targetBalance {
				time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
				channelState = transfer.GetChannelStateFor(channel.StateFromChannel(), paymentNetworkId, tokenAddress,
					partnerAddress)
			} else {
				break
			}
		}
	}
	return nil
}

func WaitForPaymentBalance(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddress common.Address, targetAddress common.Address,
	targetBalance common.TokenAmount, retryTimeout float32) {

	balance := func(channelState *transfer.NettingChannelState) (common.TokenAmount, error) {
		var result common.TokenAmount

		ourState := channelState.GetChannelEndState(0)
		partnerState := channelState.GetChannelEndState(1)

		if addressEqual(targetAddress, channel.address) {
			result = partnerState.GetGasBalance()
		} else if addressEqual(partnerAddress, channel.address) {
			result = ourState.GetGasBalance()
		} else {
			return 0, errors.New("Target Address must be one of the channel participants!")
		}

		return result, nil
	}

	chainState := channel.StateFromChannel()
	var channelState *transfer.NettingChannelState
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for {
		if channelState == nil {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
			channelState = transfer.GetChannelStateFor(channel.StateFromChannel(), paymentNetworkId, tokenAddress,
				partnerAddress)
		} else {

			currentBalance, _ := balance(channelState)
			if currentBalance < targetBalance {
				time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
				channelState = transfer.GetChannelStateFor(channel.StateFromChannel(), paymentNetworkId, tokenAddress,
					partnerAddress)
			} else {
				break
			}
		}
	}

	return
}

func WaitForClose(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, channelIds *list.List, retryTimeout float32) {

	len := channelIds.Len()
	for ; len > 0; len = channelIds.Len() {
		channelIsSettled := false

		e := channelIds.Back()
		channelState := transfer.GetChannelStateById(channel.StateFromChannel(),
			paymentNetworkId, tokenAddress, *(e.Value.(*common.ChannelID)))

		channelStatus := transfer.GetStatus(channelState)
		if channelState == nil || channelStatus == transfer.ChannelStateClosed ||
			channelStatus == transfer.ChannelStateSettled || channelStatus == transfer.ChannelStateSettling {
			channelIsSettled = true
		}

		if channelIsSettled {
			channelIds.Remove(e)
		} else {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		}
	}

	return
}

func WaitForPaymentNetwork(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, retryTimeout float32) {

	return
}

func WaitForSettle(channel *ChannelService, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, channelIds *list.List, retryTimeout float32) {

	len := channelIds.Len()
	for ; len > 0; len = channelIds.Len() {
		channelIsSettled := false

		e := channelIds.Back()
		channelState := transfer.GetChannelStateById(channel.StateFromChannel(),
			paymentNetworkId, tokenAddress, e.Value.(common.ChannelID))

		channelStatus := transfer.GetStatus(channelState)
		if channelState == nil || channelStatus == transfer.ChannelStateSettled {
			channelIsSettled = true
		}

		if channelIsSettled {
			channelIds.Remove(e)
		} else {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		}
	}

	return
}

func WaitForSettleAllChannels(channel *ChannelService, retryTimeout float32) {
	tokenNetworkState := transfer.GetTokenNetworkByTokenAddress(channel.StateFromChannel(),
		common.PaymentNetworkID{}, common.TokenAddress{})

	channelIds := list.New()
	for k := range tokenNetworkState.ChannelIdentifiersToChannels {
		channelIds.PushBack(k)
	}

	WaitForSettle(
		channel,
		common.PaymentNetworkID{},
		common.TokenAddress{},
		channelIds,
		retryTimeout)

	return
}

func WaitForhealthy(channel *ChannelService, nodeAddress common.Address, retryTimeout float32) {
	var networkStatuses *map[common.Address]string

	networkStatuses = transfer.GetNetworkStatuses(channel.StateFromChannel())
	for (*networkStatuses)[nodeAddress] != transfer.NetworkReachable {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		networkStatuses = transfer.GetNetworkStatuses(channel.StateFromChannel())
	}

	return
}

func WaitForTransferSuccess(channel *ChannelService, paymentIdentifier common.PaymentID,
	amount common.PaymentAmount, retryTimeout float32) {

	found := false
	for found == false {
		stateEvents := channel.Wal.Storage.GetEvents(-1, 0)

		for e := stateEvents.Front(); e != nil; e = e.Next() {
			if event, ok := e.Value.(*transfer.EventPaymentReceivedSuccess); ok {
				if event.Identifier == paymentIdentifier && common.PaymentAmount(event.Amount) == amount {
					found = true
					break
				}
			}

		}
	}

	return
}
