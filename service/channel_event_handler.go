package service

import (
	"reflect"
	"errors"
	"fmt"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/storage"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	sdkutils "github.com/saveio/themis-go-sdk/utils"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
)

type ChannelEventHandler struct {
}

func (self ChannelEventHandler) OnChannelEvent(channel *ChannelService, event transfer.Event) {
	log.Debugf("[OnChannelEvent]  type: %v", reflect.TypeOf(event).String())
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
	case *transfer.SendLockExpired:
		sendLockExpired := event.(*transfer.SendLockExpired)
		self.HandleSendLockedExpire(channel, sendLockExpired)
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
	case *transfer.EventWithdrawRequestTimeout:
		eventWithdrawRequestTimeout := event.(*transfer.EventWithdrawRequestTimeout)
		self.HandleWithdrawRequestTimeout(channel, eventWithdrawRequestTimeout)
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
	case *transfer.EventInvalidReceivedLockExpired:

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

		queueId := &transfer.QueueId{
			Recipient: common.Address(sendDirectTransfer.Recipient),
			ChannelId: sendDirectTransfer.ChannelId,
		}

		ret := channel.Transport.SendAsync(queueId, message)
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
		queueId := &transfer.QueueId{
			Recipient: common.Address(processedEvent.Recipient),
			ChannelId: processedEvent.ChannelId,
		}

		channel.Transport.SendAsync(queueId, message)
	} else {
		log.Warn("[HandleSendProcessed] Message is nil")
	}

	return
}

func (self ChannelEventHandler) HandlePaymentSentSuccess(channel *ChannelService, paymentSentSuccessEvent *transfer.EventPaymentSentSuccess) {
	target := paymentSentSuccessEvent.Target
	identifier := paymentSentSuccessEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		log.Error("error in HandlePaymentSentSuccess, no payment status found in the map")
		return
	}

	log.Debug("set paymentDone to true")
	paymentStatus.paymentDone <- true

	return
}

func (self ChannelEventHandler) HandlePaymentSentFailed(channel *ChannelService, paymentSentFailedEvent *transfer.EventPaymentSentFailed) {
	target := common.Address(paymentSentFailedEvent.Target)
	identifier := paymentSentFailedEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		log.Error("error in HandlePaymentSentFailed, no payment status found in the map")
		return
	}

	log.Debug("set paymentDone to false")
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

	args := channel.GetPaymentChannelArgs(channelCloseEvent.TokenNetworkId, channelCloseEvent.ChannelId)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelCloseEvent.TokenNetworkId), channelCloseEvent.ChannelId, args)
	channelProxy.Close(nonce, balanceHash, messageHash[:], signature, publicKey)
}

func (self ChannelEventHandler) HandelContractSendChannelUpdate(channel *ChannelService, channelUpdateEvent *transfer.ContractSendChannelUpdateTransfer) {
	balanceProof := channelUpdateEvent.BalanceProof

	if balanceProof != nil {
		args := channel.GetPaymentChannelArgs(channelUpdateEvent.TokenNetworkId, channelUpdateEvent.ChannelId)
		if args == nil {
			panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
		}

		channelProxy := channel.chain.PaymentChannel(common.Address(channelUpdateEvent.TokenNetworkId), channelUpdateEvent.ChannelId, args)

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
			balanceProof.ChannelId, common.TokenNetworkAddress(balanceProof.TokenNetworkId),
			balanceProof.ChainId, balanceProof.Signature)

		var ourSignature common.Signature
		var nonClosePubkey common.PubKey

		ourSignature, err := sdkutils.Sign(channel.Account, nonClosingData)
		if err != nil {
			return
		}

		nonClosePubkey = keypair.SerializePublicKey(channel.Account.PubKey())

		channelProxy.UpdateTransfer(
			balanceProof.Nonce, balanceHash,
			balanceProof.MessageHash[:],
			balanceProof.Signature, ourSignature,
			balanceProof.PublicKey, nonClosePubkey)
	}
}

