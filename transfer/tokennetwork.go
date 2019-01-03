package transfer

import (
	"container/list"
	"math/rand"
	"reflect"

	"github.com/oniio/oniChannel/typing"
)

func GetChannelIdentifier(stateChange StateChange) typing.ChannelID {
	var result typing.ChannelID

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

func GetSenderAndMessageIdentifier(stateChange StateChange) (typing.Address, typing.MessageID) {
	var sender typing.Address
	var messgeId typing.MessageID

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

func subdispatchToChannelById(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	idsToChannels := tokenNetworkState.ChannelIdentifiersToChannels
	channelIdentifier := GetChannelIdentifier(stateChange)

	channelState := idsToChannels[channelIdentifier]

	if channelState != nil {
		result := StateTransitionForChannel(
			channelState,
			stateChange,
			pseudoRandomGenerator,
			blockNumber)

		if tokenNetworkState.partnerAddressesToChannels[channelState.PartnerState.Address] == nil {
			tokenNetworkState.partnerAddressesToChannels[channelState.PartnerState.Address] = make(map[typing.ChannelID]*NettingChannelState)
		}
		partnerToChannels := tokenNetworkState.partnerAddressesToChannels[channelState.PartnerState.Address]

		if reflect.ValueOf(result.NewState).IsNil() {
			delete(idsToChannels, channelIdentifier)
			delete(partnerToChannels, channelIdentifier)
		} else {
			idsToChannels[channelIdentifier] = result.NewState.(*NettingChannelState)
			partnerToChannels[channelIdentifier] = result.NewState.(*NettingChannelState)
		}
		events.PushBackList(result.Events)
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleChannelClose(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {
	return subdispatchToChannelById(
		tokenNetworkState,
		stateChange,
		pseudoRandomGenerator,
		blockNumber)
}

func handleChannelNew(tokenNetworkState *TokenNetworkState,
	stateChange *ContractReceiveChannelNew) TransitionResult {

	events := list.New()

	channelState := stateChange.ChannelState
	channelIdentifier := channelState.Identifier

	partnerAddress := channelState.PartnerState.Address

	//[TODO] add channel info to TokenNetworkGraphState when support routing
	//ourAddress := channelState.OurState.Address

	_, ok := tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier]
	if ok == false {
		tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier] = channelState

		_, ok := tokenNetworkState.partnerAddressesToChannels[partnerAddress]
		if ok == false {
			tokenNetworkState.partnerAddressesToChannels[partnerAddress] = make(map[typing.ChannelID]*NettingChannelState)
		}

		tokenNetworkState.partnerAddressesToChannels[partnerAddress][channelIdentifier] = channelState
	}

	return TransitionResult{tokenNetworkState, events}

}

func handleBalance(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {
	return subdispatchToChannelById(
		tokenNetworkState,
		stateChange,
		pseudoRandomGenerator,
		blockNumber)
}

func handleClosed(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {

	//[TODO] remove channel from TokenNetworkGraphState when support routing
	return subdispatchToChannelById(
		tokenNetworkState,
		stateChange,
		pseudoRandomGenerator,
		blockNumber)
}

func handleSettled(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {

	return subdispatchToChannelById(
		tokenNetworkState,
		stateChange,
		pseudoRandomGenerator,
		blockNumber)
}

func handleUpdatedTransfer(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {

	return subdispatchToChannelById(
		tokenNetworkState,
		stateChange,
		pseudoRandomGenerator,
		blockNumber)
}

func handleActionTransferDirect(paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange *ActionTransferDirect,
	pseudoRandomGenerator *rand.Rand, blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	receiverAddress := stateChange.ReceiverAddress
	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	channelStates := FilterChannelsByStatus(tokenNetworkState.partnerAddressesToChannels[receiverAddress],
		excludeStates)

	if channelStates != nil && channelStates.Len() != 0 {
		iteration := StateTransitionForChannel(
			channelStates.Back().Value.(*NettingChannelState),
			stateChange,
			pseudoRandomGenerator,
			blockNumber)
		events = iteration.Events
	} else {
		failure := &EventPaymentSentFailed{
			paymentNetworkIdentifier,
			stateChange.TokenNetworkIdentifier,
			stateChange.PaymentIdentifier,
			typing.TargetAddress(receiverAddress),
			"Unknown partner channel"}

		events.PushBack(failure)
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleReceiveTransferDirect(tokenNetworkState *TokenNetworkState,
	stateChange *ReceiveTransferDirect, pseudoRandomGenerator *rand.Rand,
	blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	channelIdentifier := stateChange.BalanceProof.ChannelIdentifier
	channelState := tokenNetworkState.ChannelIdentifiersToChannels[channelIdentifier]

	if channelState != nil {
		result := StateTransitionForChannel(channelState, stateChange,
			pseudoRandomGenerator, blockNumber)
		events.PushBackList(result.Events)
	}

	return TransitionResult{tokenNetworkState, events}
}

func stateTransitionForNetwork(paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	pseudoRandomGenerator *rand.Rand, blockNumber typing.BlockHeight) TransitionResult {

	iteration := TransitionResult{}

	switch stateChange.(type) {
	case *ActionChannelClose:
		iteration = handleChannelClose(tokenNetworkState, stateChange,
			pseudoRandomGenerator, blockNumber)
	case *ContractReceiveChannelNew:
		contractReceiveChannelNew, _ := stateChange.(*ContractReceiveChannelNew)
		iteration = handleChannelNew(tokenNetworkState, contractReceiveChannelNew)
	case *ContractReceiveChannelNewBalance:
		iteration = handleBalance(tokenNetworkState, stateChange,
			pseudoRandomGenerator, blockNumber)
	case *ContractReceiveChannelClosed:
		iteration = handleClosed(tokenNetworkState, stateChange,
			pseudoRandomGenerator, blockNumber)
	case *ContractReceiveChannelSettled:
		iteration = handleSettled(tokenNetworkState, stateChange,
			pseudoRandomGenerator, blockNumber)
	case *ContractReceiveUpdateTransfer:
		iteration = handleUpdatedTransfer(tokenNetworkState, stateChange,
			pseudoRandomGenerator, blockNumber)
	case *ActionTransferDirect:
		actionTransferDirect, _ := stateChange.(*ActionTransferDirect)
		iteration = handleActionTransferDirect(paymentNetworkIdentifier,
			tokenNetworkState, actionTransferDirect, pseudoRandomGenerator, blockNumber)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleReceiveTransferDirect(tokenNetworkState, receiveTransferDirect,
			pseudoRandomGenerator, blockNumber)
	}

	return iteration
}
