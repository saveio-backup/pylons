package transfer

import (
	"crypto/sha256"
	"reflect"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
)

func TgEventsForOnChainSecretReveal(targetState *TargetTransferState, channelState *NettingChannelState,
	blockNumber common.BlockHeight) []Event {

	transfer := targetState.Transfer
	expiration := common.BlockExpiration(transfer.Lock.Expiration)

	safeToWait, _ := MdIsSafeToWait(expiration, channelState.RevealTimeout, blockNumber)

	secretKnownOffChain := IsSecretKnownOffChain(channelState.PartnerState, common.SecretHash(transfer.Lock.SecretHash))

	hasOnchainRevealStarted := (targetState.State == "onchain_secret_reveal")
	if !safeToWait && secretKnownOffChain && !hasOnchainRevealStarted {
		targetState.State = "onchain_secret_reveal"
		secret := GetSecret(channelState.PartnerState, common.SecretHash(transfer.Lock.SecretHash))
		return EventsForOnChainSecretReveal(channelState, secret, expiration)
	}

	return nil
}

func TgHandleInitTarget(stateChange *ActionInitTarget, channelState *NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	var iteration TransitionResult

	transfer := stateChange.Transfer
	route := stateChange.Route

	channelEvents, err := HandleReceiveLockedTransfer(channelState, transfer)
	if err == nil {
		log.Debug("[TgHandleInitTarget] HandleReceiveLockedTransfer err == nil")
		targetState := &TargetTransferState{Route: route, Transfer: transfer}
		safeToWait, _ := MdIsSafeToWait(common.BlockExpiration(transfer.Lock.Expiration),
			channelState.RevealTimeout, blockNumber)
		if safeToWait == true {
			messageIdentifier := GetMsgID()
			recipient := transfer.Initiator
			secretRequest := &SendSecretRequest{
				SendMessageEvent: SendMessageEvent{
					Recipient:         common.Address(recipient),
					ChannelIdentifier: ChannelIdentifierGlobalQueue,
					MessageIdentifier: messageIdentifier,
				},
				PaymentIdentifier: transfer.PaymentIdentifier,
				Amount:            transfer.Lock.Amount,
				Expiration:        common.BlockExpiration(transfer.Lock.Expiration),
				SecretHash:        common.SecretHash(transfer.Lock.SecretHash),
			}
			channelEvents = append(channelEvents, secretRequest)
		}

		iteration = TransitionResult{NewState: targetState, Events: channelEvents}
	} else {
		log.Error("[TgHandleInitTarget] HandleReceiveLockedTransfer Err: ", err.Error())
		unlockFailed := &EventUnlockClaimFailed{
			Identifier: transfer.PaymentIdentifier,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
			Reason:     err.Error()}

		channelEvents = append(channelEvents, unlockFailed)
		iteration = TransitionResult{NewState: nil, Events: channelEvents}
	}

	return &iteration
}

func TgHandleOffChainSecretReveal(targetState *TargetTransferState, stateChange *ReceiveSecretReveal,
	channelState *NettingChannelState, blockNumber common.BlockHeight) *TransitionResult {

	var events []Event
	secretHash := sha256.Sum256(stateChange.Secret)
	validSecret := secretHash == targetState.Transfer.Lock.SecretHash
	hasTransferExpired := TransferExpired(targetState.Transfer, channelState, blockNumber)

	if validSecret && !hasTransferExpired {
		RegisterOffChainSecret(channelState, stateChange.Secret, secretHash)
		route := targetState.Route
		messageIdentifier := GetMsgID()
		targetState.State = "reveal_secret"
		targetState.Secret = &stateChange.Secret
		recipient := route.NodeAddress

		//addr := common2.Address(recipient)
		//fmt.Println("[TgHandleOffChainSecretReveal] recipient: ", addr.ToBase58())

		reveal := &SendSecretReveal{
			SendMessageEvent: SendMessageEvent{
				Recipient:         common.Address(recipient),
				ChannelIdentifier: ChannelIdentifierGlobalQueue,
				MessageIdentifier: messageIdentifier,
			},
			Secret: stateChange.Secret,
		}
		events = append(events, reveal)
		return &TransitionResult{NewState: targetState, Events: events}
	} else {
		return &TransitionResult{NewState: targetState, Events: nil}
	}
}

func TgHandleOnChainSecretReveal(targetState *TargetTransferState,
	stateChange *ContractReceiveSecretReveal, channelState *NettingChannelState) *TransitionResult {

	validSecret := stateChange.SecretHash == common.SecretHash(targetState.Transfer.Lock.SecretHash)

	if validSecret {
		RegisterOnChainSecret(channelState, stateChange.Secret, stateChange.SecretHash,
			stateChange.BlockHeight, true)
		targetState.State = "reveal_secret"
		targetState.Secret = &stateChange.Secret
	}
	return &TransitionResult{NewState: targetState, Events: nil}
}

