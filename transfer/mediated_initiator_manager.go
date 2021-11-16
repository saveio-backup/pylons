package transfer

import (
	"crypto/sha256"
	"reflect"

	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
)

func RefundTransferMatchesReceived(refundTransfer *LockedTransferSignedState,
	receivedTransfer *LockedTransferUnsignedState) bool {

	refundTransferSender := refundTransfer.BalanceProof.Sender
	if refundTransferSender == receivedTransfer.Target {
		return false
	}

	if receivedTransfer.PaymentId == refundTransfer.PaymentId &&
		receivedTransfer.Lock.Amount == refundTransfer.Lock.Amount &&
		receivedTransfer.Lock.SecretHash == refundTransfer.Lock.SecretHash &&
		receivedTransfer.Target == common.Address(refundTransfer.Target) &&
		receivedTransfer.Lock.Expiration == refundTransfer.Lock.Expiration &&
		receivedTransfer.Token == common.Address(refundTransfer.Token) {
		return true
	}

	return false
}

func HandleReceiveRefundTransferCancelRoute(channelState *NettingChannelState,
	refundTransfer *ReceiveTransferRefundCancelRoute) ([]Event, error) {

	return HandleReceiveLockedTransfer(channelState, refundTransfer.Transfer)
}

//-----------------------------------------------------------------

func ImIterationFromSub(paymentState *InitiatorPaymentState, iteration *TransitionResult) *TransitionResult {
	if !IsStateNil(iteration.NewState) {
		paymentState.Initiator = iteration.NewState.(*InitiatorTransferState)
		return &TransitionResult{NewState: paymentState, Events: iteration.Events}
	}

	return iteration
}

func ImCanCancel(paymentState *InitiatorPaymentState) bool {
	return paymentState.Initiator == nil || paymentState.Initiator.RevealSecret == nil
}

func ImEventsForCancelCurrentRoute(transferDescription *TransferDescriptionWithSecretState) []Event {
	var events []Event

	unlockFailed := &EventUnlockFailed{
		Identifier: transferDescription.PaymentId,
		SecretHash: transferDescription.SecretHash,
		Reason:     string("route was canceled")}
	events = append(events, unlockFailed)
	return events
}

func ImCancelCurrentRoute(paymentState *InitiatorPaymentState) []Event {
	if ImCanCancel(paymentState) == false {
		return nil
	}

	transferDescription := paymentState.Initiator.TransferDescription
	paymentState.CancelledChannels = append(paymentState.CancelledChannels, paymentState.Initiator.ChannelId)
	paymentState.Initiator = nil

	return ImEventsForCancelCurrentRoute(transferDescription)
}

func ImHandleBlock(paymentState *InitiatorPaymentState, stateChange *Block,
	channelIdToChannels map[common.ChannelID]*NettingChannelState) *TransitionResult {

	channelId := paymentState.Initiator.ChannelId
	channelState := channelIdToChannels[channelId]
	if channelState == nil {
		return &TransitionResult{NewState: paymentState, Events: nil}
	}

	subIteration := InitHandleBlock(paymentState.Initiator, stateChange, channelState)
	iteration := ImIterationFromSub(paymentState, subIteration)

	return iteration

}

func ImHandleInit(paymentState *InitiatorPaymentState, stateChange *ActionInitInitiator,
	channelIdToChannels map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	var events []Event

	if paymentState == nil {
		subIteration := InitTryNewRoute(nil, channelIdToChannels,
			stateChange.Routes, stateChange.TransferDescription, blockNumber, true)

		events = subIteration.Events
		if !IsStateNil(subIteration.NewState) {
			transferState := subIteration.NewState.(*InitiatorTransferState)
			return &TransitionResult{NewState: transferState, Events: events}
		}
	} else {
		events = nil
	}
	return &TransitionResult{NewState: paymentState, Events: events}
}

