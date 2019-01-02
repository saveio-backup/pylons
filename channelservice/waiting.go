package channelservice

import (
	"container/list"
	"errors"
	"time"

	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

func WaitForBlock(nimbus *ChannelService, blockNumber typing.BlockNumber,
	retryTimeout float32) {

	var currentBlockNumber typing.BlockNumber

	chainState := nimbus.StateFromNimbus()
	currentBlockNumber = transfer.GetBlockNumber(chainState)

	for currentBlockNumber < blockNumber {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		currentBlockNumber = transfer.GetBlockNumber(nimbus.StateFromNimbus())
	}

	return
}

func WaitForNewChannel(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address, retryTimeout float32) {

	var channelState *transfer.NettingChannelState

	chainState := nimbus.StateFromNimbus()
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for channelState == nil {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		channelState = transfer.GetChannelStateFor(nimbus.StateFromNimbus(), paymentNetworkId, tokenAddress,
			partnerAddress)
	}

	return
}

func addressEqual(address1 typing.Address, address2 typing.Address) bool {
	result := true

	for i := 0; i < 20; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}

func WaitForParticipantNewBalance(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address, targetAddress typing.Address,
	targetBalance typing.TokenAmount, retryTimeout float32) error {

	balance := func(channelState *transfer.NettingChannelState) (typing.TokenAmount, error) {
		var result typing.TokenAmount

		ourState := channelState.GetChannelEndState(0)
		partnerState := channelState.GetChannelEndState(1)

		if addressEqual(targetAddress, nimbus.address) {
			result = ourState.GetContractBalance()
		} else if addressEqual(partnerAddress, nimbus.address) {
			result = partnerState.GetContractBalance()
		} else {
			return 0, errors.New("Target Address must be one of the channel participants!")
		}

		return result, nil
	}

	chainState := nimbus.StateFromNimbus()

	var channelState *transfer.NettingChannelState
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for {
		if channelState == nil {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
			channelState = transfer.GetChannelStateFor(nimbus.StateFromNimbus(), paymentNetworkId, tokenAddress,
				partnerAddress)
		} else {
			currentBalance, _ := balance(channelState)
			if currentBalance < targetBalance {
				time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
				channelState = transfer.GetChannelStateFor(nimbus.StateFromNimbus(), paymentNetworkId, tokenAddress,
					partnerAddress)
			} else {
				break
			}
		}
	}
	return nil
}

func WaitForPaymentBalance(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address, targetAddress typing.Address,
	targetBalance typing.TokenAmount, retryTimeout float32) {

	balance := func(channelState *transfer.NettingChannelState) (typing.TokenAmount, error) {
		var result typing.TokenAmount

		ourState := channelState.GetChannelEndState(0)
		partnerState := channelState.GetChannelEndState(1)

		if addressEqual(targetAddress, nimbus.address) {
			result = partnerState.GetBalance()
		} else if addressEqual(partnerAddress, nimbus.address) {
			result = ourState.GetBalance()
		} else {
			return 0, errors.New("Target Address must be one of the channel participants!")
		}

		return result, nil
	}

	chainState := nimbus.StateFromNimbus()
	var channelState *transfer.NettingChannelState
	channelState = transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress,
		partnerAddress)

	for {
		if channelState == nil {
			time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
			channelState = transfer.GetChannelStateFor(nimbus.StateFromNimbus(), paymentNetworkId, tokenAddress,
				partnerAddress)
		} else {

			currentBalance, _ := balance(channelState)
			if currentBalance < targetBalance {
				time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
				channelState = transfer.GetChannelStateFor(nimbus.StateFromNimbus(), paymentNetworkId, tokenAddress,
					partnerAddress)
			} else {
				break
			}
		}
	}

	return
}

func WaitForClose(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, channelIds *list.List, retryTimeout float32) {

	len := channelIds.Len()
	for ; len > 0; len = channelIds.Len() {
		channelIsSettled := false

		e := channelIds.Back()
		channelState := transfer.GetChannelStateById(nimbus.StateFromNimbus(),
			paymentNetworkId, tokenAddress, *(e.Value.(*typing.ChannelID)))

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

func WaitForPaymentNetwork(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, retryTimeout float32) {

	return
}

func WaitForSettle(nimbus *ChannelService, paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, channelIds *list.List, retryTimeout float32) {

	len := channelIds.Len()
	for ; len > 0; len = channelIds.Len() {
		channelIsSettled := false

		e := channelIds.Back()
		channelState := transfer.GetChannelStateById(nimbus.StateFromNimbus(),
			paymentNetworkId, tokenAddress, e.Value.(typing.ChannelID))

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

func WaitForSettleAllChannels(nimbus *ChannelService, retryTimeout float32) {
	tokenNetworkState := transfer.GetTokenNetworkByTokenAddress(nimbus.StateFromNimbus(),
		typing.PaymentNetworkID{}, typing.TokenAddress{})

	channelIds := list.New()
	for k := range tokenNetworkState.ChannelIdentifiersToChannels {
		channelIds.PushBack(k)
	}

	WaitForSettle(
		nimbus,
		typing.PaymentNetworkID{},
		typing.TokenAddress{},
		channelIds,
		retryTimeout)

	return
}

func WaitForhealthy(nimbus *ChannelService, nodeAddress typing.Address, retryTimeout float32) {
	var networkStatuses *map[typing.Address]string

	networkStatuses = transfer.GetNetworkStatuses(nimbus.StateFromNimbus())
	for (*networkStatuses)[nodeAddress] != transfer.NodeNetworkReachable {
		time.Sleep(time.Duration(retryTimeout*1000) * time.Millisecond)
		networkStatuses = transfer.GetNetworkStatuses(nimbus.StateFromNimbus())
	}

	return
}

func WaitForTransferSuccess(nimbus *ChannelService, paymentIdentifier typing.PaymentID,
	amount typing.PaymentAmount, retryTimeout float32) {

	found := false
	for found == false {
		stateEvents := nimbus.Wal.Storage.GetEvents(-1, 0)

		for e := stateEvents.Front(); e != nil; e = e.Next() {
			if event, ok := e.Value.(*transfer.EventPaymentReceivedSuccess); ok {
				if event.Identifier == paymentIdentifier && typing.PaymentAmount(event.Amount) == amount {
					found = true
					break
				}
			}

		}
	}

	return
}
