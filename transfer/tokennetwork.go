package transfer

import (
	"reflect"

	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
)

func GetChannelId(stateChange StateChange) common.ChannelID {
	var result common.ChannelID

	switch stateChange.(type) {
	case *ActionChannelClose:
		actionChannelClose, _ := stateChange.(*ActionChannelClose)
		result = actionChannelClose.ChannelId
	case *ContractReceiveChannelNewBalance:
		contractReceiveChannelNewBalance, _ := stateChange.(*ContractReceiveChannelNewBalance)
		result = contractReceiveChannelNewBalance.ChannelId
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		result = contractReceiveChannelClosed.ChannelId
	case *ContractReceiveChannelSettled:
		contractReceiveChannelSettled, _ := stateChange.(*ContractReceiveChannelSettled)
		result = contractReceiveChannelSettled.ChannelId
	case *ContractReceiveUpdateTransfer:
		contractReceiveUpdateTransfer, _ := stateChange.(*ContractReceiveUpdateTransfer)
		result = contractReceiveUpdateTransfer.ChannelId
	case *ReceiveWithdrawRequest:
		receiveWithdrawRequest, _ := stateChange.(*ReceiveWithdrawRequest)
		result = receiveWithdrawRequest.ChannelId
	case *ReceiveWithdraw:
		receiveWithdraw, _ := stateChange.(*ReceiveWithdraw)
		result = receiveWithdraw.ChannelId
	case *ContractReceiveChannelWithdraw:
		contractReceiveChannelWithdraw, _ := stateChange.(*ContractReceiveChannelWithdraw)
		result = contractReceiveChannelWithdraw.ChannelId
	case *ReceiveCooperativeSettleRequest:
		receiveCooperativeSettleRequest, _ := stateChange.(*ReceiveCooperativeSettleRequest)
		result = receiveCooperativeSettleRequest.ChannelId
	case *ReceiveCooperativeSettle:
		receiveCooperativeSettle, _ := stateChange.(*ReceiveCooperativeSettle)
		result = receiveCooperativeSettle.ChannelId
	case *ContractReceiveChannelCooperativeSettled:
		contractReceiveChannelCooperativeSettled, _ := stateChange.(*ContractReceiveChannelCooperativeSettled)
		result = contractReceiveChannelCooperativeSettled.ChannelId
	}

	return result
}

func GetSenderAndMessageId(stateChange StateChange) (common.Address, common.MessageID) {
	var sender common.Address
	var messageId common.MessageID

	switch stateChange.(type) {
	case *ReceiveTransferDirect:
		v, _ := stateChange.(*ReceiveTransferDirect)
		sender = v.Sender
		messageId = v.MessageId
	case *ReceiveUnlock:
		v, _ := stateChange.(*ReceiveUnlock)
		sender = v.Sender
		messageId = v.MessageId
	case *ReceiveLockExpired:
		v, _ := stateChange.(*ReceiveLockExpired)
		sender = v.BalanceProof.Sender
		messageId = v.MessageId
	case *ReceiveSecretRequest:
		v, _ := stateChange.(*ReceiveSecretRequest)
		sender = v.Sender
		messageId = v.MessageId
	case *ReceiveSecretReveal:
		v, _ := stateChange.(*ReceiveSecretReveal)
		sender = v.Sender
		messageId = v.MessageId
	case *ReceiveDelivered:
		v, _ := stateChange.(*ReceiveDelivered)
		sender = v.Sender
		messageId = v.MessageId
	case *ReceiveProcessed:
		v, _ := stateChange.(*ReceiveProcessed)
		sender = v.Sender
		messageId = v.MessageId
	default:
		log.Warn("[GetSenderAndMessageId] eventType: ", reflect.TypeOf(stateChange).String())

	}

	return sender, messageId
}

