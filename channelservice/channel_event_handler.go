package channelservice

import (
	"reflect"

	"github.com/oniio/oniChain-go-sdk/usdt"
	sdkutils "github.com/oniio/oniChain-go-sdk/utils"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/crypto/keypair"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/storage"
	"github.com/saveio/pylons/transfer"
)

type ChannelEventHandler struct {
}

func (self ChannelEventHandler) OnChannelEvent(channel *ChannelService, event transfer.Event) {
	log.Debug("[OnChannelEvent]  type: ", reflect.TypeOf(event).String())
	switch event.(type) {
	case *transfer.SendDirectTransfer:
		sendDirectTransfer := event.(*transfer.SendDirectTransfer)
		self.HandleSendDirectTransfer(channel, sendDirectTransfer)
	case *transfer.SendProcessed:
		sendProcessed := event.(*transfer.SendProcessed)
		self.HandleSendProcessed(channel, sendProcessed)
	case *transfer.EventPaymentSentSuccess:
		eventPaymentSentSuccess := event.(*transfer.EventPaymentSentSuccess)
		self.HandlePaymentSentSuccess(channel, eventPaymentSentSuccess)
	case *transfer.EventPaymentSentFailed:
		eventPaymentSentFailed := event.(*transfer.EventPaymentSentFailed)
		self.HandlePaymentSentFailed(channel, eventPaymentSentFailed)
	case *transfer.EventPaymentReceivedSuccess:
		eventPaymentReceivedSuccess := event.(*transfer.EventPaymentReceivedSuccess)
		self.HandlePaymentReceivedSuccess(channel, eventPaymentReceivedSuccess)
	case *transfer.ContractSendChannelClose:
		contractSendChannelClose := event.(*transfer.ContractSendChannelClose)
		self.HandleContractSendChannelClose(channel, contractSendChannelClose)
	case *transfer.ContractSendChannelUpdateTransfer:
		contractSendChannelUpdateTransfer := event.(*transfer.ContractSendChannelUpdateTransfer)
		self.HandelContractSendChannelUpdate(channel, contractSendChannelUpdateTransfer)
	case *transfer.ContractSendChannelSettle:
		contractSendChannelSettle := event.(*transfer.ContractSendChannelSettle)
		self.HandleContractSendChannelSettle(channel, contractSendChannelSettle)
	case *transfer.SendLockedTransfer:
		sendLockedTransfer := event.(*transfer.SendLockedTransfer)
		self.HandleSendLockedTransfer(channel, sendLockedTransfer)
	case *transfer.SendSecretReveal:
		sendSecretReveal := event.(*transfer.SendSecretReveal)
		self.HandleSendSecretReveal(channel, sendSecretReveal)
	case *transfer.SendBalanceProof:
		sendBalanceProof := event.(*transfer.SendBalanceProof)
		self.HandleSendBalanceProof(channel, sendBalanceProof)
	case *transfer.SendSecretRequest:
		sendSecretRequest := event.(*transfer.SendSecretRequest)
		self.HandleSendSecretRequest(channel, sendSecretRequest)
	case *transfer.SendRefundTransfer:
		sendRefundTransfer := event.(*transfer.SendRefundTransfer)
		self.HandleSendRefundTransfer(channel, sendRefundTransfer)
	case *transfer.ContractSendSecretReveal:
		contractSendSecretReveal := event.(*transfer.ContractSendSecretReveal)
		self.HandleContractSendSecretReveal(channel, contractSendSecretReveal)
	case *transfer.EventUnlockClaimSuccess:
		unlockClaimSuccess := event.(*transfer.EventUnlockClaimSuccess)
		self.HandleEventUnlockClaimSuccess(channel, unlockClaimSuccess)
	case *transfer.EventUnlockClaimFailed:
		unlockClaimFailed := event.(*transfer.EventUnlockClaimFailed)
		self.HandleEventUnlockClaimFailed(channel, unlockClaimFailed)
	case *transfer.EventUnlockSuccess:
		unlockSuccess := event.(*transfer.EventUnlockSuccess)
		self.HandleEventUnlockSuccess(channel, unlockSuccess)
	case *transfer.EventUnlockFailed:
		unlockClaimFailed := event.(*transfer.EventUnlockFailed)
		self.HandleEventUnlockFailed(channel, unlockClaimFailed)
	case *transfer.SendWithdrawRequest:
		sendWithdrawRequest := event.(*transfer.SendWithdrawRequest)
		self.HandleSendWithdrawRequest(channel, sendWithdrawRequest)
	case *transfer.SendWithdraw:
		sendWithdraw := event.(*transfer.SendWithdraw)
		self.HandleSendWithdraw(channel, sendWithdraw)
	case *transfer.ContractSendChannelWithdraw:
		contractSendChannelWithdraw := event.(*transfer.ContractSendChannelWithdraw)
		self.HandleContractSendChannelWithdraw(channel, contractSendChannelWithdraw)
	case *transfer.EventWithdrawRequestSentFailed:
		eventWithdrawRequestSentFailed := event.(*transfer.EventWithdrawRequestSentFailed)
		self.HandleWithdrawRequestSentFailed(channel, eventWithdrawRequestSentFailed)
	case *transfer.EventInvalidReceivedWithdraw:
		eventInvalidReceivedWithdraw := event.(*transfer.EventInvalidReceivedWithdraw)
		self.HandleInvalidReceivedWithdraw(channel, eventInvalidReceivedWithdraw)
	case *transfer.SendCooperativeSettleRequest:
		sendCooperativeSettleRequest := event.(*transfer.SendCooperativeSettleRequest)
		self.HandleSendCooperativeSettleRequest(channel, sendCooperativeSettleRequest)
	case *transfer.SendCooperativeSettle:
		sendCooperativeSettle := event.(*transfer.SendCooperativeSettle)
		self.HandleSendCooperativeSettle(channel, sendCooperativeSettle)
	case *transfer.ContractSendChannelCooperativeSettle:
		contractSendChannelCooperativeSettle := event.(*transfer.ContractSendChannelCooperativeSettle)
		self.HandleContractSendChannelCooperativeSettle(channel, contractSendChannelCooperativeSettle)
	case *transfer.ContractSendChannelBatchUnlock:
		contractSendChannelBatchUnlock := event.(*transfer.ContractSendChannelBatchUnlock)
		self.HandleContractSendChannelUnlock(channel, contractSendChannelBatchUnlock)
	default:
		log.Warn("[OnChannelEvent] Not known type: ", reflect.TypeOf(event).String())
	}
	return
}

