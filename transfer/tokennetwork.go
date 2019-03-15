package transfer

import (
	"reflect"

	"github.com/oniio/oniChannel/common"
)

func GetChannelIdentifier(stateChange StateChange) common.ChannelID {
	var result common.ChannelID

	switch stateChange.(type) {
	case *ActionChannelClose:
		actionChannelClose, _ := stateChange.(*ActionChannelClose)
		result = actionChannelClose.ChannelIdentifier
	case *ContractReceiveChannelNewBalance:
		contractReceiveChannelNewBalance, _ := stateChange.(*ContractReceiveChannelNewBalance)
		result = contractReceiveChannelNewBalance.ChannelIdentifier
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		result = contractReceiveChannelClosed.ChannelIdentifier
	case *ContractReceiveChannelSettled:
		contractReceiveChannelSettled, _ := stateChange.(*ContractReceiveChannelSettled)
		result = contractReceiveChannelSettled.ChannelIdentifier
	case *ContractReceiveUpdateTransfer:
		contractReceiveUpdateTransfer, _ := stateChange.(*ContractReceiveUpdateTransfer)
		result = contractReceiveUpdateTransfer.ChannelIdentifier
	}

	return result
}

func GetSenderAndMessageIdentifier(stateChange StateChange) (common.Address, common.MessageID) {
	var sender common.Address
	var messgeId common.MessageID

	switch stateChange.(type) {
	case *ReceiveTransferDirect:
		v, _ := stateChange.(*ReceiveTransferDirect)
		sender = v.Sender
		messgeId = v.MessageIdentifier
	case *ReceiveUnlock:
		v, _ := stateChange.(*ReceiveUnlock)
		sender = v.Sender
		messgeId = v.MessageIdentifier
	case *ReceiveDelivered:
		v, _ := stateChange.(*ReceiveDelivered)
		sender = v.Sender
		messgeId = v.MessageIdentifier
	case *ReceiveProcessed:
		v, _ := stateChange.(*ReceiveProcessed)
		sender = v.Sender
		messgeId = v.MessageIdentifier
	}

	return sender, messgeId
}

func GetSenderMessageEvent(event Event) *SendMessageEvent {
	result := new(SendMessageEvent)

	switch event.(type) {
	case *SendDirectTransfer:
		v, _ := event.(*SendDirectTransfer)
		result.Recipient = v.Recipient
		result.ChannelIdentifier = v.ChannelIdentifier
		result.MessageIdentifier = v.MessageIdentifier
	case *SendProcessed:
		v, _ := event.(*SendProcessed)
		result.Recipient = v.Recipient
		result.ChannelIdentifier = v.ChannelIdentifier
		result.MessageIdentifier = v.MessageIdentifier
	default:
		result = nil
	}

	return result
}

func GetContractSendEvent(event Event) *ContractSendEvent {
	var result *ContractSendEvent

	switch event.(type) {
	case *ContractSendChannelClose:
		return new(ContractSendEvent)
	case *ContractSendChannelSettle:
		return new(ContractSendEvent)
	case *ContractSendChannelUpdateTransfer:
		return new(ContractSendEvent)
	}

	return result
}

func GetContractReceiveStateChange(stateChange StateChange) *ContractReceiveStateChange {
	var empty *ContractReceiveStateChange

	match := new(ContractReceiveStateChange)
	switch stateChange.(type) {
	case *ContractReceiveChannelNew:
		return match
	case *ContractReceiveChannelClosed:
		return match
	case *ContractReceiveChannelSettled:
		return match
	case *ContractReceiveChannelNewBalance:
		return match
	case *ContractReceiveUpdateTransfer:
		return match
	}

	return empty
}

