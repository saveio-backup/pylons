package transfer

import (
	"crypto/sha256"
	"reflect"

	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
)

const MaximumPendingTransfers int = 160

func InitEventsForUnlockLock(initiatorState *InitiatorTransferState, channelState *NettingChannelState,
	secret common.Secret, secretHash common.SecretHash) []Event {

	transferDescription := initiatorState.TransferDescription
	messageId := common.GetMsgID()

	unlockLock := SendUnlock(channelState, messageId,
		transferDescription.PaymentId, secret, secretHash)

	paymentSentSuccess := &EventPaymentSentSuccess{
		PaymentNetworkId: channelState.PaymentNetworkId,
		TokenNetworkId:   channelState.TokenNetworkId,
		Identifier:       transferDescription.PaymentId,
		Amount:           transferDescription.Amount,
		Target:           common.Address(transferDescription.Target),
	}

	unlockSuccess := &EventUnlockSuccess{
		Identifier: transferDescription.PaymentId,
		SecretHash: transferDescription.SecretHash,
	}

	var result []Event
	result = append(result, unlockLock)
	result = append(result, paymentSentSuccess)
	result = append(result, unlockSuccess)

	return result

}

func InitHandleBlock(initiatorState *InitiatorTransferState, stateChange *Block,
	channelState *NettingChannelState) *TransitionResult {

	secretHash := common.SecretHash(initiatorState.Transfer.Lock.SecretHash)
	lockedLock, exist1 := channelState.OurState.SecretHashesToLockedLocks[secretHash]
	if exist1 == false {
		_, exist2 := channelState.PartnerState.SecretHashesToLockedLocks[secretHash]
		if exist2 {
			return &TransitionResult{NewState: initiatorState, Events: nil}
		} else {
			return &TransitionResult{NewState: nil, Events: nil}
		}
	}

	lockExpirationThreshold := lockedLock.Expiration + common.BlockHeight(common.Config.ConfirmBlockCount)*2
	lockHasExpired, _ := IsLockExpired(channelState.OurState, lockedLock, stateChange.BlockHeight, lockExpirationThreshold)

	if lockHasExpired {
		log.Debugf("expiration: %d, blockHeight: %d, confirmBlockCount: %d", lockedLock.Expiration, stateChange.BlockHeight, common.Config.ConfirmBlockCount)
		expiredLockEvents := EventsForExpiredLock(channelState, lockedLock)
		transferDescription := initiatorState.TransferDescription
		// TODO: When we introduce multiple transfers per payment this needs to be
		//       reconsidered. As we would want to try other routes once a route
		//       has failed, and a transfer failing does not mean the entire payment
		//       would have to fail.
		//       Related issue: https://github.com/raiden-network/raiden/issues/2329
		transferFailed := &EventPaymentSentFailed{
			PaymentNetworkId: transferDescription.PaymentNetworkId,
			TokenNetworkId:   transferDescription.TokenNetworkId,
			Identifier:       transferDescription.PaymentId,
			Target:           common.Address(transferDescription.Target),
			Reason:           "transfer's lock has expired",
		}
		expiredLockEvents = append(expiredLockEvents, transferFailed)
		lockExists := LockExistsInEitherChannelSide(channelState, secretHash)
		if lockExists {
			return &TransitionResult{NewState: initiatorState, Events: expiredLockEvents}
		} else {
			return &TransitionResult{NewState: nil, Events: expiredLockEvents}
		}
	}
	return &TransitionResult{NewState: initiatorState, Events: nil}
}

func InitGetInitialLockExpiration(blockNumber common.BlockHeight, revealTimeout common.BlockTimeout) common.BlockExpiration {
	return common.BlockExpiration(blockNumber) + common.BlockExpiration(revealTimeout*2)
}

func InitNextChannelFromRoutes(availableRoutes []RouteState,
	channelIdToChannels map[common.ChannelID]*NettingChannelState,
	transferAmount common.TokenAmount) *NettingChannelState {
	log.Debug("[InitNextChannelFromRoutes]")
	for i := 0; i < len(availableRoutes); i++ {
		channelId := availableRoutes[i].ChannelId
		channelState := channelIdToChannels[channelId]
		if channelState == nil || GetStatus(channelState) != ChannelStateOpened {
			continue
		}
		if channelState.OurState.MerkleTree.Layers == nil {
			log.Debug("[InitNextChannelFromRoutes] channelState.OurState.MerkleTree.Layers == nil")
		}
		pendingTransfers := getNumberOfPendingTransfers(channelState.OurState)
		log.Debug("[InitNextChannelFromRoutes] pendingTransfers: ", pendingTransfers)
		if pendingTransfers >= MaximumPendingTransfers {
			continue
		}

		distributable := GetDistributable(channelState.OurState, channelState.PartnerState)
		if transferAmount > distributable {
			continue
		}

		if IsValidAmount(channelState.OurState, transferAmount) {
			return channelState
		}
	}

	return nil
}