func (self ChannelEventHandler) HandleSendDirectTransfer(channel *ChannelService, sendDirectTransfer *transfer.SendDirectTransfer) {
	message := messages.MessageFromSendEvent(sendDirectTransfer)
	if message != nil {
		err := channel.Sign(message)
		if err != nil {
			return
		}

		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(sendDirectTransfer.Recipient),
			ChannelIdentifier: sendDirectTransfer.ChannelIdentifier,
		}

		ret := channel.channelActor.Transport.SendAsync(queueId, message)
		if ret != nil {
			log.Error("send msg failed:", ret)
		}
	}

	return
}

func (self ChannelEventHandler) HandleSendProcessed(channel *ChannelService, processedEvent *transfer.SendProcessed) {
	message := messages.MessageFromSendEvent(processedEvent)
	if message != nil {
		err := channel.Sign(message)
		if err != nil {
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(processedEvent.Recipient),
			ChannelIdentifier: processedEvent.ChannelIdentifier,
		}

		channel.channelActor.Transport.SendAsync(queueId, message)
	} else {
		log.Warn("[HandleSendProcessed] Message is nil")
	}

	return
}

func (self ChannelEventHandler) HandlePaymentSentSuccess(channel *ChannelService, paymentSentSuccessEvent *transfer.EventPaymentSentSuccess) {
	target := common.Address(paymentSentSuccessEvent.Target)
	identifier := paymentSentSuccessEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		panic("error in HandlePaymentSentSuccess, no payment status found in the map")
	}

	channel.RemovePaymentStatus(target, identifier)
	log.Debug("set paymentDone to true")
	paymentStatus.paymentDone <- true

	return
}