func (self ChannelEventHandler) HandleContractSendChannelSettle(channel *ChannelService, channelSettleEvent *transfer.ContractSendChannelSettle) {
	var ourTransferredAmount common.TokenAmount
	var ourLockedAmount common.TokenAmount
	var ourLocksroot common.LocksRoot
	var partnerTransferredAmount common.TokenAmount
	var partnerLockedAmount common.TokenAmount
	var partnerLocksroot common.LocksRoot
	var ourBalanceProof *transfer.BalanceProofUnsignedState
	var partnerBalanceProof *transfer.BalanceProofSignedState

	var chainID common.ChainID

	args := channel.GetPaymentChannelArgs(common.TokenNetworkID(channelSettleEvent.TokenNetworkId), channelSettleEvent.ChannelId)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}
	channelProxy := channel.chain.PaymentChannel(common.Address(channelSettleEvent.TokenNetworkId), channelSettleEvent.ChannelId, args)

	participanatsDetails := channelProxy.TokenNetwork.DetailParticipants(channelProxy.Participant1, channelProxy.Participant2, channelSettleEvent.ChannelId)

	// when ourDetails or PartnerDetails is nil, it means the channel has been settled already
	if participanatsDetails.OurDetails == nil || participanatsDetails.PartnerDetails == nil {
		return
	}

	ourBalanceHash := participanatsDetails.OurDetails.BalanceHash
	log.Debugf("[ChannelSettle] ourBalanceHash %v", ourBalanceHash)
	if !common.IsEmptyBalanceHash(ourBalanceHash) {
		ourBalanceProof = storage.GetLatestKnownBalanceProofFromEvents(
			channel.Wal.Storage, chainID, common.TokenNetworkID(channelSettleEvent.TokenNetworkId),
			channelSettleEvent.ChannelId, ourBalanceHash)
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
			channel.Wal.Storage, chainID, common.TokenNetworkID(channelSettleEvent.TokenNetworkId),
			channelSettleEvent.ChannelId, partnerBalanceHash, participanatsDetails.PartnerDetails.Address)
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

		queueId := &transfer.QueueId{
			Recipient: sendLockedTransfer.Recipient,
			ChannelId: sendLockedTransfer.ChannelId,
		}
		channel.Transport.SendAsync(queueId, mediatedTransferMessage)
	} else {
		log.Warn("[HandleSendLockedTransfer] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleSendLockedExpire(channel *ChannelService, sendLockExpired *transfer.SendLockExpired) {
	sendLockExpiredEventMessage := messages.MessageFromSendEvent(sendLockExpired)
	if sendLockExpiredEventMessage != nil {
		err := channel.Sign(sendLockExpiredEventMessage)
		if err != nil {
			log.Error("[HandleSendLockedExpire] ", err.Error())
			return
		}

		queueId := &transfer.QueueId{
			Recipient: common.Address(sendLockExpired.Recipient),
			ChannelId: sendLockExpired.ChannelId,
		}
		channel.Transport.SendAsync(queueId, sendLockExpiredEventMessage)
	} else {
		log.Warn("[HandleSendLockedExpire] Message is nil")
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

		queueId := &transfer.QueueId{
			Recipient: common.Address(revealSecretEvent.Recipient),
			ChannelId: revealSecretEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, revealSecretMessage)
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
		queueId := &transfer.QueueId{
			Recipient: common.Address(balanceProofEvent.Recipient),
			ChannelId: balanceProofEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, unlockMessage)
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
		queueId := &transfer.QueueId{
			Recipient: secretRequestEvent.Recipient,
			ChannelId: secretRequestEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, secretRequestMessage)
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
		queueId := &transfer.QueueId{
			Recipient: common.Address(refundTransferEvent.Recipient),
			ChannelId: refundTransferEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, refundTransferMessage)
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
		/*
			queueId := &transfer.QueueId{
				Recipient:         common.Address(withdrawRequestEvent.Recipient),
				ChannelId: withdrawRequestEvent.ChannelId,
			}
			channel.Transport.SendAsync(queueId, withdrawRequestMessage)
		*/
		// send directly instead of push to queue
		err = channel.Transport.Send(withdrawRequestEvent.Recipient, withdrawRequestMessage)
		if err != nil {
			log.Debugf("SendWithdrawRequest failed : %s", err)
			self.handleWithdrawFailure(channel, withdrawRequestEvent.TokenNetworkId, withdrawRequestEvent.ChannelId,
				errors.New("send withdraw request failed"))
			return
		}
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
		/*
			queueId := &transfer.QueueId{
				Recipient:         common.Address(withdrawEvent.Recipient),
				ChannelId: withdrawEvent.ChannelId,
			}
			channel.Transport.SendAsync(queueId, withdrawMessage)
		*/
		err = channel.Transport.Send(withdrawEvent.Recipient, withdrawMessage)
		if err != nil {
			log.Debugf("SendWithdrawfailed : %s", err)
		}
	} else {
		log.Warn("[HandleSendWithdrawRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleContractSendChannelWithdraw(channel *ChannelService, channelWithdrawEvent *transfer.ContractSendChannelWithdraw) {

	args := channel.GetPaymentChannelArgs(channelWithdrawEvent.TokenNetworkId, channelWithdrawEvent.ChannelId)
	if args == nil {
		panic("error in HandleContractSendChannelWithdraw, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelWithdrawEvent.TokenNetworkId), channelWithdrawEvent.ChannelId, args)

	// run in a goroutine in order that partner will not time out for the delivered message for withdraw
	go func() {
		err := channelProxy.Withdraw(channelWithdrawEvent.PartnerAddress, channelWithdrawEvent.TotalWithdraw, channelWithdrawEvent.PartnerSignature, channelWithdrawEvent.PartnerPublicKey,
			channelWithdrawEvent.ParticipantSignature, channelWithdrawEvent.ParticipantPublicKey)
		if err != nil {
			log.Warnf("HandleContractSendChannelWithdraw, proxy returns: %s", err)
			ok := channel.WithdrawResultNotify(channelWithdrawEvent.ChannelId, false, err)
			if !ok {
				log.Warnf("error in HandleContractSendChannelWithdraw, no withdraw status found in the map")
			}
		}
	}()
}

func (self ChannelEventHandler) HandleWithdrawRequestSentFailed(channel *ChannelService,
	withdrawRequestSentFailedEvent *transfer.EventWithdrawRequestSentFailed) {
	log.Debugf("HandleWithdrawRequestSentFailed")
	errMsg := fmt.Sprintf("withdraw request sent failed: %s", withdrawRequestSentFailedEvent.Reason)
	self.handleWithdrawFailure(channel, withdrawRequestSentFailedEvent.TokenNetworkId,
		withdrawRequestSentFailedEvent.ChannelId, errors.New(errMsg))
}

func (self ChannelEventHandler) HandleInvalidReceivedWithdraw(channel *ChannelService,
	invalidWithdrawReceivedEvent *transfer.EventInvalidReceivedWithdraw) {
	log.Debugf("HandleInvalidReceivedWithdraw")
	self.handleWithdrawFailure(channel, invalidWithdrawReceivedEvent.TokenNetworkId,
		invalidWithdrawReceivedEvent.ChannelId, errors.New("invalid receive withdraw"))
}

func (self ChannelEventHandler) HandleWithdrawRequestTimeout(channel *ChannelService,
	withdrawRequestTimeoutEvent *transfer.EventWithdrawRequestTimeout) {
	log.Debugf("HandleWithdrawRequestTimeout")
	self.handleWithdrawFailure(channel, withdrawRequestTimeoutEvent.TokenNetworkId,
		withdrawRequestTimeoutEvent.ChannelId, errors.New("withdraw request timeout"))
}

func (self ChannelEventHandler) handleWithdrawFailure(channel *ChannelService, TokenNetworkId common.TokenNetworkID,
	channelId common.ChannelID, err error) {

	ok := channel.WithdrawResultNotify(channelId, false, err)
	if !ok {
		log.Warnf("handleWithdrawFailure no withdraw status found in the map")
	}

	channelState := transfer.GetChannelStateByTokenNetworkId(channel.StateFromChannel(), TokenNetworkId, channelId)
	if channelState == nil {
		panic("error in handleWithdrawFailure, channelState is nil")
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
		queueId := &transfer.QueueId{
			Recipient: common.Address(cooperativeSettleRequestEvent.Recipient),
			ChannelId: cooperativeSettleRequestEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, cooperativeSettleRequestMessage)
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
		queueId := &transfer.QueueId{
			Recipient: common.Address(cooperativeSettleEvent.Recipient),
			ChannelId: cooperativeSettleEvent.ChannelId,
		}
		channel.Transport.SendAsync(queueId, cooperativeSettleMessage)
	} else {
		log.Warn("[HandleSendCooperativeSettleRequest] Message is nil")
	}
	return
}

func (self ChannelEventHandler) HandleContractSendChannelCooperativeSettle(channel *ChannelService, channelCooperativeSettleEvent *transfer.ContractSendChannelCooperativeSettle) {
	log.Debug("in HandleContractSendChannelCooperativeSettle")
	args := channel.GetPaymentChannelArgs(channelCooperativeSettleEvent.TokenNetworkId, channelCooperativeSettleEvent.ChannelId)
	if args == nil {
		panic("error in HandleContractSendChannelCooperativeSettle, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(channelCooperativeSettleEvent.TokenNetworkId), channelCooperativeSettleEvent.ChannelId, args)

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

	TokenNetworkId := channelUnlockEvent.TokenNetworkId
	channelId := channelUnlockEvent.ChannelId
	participant := channelUnlockEvent.Participant
	tokenAddress := channelUnlockEvent.TokenAddress

	args := channel.GetPaymentChannelArgs(TokenNetworkId, channelId)
	if args == nil {
		panic("error in HandleContractSendChannelUnlock, cannot get paymentchannel args")
	}

	channelProxy := channel.chain.PaymentChannel(common.Address(TokenNetworkId), channelId, args)

	participanatsDetails := channelProxy.TokenNetwork.DetailParticipants(channelProxy.Participant1, channelProxy.Participant2, channelId)

	ourDetails := participanatsDetails.OurDetails
	ourLocksroot := ourDetails.LocksRoot

	partnerDetails := participanatsDetails.PartnerDetails
	partnerLocksroot := partnerDetails.LocksRoot

	emptyHash := common.LocksRoot{}
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
			channel.Wal.Storage, chainID, TokenNetworkId, channelId,
			partnerLocksroot, partnerDetails.Address)

		if stateChangeRecord != nil {
			stateChangeIdentifier = stateChangeRecord.StateChangeIdentifier
		}
	} else if isOurUnlock {
		eventRecord := storage.GetEventWithBalanceProofByLocksroot(
			channel.Wal.Storage, chainID, TokenNetworkId, channelId,
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

	restoredChannelState := storage.ChannelStateUntilStateChange(channel.Wal.Storage, common.PaymentNetworkID{},
		tokenAddress, channelId, stateChangeIdentifier, channel.address)

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