func GetSenderMessageEvent(event Event) *SendMessageEvent {
	result := new(SendMessageEvent)
	switch event.(type) {
	case *SendDirectTransfer:
		v, _ := event.(*SendDirectTransfer)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendLockedTransfer:
		v, _ := event.(*SendLockedTransfer)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendLockExpired:
		v, _ := event.(*SendLockExpired)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendBalanceProof:
		v, _ := event.(*SendBalanceProof)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendSecretReveal:
		v, _ := event.(*SendSecretReveal)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendSecretRequest:
		v, _ := event.(*SendSecretRequest)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendRefundTransfer:
		v, _ := event.(*SendRefundTransfer)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendProcessed:
		v, _ := event.(*SendProcessed)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	// NOTE : dont put withdraw request in the queue so timeouted request will not be resend on startup
	//case *SendWithdrawRequest:
	//	v, _ := event.(*SendWithdrawRequest)
	//	result.Recipient = v.Recipient
	//	result.ChannelId = v.ChannelId
	//	result.MessageId = v.MessageId
	case *SendCooperativeSettleRequest:
		v, _ := event.(*SendCooperativeSettleRequest)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	case *SendCooperativeSettle:
		v, _ := event.(*SendCooperativeSettle)
		result.Recipient = v.Recipient
		result.ChannelId = v.ChannelId
		result.MessageId = v.MessageId
	default:
		//log.Warn("[GetSenderMessageEvent] eventType: ", reflect.TypeOf(event).String())
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
	case *ContractSendChannelWithdraw:
		return new(ContractSendEvent)
	case *ContractSendChannelCooperativeSettle:
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
	case *ContractReceiveChannelWithdraw:
		return match
	case *ContractReceiveChannelCooperativeSettled:
		return match
	}

	return empty
}