func (self ChannelEventHandler) HandlePaymentSentFailed(channel *ChannelService, paymentSentFailedEvent *transfer.EventPaymentSentFailed) {
	target := common.Address(paymentSentFailedEvent.Target)
	identifier := paymentSentFailedEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		panic("error in HandlePaymentSentFailed, no payment status found in the map")
	}

	channel.RemovePaymentStatus(target, identifier)
	log.Warn("set paymentDone to false")
	paymentStatus.paymentDone <- false

	return
}

func (self ChannelEventHandler) HandlePaymentReceivedSuccess(channel *ChannelService, paymentReceivedSuccessEvent *transfer.EventPaymentReceivedSuccess) {
	for ch := range channel.ReceiveNotificationChannels {
		ch <- paymentReceivedSuccessEvent
	}
}

func (self ChannelEventHandler) HandleContractSendChannelClose(channel *ChannelService, channelCloseEvent *transfer.ContractSendChannelClose) {
	var nonce common.Nonce
	var balanceHash common.BalanceHash
	var signature common.Signature
	var messageHash common.Keccak256
	var publicKey common.PubKey

	balanceProof := channelCloseEvent.BalanceProof

	if balanceProof != nil {
		balanceHash = transfer.HashBalanceData(
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
		)

		log.Debugf("[HandleContractSendChannelClose] ta : %d, la : %d, lr : %v, balanceHash : %v",
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
			balanceHash,
		)

		nonce = balanceProof.Nonce
		signature = balanceProof.Signature
		messageHash = balanceProof.MessageHash
		publicKey = balanceProof.PublicKey
	}

	args := channel.GetPaymentChannelArgs(channelCloseEvent.TokenNetworkIdentifier, channelCloseEvent.ChannelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelCloseEvent.TokenNetworkIdentifier), channelCloseEvent.ChannelIdentifier, args)

	channelProxy.Close(nonce, balanceHash, common.AdditionalHash(messageHash[:]), signature, publicKey)
}

func (self ChannelEventHandler) HandelContractSendChannelUpdate(channel *ChannelService, channelUpdateEvent *transfer.ContractSendChannelUpdateTransfer) {
	balanceProof := channelUpdateEvent.BalanceProof

	if balanceProof != nil {
		args := channel.GetPaymentChannelArgs(channelUpdateEvent.TokenNetworkIdentifier, channelUpdateEvent.ChannelIdentifier)
		if args == nil {
			panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
		}

		channelProxy := channel.chain.PaymentChannel(common.Address(channelUpdateEvent.TokenNetworkIdentifier), channelUpdateEvent.ChannelIdentifier, args)

		balanceHash := transfer.HashBalanceData(
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
		)

		log.Debugf("[HandelContractSendChannelUpdate] ta : %d, la : %d, lr : %v, balanceHash : %v",
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
			balanceHash,
		)

		nonClosingData := transfer.PackBalanceProofUpdate(
			balanceProof.Nonce, balanceHash, common.AdditionalHash(balanceProof.MessageHash[:]),
			balanceProof.ChannelIdentifier, common.TokenNetworkAddress(balanceProof.TokenNetworkIdentifier),
			balanceProof.ChainId, balanceProof.Signature)

		var ourSignature common.Signature
		var nonClosePubkey common.PubKey

		ourSignature, err := sdkutils.Sign(channel.Account, nonClosingData)
		if err != nil {
			return
		}

		nonClosePubkey = keypair.SerializePublicKey(channel.Account.PubKey())

		channelProxy.UpdateTransfer(
			balanceProof.Nonce, balanceHash, common.AdditionalHash(balanceProof.MessageHash[:]),
			balanceProof.Signature, ourSignature,
			balanceProof.PublicKey, nonClosePubkey)
	}
}