func TgHandleUnlock(targetState *TargetTransferState, stateChange *ReceiveUnlock,
	channelState *NettingChannelState) *TransitionResult {

	balanceProofSender := stateChange.BalanceProof.Sender
	isValid, events, _ := HandleUnlock(channelState, stateChange)

	if isValid {
		transfer := targetState.Transfer
		paymentReceivedSuccess := &EventPaymentReceivedSuccess{
			PaymentNetworkIdentifier: channelState.PaymentNetworkIdentifier,
			TokenNetworkIdentifier:   channelState.TokenNetworkIdentifier,
			Identifier:               transfer.PaymentIdentifier,
			Amount:                   transfer.Lock.Amount,
			Initiator:                transfer.Initiator,
		}

		unlockSuccess := &EventUnlockClaimSuccess{
			Identifier: transfer.PaymentIdentifier,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
		}

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient:         common.Address(balanceProofSender),
				ChannelIdentifier: ChannelIdentifierGlobalQueue,
				MessageIdentifier: stateChange.MessageIdentifier,
			},
		}

		events = append(events, paymentReceivedSuccess)
		events = append(events, unlockSuccess)
		events = append(events, sendProcessed)
		targetState = nil
	}
	return &TransitionResult{NewState: targetState, Events: events}
}

func TgHandleBlock(targetState *TargetTransferState, channelState *NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {
	//""" After Raiden learns about a new block this function must be called to
	//handle expiration of the hash time lock.
	//"""
	transfer := targetState.Transfer
	var events []Event
	lock := transfer.Lock

	secretKnown := IsSecretKnown(channelState.PartnerState, common.SecretHash(lock.SecretHash))
	lockHasExpired, _ := IsLockExpired(channelState.OurState, lock, blockNumber,
		lock.Expiration+DefaultNumberOfBlockConfirmations)

	if lockHasExpired && targetState.State != "expired" {
		failed := &EventUnlockClaimFailed{
			Identifier: transfer.PaymentIdentifier,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
			Reason:     "lock expired",
		}
		targetState.State = "expired"
		events = append(events, failed)
	} else if secretKnown {
		contractSendSecretRevealEvents := TgEventsForOnChainSecretReveal(targetState, channelState, blockNumber)
		events = append(events, contractSendSecretRevealEvents...)
	}
	return &TransitionResult{NewState: targetState, Events: events}
}

func TgHandleLockExpired(targetState *TargetTransferState, stateChange *ReceiveLockExpired,
	channelState *NettingChannelState, blockNumber common.BlockHeight) *TransitionResult {
	//"""Remove expired locks from channel states."""
	result := handleReceiveLockExpired(channelState, stateChange, blockNumber)

	partnerState := result.NewState.(*NettingChannelState).PartnerState
	if GetLock(partnerState, common.SecretHash(targetState.Transfer.Lock.SecretHash)) != nil {
		transfer := targetState.Transfer
		unlockFailed := &EventUnlockClaimFailed{
			Identifier: transfer.PaymentIdentifier,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
			Reason:     "Lock expired",
		}
		result.Events = append(result.Events, unlockFailed)
		return &TransitionResult{NewState: nil, Events: result.Events}
	}
	return &TransitionResult{NewState: targetState, Events: result.Events}
}

//func StateTransitionForTarget(targetState *TargetTransferState, stateChange StateChange,
//    channelState *NettingChannelState, pseudoRandomGenerator *rand.Rand, blockNumber common.BlockHeight) TransitionResult {
//
//	iteration := TransitionResult{targetState, list.New()}
//
//	switch stateChange.(type) {
//	case *Block:
//		block, _ := stateChange.(*Block)
//		iteration = TargetHandleBlock(targetState, channelState, block.BlockNumber)
//	}
//}

func TgStateTransition(targetState *TargetTransferState, stateChange interface{},
	channelState *NettingChannelState, blockNumber common.BlockHeight) *TransitionResult {
	// State machine for the target node of a mediated transfer.
	// pylint: disable=too-many-branches,unidiomatic-typecheck
	log.Debug("[TgStateTransition] stateChange type: ", reflect.TypeOf(stateChange).String())
	var iteration *TransitionResult
	switch stateChange.(type) {
	case *ActionInitTarget:
		if targetState == nil {
			actionInitTarget, _ := stateChange.(*ActionInitTarget)
			iteration = TgHandleInitTarget(actionInitTarget, channelState, blockNumber)
		}
	case *Block:
		block, _ := stateChange.(*Block)
		iteration = TgHandleBlock(targetState, channelState, block.BlockHeight)
	case *ReceiveSecretReveal:
		receiveSecretReveal, _ := stateChange.(*ReceiveSecretReveal)
		iteration = TgHandleOffChainSecretReveal(targetState, receiveSecretReveal, channelState, blockNumber)
	case *ContractReceiveSecretReveal:
		contractReceiveSecretReveal, _ := stateChange.(*ContractReceiveSecretReveal)
		iteration = TgHandleOnChainSecretReveal(targetState, contractReceiveSecretReveal, channelState)
	case *ReceiveUnlock:
		receiveUnlock, _ := stateChange.(*ReceiveUnlock)
		iteration = TgHandleUnlock(targetState, receiveUnlock, channelState)
	case *ReceiveLockExpired:
		receiveLockExpired, _ := stateChange.(*ReceiveLockExpired)
		iteration = TgHandleLockExpired(targetState, receiveLockExpired, channelState, blockNumber)
	default:
		log.Warn("[TgStateTransition] unknown stateChange type: ", reflect.TypeOf(stateChange).String())
	}
	return iteration
}