func ImHandleCancelRoute(paymentState *InitiatorPaymentState, stateChange *ActionCancelRoute,
	channelIdToChannels map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	var events []Event
	if ImCanCancel(paymentState) == true {
		//oldInitiatorState := paymentState.Initiator
		transferDescription := paymentState.Initiator.TransferDescription
		cancelEvents := ImCancelCurrentRoute(paymentState)

		if paymentState.Initiator != nil {
			//[TODO] wirte log or fmt "The previous transfer must be cancelled prior to trying a new route"
		}
		//msg := "The previous transfer must be cancelled prior to trying a new route"
		subIteration := InitTryNewRoute(nil, channelIdToChannels, stateChange.Routes,
			transferDescription, blockNumber, false)
		events = append(events, cancelEvents...)
		events = append(events, subIteration.Events...)

		if !IsStateNil(subIteration.NewState) {
			paymentState.Initiator = subIteration.NewState.(*InitiatorTransferState)
		} else {
			paymentState = nil
		}
	}

	return &TransitionResult{NewState: paymentState, Events: events}
}

func ImHandleCancelPayment(paymentState *InitiatorPaymentState, channelState *NettingChannelState) *TransitionResult {
	var iteration *TransitionResult

	if ImCanCancel(paymentState) {
		transferDescription := paymentState.Initiator.TransferDescription
		cancelEvents := ImCancelCurrentRoute(paymentState)

		cancel := &EventPaymentSentFailed{
			PaymentNetworkId: channelState.PaymentNetworkId,
			TokenNetworkId:   channelState.TokenNetworkId,
			Identifier:       transferDescription.PaymentId,
			Target:           common.Address(transferDescription.Target),
			Reason:           string("user canceld payment"),
		}

		cancelEvents = append(cancelEvents, cancel)
		iteration = &TransitionResult{NewState: nil, Events: cancelEvents}
	} else {
		iteration = &TransitionResult{NewState: paymentState, Events: nil}
	}

	return iteration
}

func ImHandleTransferRefundCancelRoute(paymentState *InitiatorPaymentState,
	stateChange *ReceiveTransferRefundCancelRoute, channelIdToChannels map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	channelId := paymentState.Initiator.ChannelId
	channelState := channelIdToChannels[channelId]
	refundTransfer := stateChange.Transfer
	originalTransfer := paymentState.Initiator.Transfer

	isValidLock := false
	if refundTransfer.Lock.SecretHash == originalTransfer.Lock.SecretHash &&
		refundTransfer.Lock.Amount == originalTransfer.Lock.Amount &&
		refundTransfer.Lock.Expiration == originalTransfer.Lock.Expiration {
		isValidLock = true
	}

	isValidRefund := RefundTransferMatchesReceived(refundTransfer, originalTransfer)

	var events []Event
	if isValidLock && isValidRefund {
		channelEvents, err := HandleReceiveRefundTransferCancelRoute(channelState, stateChange)
		events = append(events, channelEvents...)

		if err == nil {
			oldDescription := paymentState.Initiator.TransferDescription
			transferDescription := &TransferDescriptionWithSecretState{
				PaymentNetworkId: oldDescription.PaymentNetworkId,
				PaymentId:        oldDescription.PaymentId,
				Amount:           oldDescription.Amount,
				TokenNetworkId:   oldDescription.TokenNetworkId,
				Initiator:        oldDescription.Initiator,
				Target:           oldDescription.Target,
				Secret:           stateChange.Secret,
				SecretHash:       sha256.Sum256(stateChange.Secret[:]),
			}
			paymentState.Initiator.TransferDescription = transferDescription

			//Construct fake ActionCancelRoute, only need routes field!
			actionCancelRoute := &ActionCancelRoute{Routes: stateChange.Routes}
			subIteration := ImHandleCancelRoute(paymentState, actionCancelRoute, channelIdToChannels, blockNumber)

			events = append(events, subIteration.Events...)
			if IsStateNil(subIteration.NewState) {
				paymentState = nil
			}
		}
	}

	return &TransitionResult{NewState: paymentState, Events: events}
}

func ImHandleLockExpired(paymentState *InitiatorPaymentState, stateChange *ReceiveLockExpired,
	channelIdToChannels map[common.ChannelID]*NettingChannelState, blockNumber common.BlockHeight) *TransitionResult {

	//"""Initiator also needs to handle LockExpired messages when refund transfers are involved.
	//
	//A -> B -> C
	//
	//- A sends locked transfer to B
	//- B attempted to forward to C but has not enough capacity
	//- B sends a refund transfer with the same secrethash back to A
	//- When the lock expires B will also send a LockExpired message to A
	//- A needs to be able to properly process it
	//
	//Related issue: https://github.com/raiden-network/raiden/issues/3183
	//"""
	channelId := paymentState.Initiator.ChannelId
	channelState := channelIdToChannels[channelId]
	secretHash := paymentState.Initiator.Transfer.Lock.SecretHash
	result := handleReceiveLockExpired(channelState, stateChange, blockNumber)

	nettingChannelState := result.NewState.(*NettingChannelState)
	state := GetLock(nettingChannelState.PartnerState, common.SecretHash(secretHash))
	if state != nil {
		transfer := paymentState.Initiator.Transfer
		unlockFailed := &EventUnlockClaimFailed{
			Identifier: transfer.PaymentId,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
			Reason:     "Lock expired",
		}
		result.Events = append(result.Events, unlockFailed)
	}
	return &TransitionResult{NewState: paymentState, Events: result.Events}
}