func InitTryNewRoute(oldInitiatorState *InitiatorTransferState,
	channelIdToChannels map[common.ChannelID]*NettingChannelState,
	availableRoutes []RouteState, transferDescription *TransferDescriptionWithSecretState,
	blockNumber common.BlockHeight, init bool) *TransitionResult {

	var channelState *NettingChannelState
	// FIXME: add new route to task
	if init {
		channelState = InitNextChannelFromRoutes(availableRoutes, channelIdToChannels, transferDescription.Amount)
	}
	var events []Event
	var initiatorState *InitiatorTransferState

	if channelState == nil {
		var reason string

		if len(availableRoutes) == 0 {
			reason = "there is no route available"
		} else {
			reason = "none of the available routes could be used"
		}

		transferFailed := &EventPaymentSentFailed{
			PaymentNetworkId: transferDescription.PaymentNetworkId,
			TokenNetworkId:   transferDescription.TokenNetworkId,
			Identifier:       transferDescription.PaymentId,
			Target:           common.Address(transferDescription.Target),
			Reason:           reason,
		}
		events = append(events, transferFailed)
		initiatorState = oldInitiatorState
	} else {
		messageId := common.GetMsgID()
		lockedTransferEvent := InitSendLockedTransfer(transferDescription, channelState,
			messageId, blockNumber)

		initiatorState = &InitiatorTransferState{
			TransferDescription: transferDescription,
			ChannelId:           channelState.Identifier,
			Transfer:            lockedTransferEvent.Transfer,
		}
		events = append(events, lockedTransferEvent)
	}

	return &TransitionResult{NewState: initiatorState, Events: events}
}

func InitSendLockedTransfer(transferDescription *TransferDescriptionWithSecretState,
	channelState *NettingChannelState, messageId common.MessageID,
	blockNumber common.BlockHeight) *SendLockedTransfer {

	lockExpiration := InitGetInitialLockExpiration(blockNumber, channelState.RevealTimeout)

	lockedTransferEvent := sendLockedTransfer(channelState, transferDescription.Initiator,
		common.Address(transferDescription.Target), common.PaymentAmount(transferDescription.Amount),
		messageId, transferDescription.PaymentId, lockExpiration, transferDescription.EncSecret,
		transferDescription.SecretHash)

	return lockedTransferEvent
}

