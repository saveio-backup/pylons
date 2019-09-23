package transfer

import (
	"crypto/sha256"
	"reflect"

	"fmt"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network/secretcrypt"
	"github.com/saveio/themis/common/log"
)

func TgEventsForOnChainSecretReveal(targetState *TargetTransferState, channelState *NettingChannelState,
	blockNumber common.BlockHeight) []Event {

	transfer := targetState.Transfer
	expiration := common.BlockExpiration(transfer.Lock.Expiration)

	safeToWait, _ := MdIsSafeToWait(expiration, channelState.RevealTimeout, blockNumber)

	secretKnownOffChain := IsSecretKnownOffChain(channelState.PartnerState, common.SecretHash(transfer.Lock.SecretHash))

	hasOnchainRevealStarted := targetState.State == "onchain_secret_reveal"
	if !safeToWait && secretKnownOffChain && !hasOnchainRevealStarted {
		targetState.State = "onchain_secret_reveal"
		secret := GetSecret(channelState.PartnerState, common.SecretHash(transfer.Lock.SecretHash))
		return EventsForOnChainSecretReveal(channelState, secret, expiration)
	}

	return nil
}

func HandleInitTarget(stateChange *ActionInitTarget, channelState *NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {
	//	if safeToWait == true {
	//		messageId := common.GetMsgID()
	//		recipient := transfer.Initiator
	//		secretRequest := &SendSecretRequest{
	//			SendMessageEvent: SendMessageEvent{
	//				Recipient:         common.Address(recipient),
	//				ChannelId: ChannelIdGlobalQueue,
	//				MessageId: messageId,
	//			},
	//			PaymentId: transfer.PaymentId,
	//			Amount:            transfer.Lock.Amount,
	//			Expiration:        common.BlockExpiration(transfer.Lock.Expiration),
	//			SecretHash:        common.SecretHash(transfer.Lock.SecretHash),
	//		}
	//		channelEvents = append(channelEvents, secretRequest)
	//	}
	//
	//	iteration = TransitionResult{NewState: targetState, Events: channelEvents}

	var iteration TransitionResult
	transfer := stateChange.Transfer
	transferRoute := stateChange.Route

	channelEvents, err := HandleReceiveLockedTransfer(channelState, transfer)
	if err == nil {
		log.Debug("[HandleInitTarget] HandleReceiveLockedTransfer err == nil")
		targetState := &TargetTransferState{Route: transferRoute, Transfer: transfer}
		safeToWait, _ := MdIsSafeToWait(common.BlockExpiration(transfer.Lock.Expiration),
			channelState.RevealTimeout, blockNumber)
		if safeToWait == true {
			secret, err := secretcrypt.SecretCryptService.DecryptSecret(stateChange.Transfer.EncSecret)
			if err == nil {
				secretHash := common.GetHash(secret)
				if common.Keccak256(secretHash) == targetState.Transfer.Lock.SecretHash {
					if !TransferExpired(targetState.Transfer, channelState, blockNumber) {
						RegisterOffChainSecret(channelState, secret, secretHash)

						targetState.State = "reveal_secret"
						comSecret := common.Secret(secret)
						targetState.Secret = &comSecret

						reveal := &SendSecretReveal{
							SendMessageEvent: SendMessageEvent{
								Recipient: targetState.Route.NodeAddress,
								ChannelId: targetState.Route.ChannelId,
								MessageId: common.GetMsgID(),
							},
							Secret: comSecret,
						}
						channelEvents = append(channelEvents, reveal)
						return &TransitionResult{NewState: targetState, Events: channelEvents}
					} else {
						err = fmt.Errorf("[TgHandleInitTarget] transfer is expired")
					}
				} else {
					err = fmt.Errorf("[TgHandleInitTarget] Secret is invalid")
				}
			}
		} else {
			err = fmt.Errorf("[TgHandleInitTarget] not safe to wait ")
		}
	}

	log.Error("[TgHandleInitTarget] HandleReceiveLockedTransfer Err: ", err.Error())
	unlockFailed := &EventUnlockClaimFailed{
		Identifier: transfer.PaymentId,
		SecretHash: common.SecretHash(transfer.Lock.SecretHash),
		Reason:     err.Error(),
	}

	channelEvents = append(channelEvents, unlockFailed)
	iteration = TransitionResult{NewState: nil, Events: channelEvents}
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

		targetState.State = "reveal_secret"
		targetState.Secret = &stateChange.Secret

		//addr := common2.Address(recipient)
		//fmt.Println("[TgHandleOffChainSecretReveal] recipient: ", addr.ToBase58())

		messageId := common.GetMsgID()
		reveal := &SendSecretReveal{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(route.NodeAddress),
				//check route channelId
				ChannelId: route.ChannelId,
				MessageId: messageId,
			},
			Secret: stateChange.Secret,
		}
		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(stateChange.Sender),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}
		recipient := common.ToBase58(route.NodeAddress)
		sender := common.ToBase58(stateChange.Sender)
		log.Debugf("recipient: %s route.channelId :%d    sender:%s", recipient, route.ChannelId, sender)
		events = append(events, sendProcessed)
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

	//balanceProofSender := stateChange.BalanceProof.Sender
	isValid, events, _ := HandleUnlock(channelState, stateChange)

	if isValid {
		transfer := targetState.Transfer
		paymentReceivedSuccess := &EventPaymentReceivedSuccess{
			PaymentNetworkId: channelState.PaymentNetworkId,
			TokenNetworkId:   channelState.TokenNetworkId,
			Identifier:       transfer.PaymentId,
			Amount:           transfer.Lock.Amount,
			Initiator:        transfer.Initiator,
		}

		unlockSuccess := &EventUnlockClaimSuccess{
			Identifier: transfer.PaymentId,
			SecretHash: common.SecretHash(transfer.Lock.SecretHash),
		}

		events = append(events, paymentReceivedSuccess)
		events = append(events, unlockSuccess)
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
			Identifier: transfer.PaymentId,
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
			Identifier: transfer.PaymentId,
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
			iteration = HandleInitTarget(actionInitTarget, channelState, blockNumber)
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