func (self ChannelEventHandler) HandleContractSendChannelSettle(channel *ChannelService, channelSettleEvent *transfer.ContractSendChannelSettle) {
	var ourTransferredAmount common.TokenAmount
	var ourLockedAmount common.TokenAmount
	var ourLocksroot common.Locksroot
	var partnerTransferredAmount common.TokenAmount
	var partnerLockedAmount common.TokenAmount
	var partnerLocksroot common.Locksroot
	var ourBalanceProof *transfer.BalanceProofUnsignedState
	var partnerBalanceProof *transfer.BalanceProofSignedState

	var chainID common.ChainID

	args := channel.GetPaymentChannelArgs(common.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier), channelSettleEvent.ChannelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}
	channelProxy := channel.chain.PaymentChannel(common.Address(channelSettleEvent.TokenNetworkIdentifier), channelSettleEvent.ChannelIdentifier, args)

	participanatsDetails := channelProxy.TokenNetwork.DetailParticipants(channelProxy.Participant1, channelProxy.Participant2, channelSettleEvent.ChannelIdentifier)

	// when ourDetails or PartnerDetails is nil, it means the channel has been settled already
	if participanatsDetails.OurDetails == nil || participanatsDetails.PartnerDetails == nil {
		return
	}

	ourBalanceHash := participanatsDetails.OurDetails.BalanceHash
	log.Debugf("[ChannelSettle] ourBalanceHash %v", ourBalanceHash)
	if !common.IsEmptyBalanceHash(ourBalanceHash) {
		ourBalanceProof = storage.GetLatestKnownBalanceProofFromEvents(
			channel.Wal.Storage, chainID, common.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier),
			channelSettleEvent.ChannelIdentifier, ourBalanceHash)
	}

	if ourBalanceProof != nil {
		ourTransferredAmount = ourBalanceProof.TransferredAmount
		ourLockedAmount = ourBalanceProof.LockedAmount
		ourLocksroot = ourBalanceProof.LocksRoot
	}

	partnerBalanceHash := participanatsDetails.PartnerDetails.BalanceHash
	log.Debugf("[ChannelSettle] partnerBalanceHash %v", partnerBalanceHash)
	if !common.IsEmptyBalanceHash(partnerBalanceHash) {
		partnerBalanceProof = storage.GetLatestKnownBalanceProofFromStateChanges(
			channel.Wal.Storage, chainID, common.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier),
			channelSettleEvent.ChannelIdentifier, partnerBalanceHash, participanatsDetails.PartnerDetails.Address)
	}

	if partnerBalanceProof != nil {
		partnerTransferredAmount = partnerBalanceProof.TransferredAmount
		partnerLockedAmount = partnerBalanceProof.LockedAmount
		partnerLocksroot = partnerBalanceProof.LocksRoot
	}

	log.Debugf("[ChannelSettle] ourBP : %v, partnerBP : %v", ourBalanceProof, partnerBalanceProof)
	log.Debugf("[ChannelSettle] our: tA = %d, lA : %d, locksroot : %v, partner: tA : %d, lA : %d, locksroot : %v ", ourTransferredAmount, ourLockedAmount, ourLocksroot,
		partnerTransferredAmount, partnerLockedAmount, partnerLocksroot)
	channelProxy.Settle(
		ourTransferredAmount, ourLockedAmount, ourLocksroot,
		partnerTransferredAmount, partnerLockedAmount, partnerLocksroot)
}