/*
func InitSendLockedTransfer(transferDescription *TransferDescriptionWithSecretState,
	channelState *NettingChannelState, messageId common.MessageID,
	blockNumber common.BlockHeight) *SendLockedTransfer {

	//[TODO] implement channel.SendLockedTransfer, change name to channel.ChannelSendLockedTransfer

	lockExpiration := InitGetInitialLockExpiration(blockNumber, channelState.RevealTimeout)

	lockedTransferEvent := SendLockedTransfer {
		SendMessageEvent: SendMessageEvent{
			Recipient: common.Address(channelState.PartnerState.Address),
			ChannelId: channelState.Identifier,
			MessageId: messageId,
		},
		Transfer: LockedTransferUnsignedState{
			PaymentId:transferDescription.PaymentId,
			Token: channelState.TokenAddress,
			BalanceProof: transferDescription,
			Lock: transferDescription,
			Initiator: transferDescription.Initiator,
			Target: transferDescription.Target,
		},

		channelState,
		transferDescription.Initiator,
		transferDescription.Target,
		transferDescription.Amount,


		lockExpiration,
		transferDescription.SecretHash,
	}


	return lockedTransferEvent
}
*/
func HandleSecretRequest(initiatorState *InitiatorTransferState, stateChange *ReceiveSecretRequest,
	channelState *NettingChannelState) *TransitionResult {
	var iteration *TransitionResult

	isMessageFromTarget := false
	if stateChange.Sender == initiatorState.TransferDescription.Target &&
		stateChange.SecretHash == initiatorState.TransferDescription.SecretHash &&
		stateChange.PaymentId == initiatorState.TransferDescription.PaymentId {
		isMessageFromTarget = true
	}

	lock := GetLock(channelState.OurState, initiatorState.TransferDescription.SecretHash)
	alreadyReceivedSecretRequest := initiatorState.ReceivedSecretRequest

	isValidSecretRequest := false
	if common.TokenAmount(stateChange.Amount) == initiatorState.TransferDescription.Amount &&
		stateChange.Expiration == common.BlockExpiration(lock.Expiration) {
		isValidSecretRequest = true
	}

	if alreadyReceivedSecretRequest == true && isMessageFromTarget == true {
		log.Warn("[HandleSecretRequest] alreadyReceivedSecretRequest == true && isMessageFromTarget == true")
		iteration = &TransitionResult{NewState: initiatorState, Events: nil}
	} else if isValidSecretRequest == true && isMessageFromTarget == true {
		log.Debug("[HandleSecretRequest] isValidSecretRequest == true && isMessageFromTarget == true")
		messageId := common.GetMsgID()
		transferDescription := initiatorState.TransferDescription
		recipient := transferDescription.Target
		revealSecret := &SendSecretReveal{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(recipient),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: messageId,
			},
			Secret: transferDescription.Secret,
		}
		revealSecret.SecretHash = sha256.Sum256(transferDescription.Secret)

		initiatorState.RevealSecret = revealSecret
		initiatorState.ReceivedSecretRequest = true

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(stateChange.Sender),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}

		var events []Event
		events = append(events, revealSecret)
		events = append(events, sendProcessed)
		iteration = &TransitionResult{NewState: initiatorState, Events: events}

	} else if isValidSecretRequest == false && isMessageFromTarget {
		log.Warn("[HandleSecretRequest] isValidSecretRequest == false && isMessageFromTarget")
		cancel := &EventPaymentSentFailed{
			PaymentNetworkId: channelState.PaymentNetworkId,
			TokenNetworkId:   channelState.TokenNetworkId,
			Identifier:       initiatorState.TransferDescription.PaymentId,
			Target:           common.Address(initiatorState.TransferDescription.Target),
			Reason:           string("bad secret request message from target")}

		initiatorState.ReceivedSecretRequest = true
		var events []Event
		events = append(events, cancel)
		iteration = &TransitionResult{NewState: initiatorState, Events: events}
	} else {
		log.Debug("[HandleSecretRequest] else")
		iteration = &TransitionResult{NewState: initiatorState, Events: nil}
	}

	log.Debug("[HandleSecretRequest] event Num: ", len(iteration.Events))
	for _, e := range iteration.Events {
		log.Debug("[HandleSecretRequest] e Type:  ", reflect.TypeOf(e).String())
	}
	return iteration
}

func InitHandleOffChainSecretReveal(initiatorState *InitiatorTransferState,
	stateChange *ReceiveSecretReveal, channelState *NettingChannelState) *TransitionResult {
	var iteration *TransitionResult

	//validReveal := false
	//if stateChange.Sender == channelState.PartnerState.Address &&
	//    stateChange.SecretHash == initiatorState.TransferDescription.SecretHash {
	//	isValidSecretReveal = true
	//}

	//var emptyHash common.Secret
	//if stateChange.Secret != emptyHash && secretHash == initiatorState.TransferDescription.SecretHash {
	//	validReveal = true
	//}

	secretHash := sha256.Sum256(stateChange.Secret)
	validReveal := secretHash == initiatorState.TransferDescription.SecretHash
	sentByPartner := stateChange.Sender == channelState.PartnerState.Address
	isChannelOpen := GetStatus(channelState) == ChannelStateOpened

	if validReveal && isChannelOpen && sentByPartner {
		events := InitEventsForUnlockLock(initiatorState, channelState, stateChange.Secret, secretHash)

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(stateChange.Sender),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}
		events = append(events, sendProcessed)
		iteration = &TransitionResult{NewState: nil, Events: events}
	} else {
		iteration = &TransitionResult{NewState: initiatorState, Events: nil}
	}
	return iteration
}

func InitHandleOnChainSecretReveal(initiatorState *InitiatorTransferState,
	stateChange *ContractReceiveSecretReveal, channelState *NettingChannelState) *TransitionResult {

	isValidSecret := stateChange.SecretHash == common.SecretHash(initiatorState.Transfer.Lock.SecretHash)
	isChannelOpen := GetStatus(channelState) == ChannelStateOpened
	isLockExpired := stateChange.BlockHeight > initiatorState.Transfer.Lock.Expiration

	var events []Event
	var iteration *TransitionResult
	if isValidSecret && isChannelOpen && !isLockExpired {
		events = InitEventsForUnlockLock(initiatorState, channelState, stateChange.Secret,
			stateChange.SecretHash)

		iteration = &TransitionResult{NewState: nil, Events: events}
	} else {
		iteration = &TransitionResult{NewState: initiatorState, Events: events}
	}

	return iteration
}