func subDispatchToChannelById(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	idsToChannels := tokenNetworkState.ChannelIdentifiersToChannels
	channelIdentifier := GetChannelIdentifier(stateChange)

	channelState := idsToChannels[channelIdentifier]

	if channelState != nil {
		result := StateTransitionForChannel(channelState, stateChange, blockNumber)

		if tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address] == nil {
			tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address] = make(map[common.ChannelID]*NettingChannelState)
		}
		partnerToChannels := tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address]

		if reflect.ValueOf(result.NewState).IsNil() {
			delete(idsToChannels, channelIdentifier)
			delete(partnerToChannels, channelIdentifier)
		} else {
			idsToChannels[channelIdentifier] = result.NewState.(*NettingChannelState)
			partnerToChannels[channelIdentifier] = result.NewState.(*NettingChannelState)
		}
		events = append(events, result.Events...)
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleChannelClose(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	channelIdentifier := stateChange.(*ActionChannelClose).ChannelIdentifier
	tokenNetworkState.DelRoute(channelIdentifier)
	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleChannelNew(tokenNetworkState *TokenNetworkState,
	stateChange *ContractReceiveChannelNew) TransitionResult {
	channelState := stateChange.ChannelState
	channelIdentifier := channelState.Identifier

	ourAddress := channelState.OurState.Address
	partnerAddress := channelState.PartnerState.Address

	tokenNetworkState.AddRoute(ourAddress, partnerAddress, stateChange.ChannelIdentifier)
	_, ok := tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier]
	if ok == false {
		tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier] = channelState

		_, ok := tokenNetworkState.PartnerAddressesToChannels[partnerAddress]
		if ok == false {
			tokenNetworkState.PartnerAddressesToChannels[partnerAddress] =
				make(map[common.ChannelID]*NettingChannelState)
		}
		tokenNetworkState.PartnerAddressesToChannels[partnerAddress][channelIdentifier] = channelState
	}
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleBalance(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {
	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleClosed(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	channelIdentifier := stateChange.(*ContractReceiveChannelClosed).ChannelIdentifier
	tokenNetworkState.DelRoute(channelIdentifier)
	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleSettled(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleUpdatedTransfer(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleActionTransferDirect(paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange *ActionTransferDirect,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	receiverAddress := stateChange.ReceiverAddress
	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	channelStates := FilterChannelsByStatus(tokenNetworkState.PartnerAddressesToChannels[receiverAddress],
		excludeStates)

	if channelStates != nil && channelStates.Len() != 0 {
		iteration := StateTransitionForChannel(channelStates.Back().Value.(*NettingChannelState),
			stateChange, blockNumber)
		events = iteration.Events
	} else {
		failure := &EventPaymentSentFailed{
			PaymentNetworkIdentifier: paymentNetworkIdentifier,
			TokenNetworkIdentifier:   stateChange.TokenNetworkIdentifier,
			Identifier:               stateChange.PaymentIdentifier,
			Target:                   common.Address(receiverAddress),
			Reason:                   "Unknown partner channel",
		}

		events = append(events, failure)
	}

	return TransitionResult{NewState: tokenNetworkState, Events: events}
}

func handleNewRoute(tokenNetworkState *TokenNetworkState, stateChange *ContractReceiveRouteNew) TransitionResult {
	tokenNetworkState.AddRoute(stateChange.Participant1, stateChange.Participant2, stateChange.ChannelIdentifier)
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleCloseRoute(tokenNetworkState *TokenNetworkState, stateChange *ContractReceiveRouteClosed) TransitionResult {
	tokenNetworkState.DelRoute(stateChange.ChannelIdentifier)
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleReceiveTransferDirect(tokenNetworkState *TokenNetworkState,
	stateChange *ReceiveTransferDirect,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	channelIdentifier := stateChange.BalanceProof.ChannelIdentifier
	channelState := tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier]

	if channelState != nil {
		result := StateTransitionForChannel(channelState, stateChange, blockNumber)
		events = append(events, result.Events...)
	}
	return TransitionResult{tokenNetworkState, events}
}

func handleActionWithdraw(paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange *ActionWithdraw,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	receiverAddress := stateChange.Partner
	excludeStates := make(map[string]int)
	excludeStates[ChannelStateClosed] = 0
	excludeStates[ChannelStateClosing] = 0
	excludeStates[ChannelStateSettled] = 0
	excludeStates[ChannelStateSettling] = 0
	excludeStates[ChannelStateUnusable] = 0

	channelStates := FilterChannelsByStatus(tokenNetworkState.PartnerAddressesToChannels[receiverAddress],
		excludeStates)

	if channelStates != nil && channelStates.Len() != 0 {
		iteration := StateTransitionForChannel(channelStates.Back().Value.(*NettingChannelState),
			stateChange, blockNumber)
		events = iteration.Events
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleReceiveWithdrawRequest(paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleReceiveWithdraw(paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleWithdraw(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func stateTransitionForNetwork(paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	iteration := TransitionResult{}

	switch stateChange.(type) {
	case *ActionChannelClose:
		iteration = handleChannelClose(tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelNew:
		contractReceiveChannelNew, _ := stateChange.(*ContractReceiveChannelNew)
		iteration = handleChannelNew(tokenNetworkState, contractReceiveChannelNew)
	case *ContractReceiveChannelNewBalance:
		iteration = handleBalance(tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelClosed:
		iteration = handleClosed(tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelSettled:
		iteration = handleSettled(tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveUpdateTransfer:
		iteration = handleUpdatedTransfer(tokenNetworkState, stateChange, blockNumber)
	case *ActionTransferDirect:
		actionTransferDirect, _ := stateChange.(*ActionTransferDirect)
		iteration = handleActionTransferDirect(paymentNetworkIdentifier, tokenNetworkState,
			actionTransferDirect, blockNumber)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleReceiveTransferDirect(tokenNetworkState, receiveTransferDirect, blockNumber)
	case *ContractReceiveRouteNew:
		contractReceiveRouteNew, _ := stateChange.(*ContractReceiveRouteNew)
		iteration = handleNewRoute(tokenNetworkState, contractReceiveRouteNew)
	case *ContractReceiveRouteClosed:
		contractReceiveRouteClosed, _ := stateChange.(*ContractReceiveRouteClosed)
		iteration = handleCloseRoute(tokenNetworkState, contractReceiveRouteClosed)
	case *ActionWithdraw:
		actionWithdraw, _ := stateChange.(*ActionWithdraw)
		iteration = handleActionWithdraw(paymentNetworkIdentifier, tokenNetworkState,
			actionWithdraw, blockNumber)
	case *ReceiveWithdrawRequest:
		iteration = handleReceiveWithdrawRequest(paymentNetworkIdentifier, tokenNetworkState, stateChange, blockNumber)
	case *ReceiveWithdraw:
		iteration = handleReceiveWithdraw(paymentNetworkIdentifier, tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelWithdraw:
		iteration = handleWithdraw(tokenNetworkState, stateChange, blockNumber)
	}

	return iteration
}