func (self ChannelEventHandler) HandleSendLockedTransfer(channel *ChannelService, sendLockedTransfer *transfer.SendLockedTransfer) {
	mediatedTransferMessage := messages.MessageFromSendEvent(sendLockedTransfer)
	if mediatedTransferMessage != nil {
		err := channel.Sign(mediatedTransferMessage)
		if err != nil {
			log.Error("[HandleSendLockedTransfer] ", err.Error())
			return
		}

		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(sendLockedTransfer.Recipient),
			ChannelIdentifier: sendLockedTransfer.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, mediatedTransferMessage)
	} else {
		log.Warn("[HandleSendLockedTransfer] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendSecretReveal(channel *ChannelService, revealSecretEvent *transfer.SendSecretReveal) {
	revealSecretMessage := messages.MessageFromSendEvent(revealSecretEvent)
	if revealSecretMessage != nil {
		err := channel.Sign(revealSecretMessage)
		if err != nil {
			log.Error("[HandleSendSecretReveal] ", err.Error())
			return
		}

		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(revealSecretEvent.Recipient),
			ChannelIdentifier: revealSecretEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, revealSecretMessage)
	} else {
		log.Warn("[HandleSendSecretReveal] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendBalanceProof(channel *ChannelService, balanceProofEvent *transfer.SendBalanceProof) {
	unlockMessage := messages.MessageFromSendEvent(balanceProofEvent)
	if unlockMessage != nil {
		err := channel.Sign(unlockMessage)
		if err != nil {
			log.Error("[HandleSendBalanceProof] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(balanceProofEvent.Recipient),
			ChannelIdentifier: balanceProofEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, unlockMessage)
	} else {
		log.Warn("[HandleSendBalanceProof] Message is nil")
	}
	return

}

func (self ChannelEventHandler) HandleSendSecretRequest(channel *ChannelService, secretRequestEvent *transfer.SendSecretRequest) {
	secretRequestMessage := messages.MessageFromSendEvent(secretRequestEvent)
	if secretRequestMessage != nil {
		err := channel.Sign(secretRequestMessage)
		if err != nil {
			log.Error("[HandleSendSecretRequest] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(secretRequestEvent.Recipient),
			ChannelIdentifier: secretRequestEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, secretRequestMessage)
	} else {
		log.Warn("[HandleSendSecretRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendRefundTransfer(channel *ChannelService, refundTransferEvent *transfer.SendRefundTransfer) {
	refundTransferMessage := messages.MessageFromSendEvent(refundTransferEvent)
	if refundTransferMessage != nil {
		err := channel.Sign(refundTransferMessage)
		if err != nil {
			log.Error("[HandleSendRefundTransfer] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(refundTransferEvent.Recipient),
			ChannelIdentifier: refundTransferEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, refundTransferMessage)
	} else {
		log.Warn("[HandleSendRefundTransfer] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleContractSendSecretReveal(channel *ChannelService, sendSecretRevealEvent *transfer.ContractSendSecretReveal) {
	secretRegistry := channel.chain.SecretRegistry(common.SecretRegistryAddress(usdt.USDT_CONTRACT_ADDRESS))
	// register secret in go routinue to avoid block other register
	go secretRegistry.RegisterSecret(sendSecretRevealEvent.Secret)
}

func (self ChannelEventHandler) HandleEventUnlockClaimSuccess(channel *ChannelService,
	unlockClaimSuccess *transfer.EventUnlockClaimSuccess) {
	log.Debug("[OnChannelEvent] Unlock Claim Success PaymentId: ", unlockClaimSuccess.Identifier)
}

func (self ChannelEventHandler) HandleEventUnlockClaimFailed(channel *ChannelService,
	unlockClaimFailed *transfer.EventUnlockClaimFailed) {
	log.Debug("[OnChannelEvent] Unlock Claim Failed PaymentId: ", unlockClaimFailed.Identifier)
}

func (self ChannelEventHandler) HandleEventUnlockSuccess(channel *ChannelService,
	unlockSuccess *transfer.EventUnlockSuccess) {
	log.Debug("[OnChannelEvent] Unlock Success PaymentId: ", unlockSuccess.Identifier)
}

func (self ChannelEventHandler) HandleEventUnlockFailed(channel *ChannelService,
	unlockFailed *transfer.EventUnlockFailed) {
	log.Debug("[OnChannelEvent] Unlock Failed PaymentId: ", unlockFailed.Identifier)
}

func (self ChannelEventHandler) HandleSendWithdrawRequest(channel *ChannelService, withdrawRequestEvent *transfer.SendWithdrawRequest) {
	withdrawRequestMessage := messages.MessageFromSendEvent(withdrawRequestEvent)
	if withdrawRequestMessage != nil {
		err := channel.Sign(withdrawRequestMessage)
		if err != nil {
			log.Error("[HandleSendWithdrawRequest] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(withdrawRequestEvent.Recipient),
			ChannelIdentifier: withdrawRequestEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, withdrawRequestMessage)
	} else {
		log.Warn("[HandleSendWithdrawRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendWithdraw(channel *ChannelService, withdrawEvent *transfer.SendWithdraw) {
	withdrawMessage := messages.MessageFromSendEvent(withdrawEvent)
	if withdrawMessage != nil {
		err := channel.Sign(withdrawMessage)
		if err != nil {
			log.Error("[HandleSendWithdrawRequest] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(withdrawEvent.Recipient),
			ChannelIdentifier: withdrawEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, withdrawMessage)
	} else {
		log.Warn("[HandleSendWithdrawRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleContractSendChannelWithdraw(channel *ChannelService, channelWithdrawEvent *transfer.ContractSendChannelWithdraw) {

	args := channel.GetPaymentChannelArgs(channelWithdrawEvent.TokenNetworkIdentifier, channelWithdrawEvent.ChannelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelWithdraw, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelWithdrawEvent.TokenNetworkIdentifier), channelWithdrawEvent.ChannelIdentifier, args)

	// run in a goroutine in order that partner will not time out for the delivered message for withdraw
	go func() {
		err := channelProxy.Withdraw(channelWithdrawEvent.PartnerAddress, channelWithdrawEvent.TotalWithdraw, channelWithdrawEvent.PartnerSignature, channelWithdrawEvent.PartnerPublicKey,
			channelWithdrawEvent.ParticipantSignature, channelWithdrawEvent.ParticipantPublicKey)
		if err != nil {
			log.Warnf("HandleContractSendChannelWithdraw, proxy returns: %s", err)
			ok := channel.WithdrawResultNotify(channelWithdrawEvent.ChannelIdentifier, false)
			if !ok {
				log.Warnf("error in HandleContractSendChannelWithdraw, no withdraw status found in the map")
			}
		}
	}()
}

func (self ChannelEventHandler) HandleWithdrawRequestSentFailed(channel *ChannelService, withdrawRequestSentFailedEvent *transfer.EventWithdrawRequestSentFailed) {
	ok := channel.WithdrawResultNotify(withdrawRequestSentFailedEvent.ChannelIdentifier, false)
	if !ok {
		panic("error in HandleWithdrawRequestSentFailed, no withdraw status found in the map")
	}

	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(channel.StateFromChannel(),
		withdrawRequestSentFailedEvent.TokenNetworkIdentifier, withdrawRequestSentFailedEvent.ChannelIdentifier)
	if channelState == nil {
		panic("error in HandleWithdrawRequestSentFailed, channelState is nil")
	}

	transfer.DeleteWithdrawTransaction(channelState)
}

func (self ChannelEventHandler) HandleInvalidReceivedWithdraw(channel *ChannelService, invalidWithdrawReceivedEvent *transfer.EventInvalidReceivedWithdraw) {
	ok := channel.WithdrawResultNotify(invalidWithdrawReceivedEvent.ChannelIdentifier, false)
	if !ok {
		panic("error in HandleInvalidReceivedWithdraw, no withdraw status found in the map")
	}

	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(channel.StateFromChannel(),
		invalidWithdrawReceivedEvent.TokenNetworkIdentifier, invalidWithdrawReceivedEvent.ChannelIdentifier)
	if channelState == nil {
		panic("error in HandleInvalidReceivedWithdraw, channelState is nil")
	}

	transfer.DeleteWithdrawTransaction(channelState)

}

func (self ChannelEventHandler) HandleSendCooperativeSettleRequest(channel *ChannelService, cooperativeSettleRequestEvent *transfer.SendCooperativeSettleRequest) {
	cooperativeSettleRequestMessage := messages.MessageFromSendEvent(cooperativeSettleRequestEvent)
	if cooperativeSettleRequestMessage != nil {
		err := channel.Sign(cooperativeSettleRequestMessage)
		if err != nil {
			log.Error("[HandleSendCooperativeSettleRequest] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(cooperativeSettleRequestEvent.Recipient),
			ChannelIdentifier: cooperativeSettleRequestEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, cooperativeSettleRequestMessage)
	} else {
		log.Warn("[HandleSendCooperativeSettleRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendCooperativeSettle(channel *ChannelService, cooperativeSettleEvent *transfer.SendCooperativeSettle) {
	cooperativeSettleMessage := messages.MessageFromSendEvent(cooperativeSettleEvent)
	if cooperativeSettleMessage != nil {
		err := channel.Sign(cooperativeSettleMessage)
		if err != nil {
			log.Error("[HandleSendCooperativeSettle] ", err.Error())
			return
		}
		queueId := &transfer.QueueIdentifier{
			Recipient:         common.Address(cooperativeSettleEvent.Recipient),
			ChannelIdentifier: cooperativeSettleEvent.ChannelIdentifier,
		}
		channel.channelActor.Transport.SendAsync(queueId, cooperativeSettleMessage)
	} else {
		log.Warn("[HandleSendCooperativeSettleRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleContractSendChannelCooperativeSettle(channel *ChannelService, channelCooperativeSettleEvent *transfer.ContractSendChannelCooperativeSettle) {
	log.Debug("in HandleContractSendChannelCooperativeSettle")
	args := channel.GetPaymentChannelArgs(channelCooperativeSettleEvent.TokenNetworkIdentifier, channelCooperativeSettleEvent.ChannelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelCooperativeSettle, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelCooperativeSettleEvent.TokenNetworkIdentifier), channelCooperativeSettleEvent.ChannelIdentifier, args)

	go func() {
		err := channelProxy.CooperativeSettle(channelCooperativeSettleEvent.Participant1, channelCooperativeSettleEvent.Participant1Balance,
			channelCooperativeSettleEvent.Participant2, channelCooperativeSettleEvent.Participant2Balance, channelCooperativeSettleEvent.Participant1Signature,
			channelCooperativeSettleEvent.Participant1PublicKey, channelCooperativeSettleEvent.Participant2Signature, channelCooperativeSettleEvent.Participant2PublicKey)
		if err != nil {
			log.Warnf("[HandleContractSendChannelCooperativeSettle] error: %s", err)
		}
	}()
}

func (self ChannelEventHandler) HandleContractSendChannelUnlock(channel *ChannelService, channelUnlockEvent *transfer.ContractSendChannelBatchUnlock) {
	var chainID common.ChainID

	tokenNetworkIdentifier := channelUnlockEvent.TokenNetworkIdentifier
	channelIdentifier := channelUnlockEvent.ChannelIdentifier
	participant := channelUnlockEvent.Participant
	tokenAddress := channelUnlockEvent.TokenAddress

	args := channel.GetPaymentChannelArgs(tokenNetworkIdentifier, channelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelUnlock, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(tokenNetworkIdentifier), channelIdentifier, args)

	participanatsDetails := channelProxy.TokenNetwork.DetailParticipants(channelProxy.Participant1, channelProxy.Participant2, channelIdentifier)

	ourDetails := participanatsDetails.OurDetails
	ourLocksroot := ourDetails.Locksroot

	partnerDetails := participanatsDetails.PartnerDetails
	partnerLocksroot := partnerDetails.Locksroot

	emptyHash := common.Locksroot{}
	isPartnerUnlock := false
	if partnerDetails.Address == participant && partnerLocksroot != emptyHash {
		isPartnerUnlock = true
	}

	isOurUnlock := false
	if ourDetails.Address == participant && ourLocksroot != emptyHash {
		isOurUnlock = true
	}

	stateChangeIdentifier := 0

	if isPartnerUnlock {
		stateChangeRecord := storage.GetStateChangeWithBalanceProofByLocksroot(
			channel.Wal.Storage, chainID, tokenNetworkIdentifier, channelIdentifier,
			partnerLocksroot, partnerDetails.Address)

		if stateChangeRecord != nil {
			stateChangeIdentifier = stateChangeRecord.StateChangeIdentifier
		}
	} else if isOurUnlock {
		eventRecord := storage.GetEventWithBalanceProofByLocksroot(
			channel.Wal.Storage, chainID, tokenNetworkIdentifier, channelIdentifier,
			ourLocksroot)

		if eventRecord != nil {
			stateChangeIdentifier = eventRecord.StateChangeIdentifier
		}
	} else {
		log.Warn("Onchain unlock already mined")
		return
	}

	if stateChangeIdentifier == 0 {
		panic("Failed to find state/event that match current channel locksroots")
	}

	restoredChannelState := storage.ChannelStateUntilStateChange(channel.Wal.Storage, common.PaymentNetworkID{}, tokenAddress, channelIdentifier, stateChangeIdentifier)

	ourState := restoredChannelState.OurState
	partnerState := restoredChannelState.PartnerState

	var merkleTreeLeaves []*transfer.HashTimeLockState

	if common.AddressEqual(partnerState.Address, participant) {
		merkleTreeLeaves = transfer.GetBatchUnlock(partnerState)
	} else if common.AddressEqual(ourState.Address, participant) {
		merkleTreeLeaves = transfer.GetBatchUnlock(ourState)
	}

	channelProxy.Unlock(merkleTreeLeaves)
}