func subDispatchToChannelById(tokenNetworkState *TokenNetworkState,
	stateChange StateChange, blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	idsToChannels := tokenNetworkState.ChannelsMap
	channelId := GetChannelId(stateChange)

	channelState := idsToChannels[channelId]

	if channelState != nil {
		result := StateTransitionForChannel(channelState, stateChange, blockNumber)

		if tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address] == nil {
			tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address] = make(map[common.ChannelID]*NettingChannelState)
		}
		partnerToChannels := tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address]

		if IsStateNil(result.NewState) {
			log.Debugf("channel for channelId %d is deleted", channelId)
			delete(idsToChannels, channelId)
			delete(partnerToChannels, channelId)
		} else {
			idsToChannels[channelId] = result.NewState.(*NettingChannelState)
			partnerToChannels[channelId] = result.NewState.(*NettingChannelState)
		}
		events = append(events, result.Events...)
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleChannelClose(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	channelId := stateChange.(*ActionChannelClose).ChannelId
	tokenNetworkState.DelRoute(channelId)
	tokenNetworkState.UpdateAllDns()
	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleChannelNew(tokenNetworkState *TokenNetworkState,
	stateChange *ContractReceiveChannelNew) TransitionResult {
	channelState := stateChange.ChannelState
	channelId := channelState.Identifier

	ourAddress := channelState.OurState.Address
	partnerAddress := channelState.PartnerState.Address

	tokenNetworkState.AddRoute(ourAddress, partnerAddress, stateChange.ChannelId)
	tokenNetworkState.UpdateAllDns()
	_, ok := tokenNetworkState.ChannelsMap[channelId]
	if ok == false {
		tokenNetworkState.ChannelsMap[channelId] = channelState

		_, ok := tokenNetworkState.PartnerAddressesToChannels[partnerAddress]
		if ok == false {
			tokenNetworkState.PartnerAddressesToChannels[partnerAddress] =
				make(map[common.ChannelID]*NettingChannelState)
		}
		tokenNetworkState.PartnerAddressesToChannels[partnerAddress][channelId] = channelState
	}
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleBalance(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {
	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleClosed(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	channelId := stateChange.(*ContractReceiveChannelClosed).ChannelId
	tokenNetworkState.DelRoute(channelId)
	tokenNetworkState.UpdateAllDns()
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

func handleBatchUnlock(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	unlock := stateChange.(*ContractReceiveChannelBatchUnlock)

	participant1 := unlock.Participant
	participant2 := unlock.Partner

	for _, channelState := range tokenNetworkState.ChannelsMap {
		areAddressesValid1 := false
		areAddressesValid2 := false

		if common.AddressEqual(channelState.OurState.Address, participant1) &&
			common.AddressEqual(channelState.PartnerState.Address, participant2) {
			areAddressesValid1 = true
		}

		if common.AddressEqual(channelState.OurState.Address, participant2) &&
			common.AddressEqual(channelState.PartnerState.Address, participant1) {
			areAddressesValid2 = true
		}

		isValidLocksroot := true

		if (areAddressesValid1 || areAddressesValid2) && isValidLocksroot {
			iteration := StateTransitionForChannel(channelState, stateChange, blockNumber)
			events = append(events, iteration.Events[:]...)

			if IsStateNil(iteration.NewState) {
				log.Infof("handleBatchUnlock, delete channel")

				delete(tokenNetworkState.ChannelsMap, channelState.Identifier)

				if _, exist := tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address]; exist {
					delete(tokenNetworkState.PartnerAddressesToChannels[channelState.PartnerState.Address], channelState.Identifier)
				}
			}
		}
	}

	return TransitionResult{NewState: tokenNetworkState, Events: events}
}

func handleActionTransferDirect(paymentNetworkId common.PaymentNetworkID,
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
			PaymentNetworkId: paymentNetworkId,
			TokenNetworkId:   stateChange.TokenNetworkId,
			Identifier:       stateChange.PaymentId,
			Target:           common.Address(receiverAddress),
			Reason:           "Unknown partner channel",
		}

		events = append(events, failure)
	}

	return TransitionResult{NewState: tokenNetworkState, Events: events}
}

func handleNewRoute(tokenNetworkState *TokenNetworkState, stateChange *ContractReceiveRouteNew) TransitionResult {
	tokenNetworkState.AddRoute(stateChange.Participant1, stateChange.Participant2, stateChange.ChannelId)
	tokenNetworkState.UpdateAllDns()
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleCloseRoute(tokenNetworkState *TokenNetworkState, stateChange *ContractReceiveRouteClosed) TransitionResult {
	tokenNetworkState.DelRoute(stateChange.ChannelId)
	tokenNetworkState.UpdateAllDns()
	return TransitionResult{NewState: tokenNetworkState, Events: nil}
}

func handleReceiveTransferDirect(tokenNetworkState *TokenNetworkState,
	stateChange *ReceiveTransferDirect,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	channelId := stateChange.BalanceProof.ChannelId
	channelState := tokenNetworkState.ChannelsMap[channelId]

	if channelState != nil {
		result := StateTransitionForChannel(channelState, stateChange, blockNumber)
		events = append(events, result.Events...)
	}
	return TransitionResult{tokenNetworkState, events}
}

func handleActionWithdraw(paymentNetworkId common.PaymentNetworkID,
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

func handleReceiveWithdrawRequest(paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleReceiveWithdraw(paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleWithdraw(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func handleActionCooperativeSettle(paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange *ActionCooperativeSettle,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	channelState, ok := tokenNetworkState.ChannelsMap[stateChange.ChannelId]
	if ok && GetStatus(channelState) == ChannelStateOpened {
		iteration := StateTransitionForChannel(channelState, stateChange, blockNumber)
		events = iteration.Events
	}

	return TransitionResult{tokenNetworkState, events}
}

func handleReceiveCooperativeSettleRequest(paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleReceiveCooperativeSettle(paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}
func handleCoopeativeSettle(tokenNetworkState *TokenNetworkState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	return subDispatchToChannelById(tokenNetworkState, stateChange, blockNumber)
}

func stateTransitionForNetwork(paymentNetworkId common.PaymentNetworkID,
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
		iteration = handleActionTransferDirect(paymentNetworkId, tokenNetworkState,
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
		iteration = handleActionWithdraw(paymentNetworkId, tokenNetworkState,
			actionWithdraw, blockNumber)
	case *ReceiveWithdrawRequest:
		iteration = handleReceiveWithdrawRequest(paymentNetworkId, tokenNetworkState, stateChange, blockNumber)
	case *ReceiveWithdraw:
		iteration = handleReceiveWithdraw(paymentNetworkId, tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelWithdraw:
		iteration = handleWithdraw(tokenNetworkState, stateChange, blockNumber)
	case *ActionCooperativeSettle:
		actionCooperativeSettle, _ := stateChange.(*ActionCooperativeSettle)
		iteration = handleActionCooperativeSettle(paymentNetworkId, tokenNetworkState,
			actionCooperativeSettle, blockNumber)
	case *ReceiveCooperativeSettleRequest:
		iteration = handleReceiveWithdrawRequest(paymentNetworkId, tokenNetworkState, stateChange, blockNumber)
	case *ReceiveCooperativeSettle:
		iteration = handleReceiveWithdrawRequest(paymentNetworkId, tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelCooperativeSettled:
		iteration = handleCoopeativeSettle(tokenNetworkState, stateChange, blockNumber)
	case *ContractReceiveChannelBatchUnlock:
		iteration = handleBatchUnlock(tokenNetworkState, stateChange, blockNumber)
	}

	return iteration
}