func ImHandleOffchainSecretReveal(paymentState *InitiatorPaymentState, stateChange *ReceiveSecretReveal,
	channelIdToChannels map[common.ChannelID]*NettingChannelState) *TransitionResult {

	channelId := paymentState.Initiator.ChannelId
	channelState := channelIdToChannels[channelId]
	subIteration := InitHandleOffChainSecretReveal(paymentState.Initiator, stateChange, channelState)
	return ImIterationFromSub(paymentState, subIteration)
}

func ImHandleOnChainSecretReveal(paymentState *InitiatorPaymentState, stateChange *ContractReceiveSecretReveal,
	channelIdToChannels map[common.ChannelID]*NettingChannelState) *TransitionResult {

	channelId := paymentState.Initiator.ChannelId
	channelState := channelIdToChannels[channelId]

	subIteration := InitHandleOnChainSecretReveal(paymentState.Initiator, stateChange, channelState)
	iteration := ImIterationFromSub(paymentState, subIteration)
	return iteration
}

func ImStateTransition(paymentState *InitiatorPaymentState, stateChange interface{},
	channelIdToChannels map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {
	log.Debug("[ImStateTransition] stateChange Type: ", reflect.TypeOf(stateChange).String())

	var iteration *TransitionResult
	switch stateChange.(type) {
	case *Block:
		block, _ := stateChange.(*Block)
		iteration = ImHandleBlock(paymentState, block, channelIdToChannels)
	case *ActionInitInitiator:
		actionInitInitiator, _ := stateChange.(*ActionInitInitiator)
		iteration = ImHandleInit(paymentState, actionInitInitiator, channelIdToChannels, blockNumber)
	case *ReceiveSecretRequest:
		receiveSecretRequest, _ := stateChange.(*ReceiveSecretRequest)
		channelId := paymentState.Initiator.ChannelId
		channelState := channelIdToChannels[channelId]
		subIteration := HandleSecretRequest(paymentState.Initiator, receiveSecretRequest, channelState)
		iteration = ImIterationFromSub(paymentState, subIteration)
	case *ActionCancelRoute:
		actionCancelRoute, _ := stateChange.(*ActionCancelRoute)
		iteration = ImHandleCancelRoute(paymentState, actionCancelRoute, channelIdToChannels, blockNumber)
	case *ReceiveTransferRefundCancelRoute:
		receiveTransferRefundCancelRoute, _ := stateChange.(*ReceiveTransferRefundCancelRoute)
		iteration = ImHandleTransferRefundCancelRoute(paymentState, receiveTransferRefundCancelRoute,
			channelIdToChannels, blockNumber)
	case *ActionCancelPayment:
		channelId := paymentState.Initiator.ChannelId
		channelState := channelIdToChannels[channelId]
		iteration = ImHandleCancelPayment(paymentState, channelState)
	case *ReceiveSecretReveal:
		receiveSecretReveal, _ := stateChange.(*ReceiveSecretReveal)
		iteration = ImHandleOffchainSecretReveal(paymentState, receiveSecretReveal, channelIdToChannels)
	case *ReceiveLockExpired:
		receiveLockExpired, _ := stateChange.(*ReceiveLockExpired)
		iteration = ImHandleLockExpired(paymentState, receiveLockExpired, channelIdToChannels, blockNumber)
	case *ContractReceiveSecretReveal:
		contractReceiveSecretReveal, _ := stateChange.(*ContractReceiveSecretReveal)
		iteration = ImHandleOnChainSecretReveal(paymentState, contractReceiveSecretReveal, channelIdToChannels)
	default:
		log.Warn("[ImStateTransition] unknown stateChange Type: ", reflect.TypeOf(stateChange).String())
	}
	return iteration
}
