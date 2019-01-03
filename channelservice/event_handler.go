package channelservice

import (
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/storage"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniChannel/utils"
)

type ChannelEventHandler struct {
}

func (self ChannelEventHandler) OnChannelEvent(channel *ChannelService, event transfer.Event) {

	switch event.(type) {
	case *transfer.SendDirectTransfer:
		sendDirectTransfer := event.(*transfer.SendDirectTransfer)
		self.HandleSendDirecttransfer(channel, sendDirectTransfer)
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
	default:
	}
	return
}

func (self ChannelEventHandler) HandleSendDirecttransfer(channel *ChannelService, sendDirectTransfer *transfer.SendDirectTransfer) {
	message := messages.MessageFromSendEvent(sendDirectTransfer)
	if message != nil {
		err := channel.Sign(message)
		if err != nil {
			return
		}

		queueId := &transfer.QueueIdentifier{
			Recipient:         typing.Address(sendDirectTransfer.Recipient),
			ChannelIdentifier: sendDirectTransfer.ChannelIdentifier,
		}

		channel.transport.SendAsync(queueId, message)
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
			Recipient:         typing.Address(processedEvent.Recipient),
			ChannelIdentifier: processedEvent.ChannelIdentifier,
		}

		channel.transport.SendAsync(queueId, message)
	}

	return
}

func (self ChannelEventHandler) HandlePaymentSentSuccess(channel *ChannelService, paymentSentSuccessEvent *transfer.EventPaymentSentSuccess) {
	target := typing.Address(paymentSentSuccessEvent.Target)
	identifier := paymentSentSuccessEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		panic("error in HandlePaymentSentSuccess, no payment status found in the map")
	}

	channel.RemovePaymentStatus(target, identifier)

	paymentStatus.paymentDone <- true

	return
}

func (self ChannelEventHandler) HandlePaymentSentFailed(channel *ChannelService, paymentSentFailedEvent *transfer.EventPaymentSentFailed) {
	target := typing.Address(paymentSentFailedEvent.Target)
	identifier := paymentSentFailedEvent.Identifier

	paymentStatus, exist := channel.GetPaymentStatus(target, identifier)
	if !exist {
		panic("error in HandlePaymentSentFailed, no payment status found in the map")
	}

	channel.RemovePaymentStatus(target, identifier)

	paymentStatus.paymentDone <- false

	return
}

func (self ChannelEventHandler) HandlePaymentReceivedSuccess(channel *ChannelService, paymentReceivedSuccessEvent *transfer.EventPaymentReceivedSuccess) {
	for ch := range channel.ReceiveNotificationChannels {
		ch <- paymentReceivedSuccessEvent
	}
}

func (self ChannelEventHandler) HandleContractSendChannelClose(channel *ChannelService, channelCloseEvent *transfer.ContractSendChannelClose) {
	var nonce typing.Nonce
	var balanceHash typing.BalanceHash
	var signature typing.Signature
	var messageHash typing.Keccak256
	var publicKey typing.PubKey

	balanceProof := channelCloseEvent.BalanceProof

	if balanceProof != nil {
		balanceHash = transfer.HashBalanceData(
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
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

	channelProxy := channel.chain.PaymentChannel(typing.Address{}, channelCloseEvent.ChannelIdentifier, args)

	channelProxy.Close(nonce, balanceHash, typing.AdditionalHash(messageHash[:]), signature, publicKey)
}

func (self ChannelEventHandler) HandelContractSendChannelUpdate(channel *ChannelService, channelUpdateEvent *transfer.ContractSendChannelUpdateTransfer) {
	balanceProof := channelUpdateEvent.BalanceProof

	if balanceProof != nil {
		args := channel.GetPaymentChannelArgs(channelUpdateEvent.TokenNetworkIdentifier, channelUpdateEvent.ChannelIdentifier)
		if args == nil {
			panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
		}

		channelProxy := channel.chain.PaymentChannel(typing.Address{}, channelUpdateEvent.ChannelIdentifier, args)

		balanceHash := transfer.HashBalanceData(
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
		)

		nonClosingData := transfer.PackBalanceProofUpdate(
			balanceProof.Nonce, balanceHash, typing.AdditionalHash(balanceProof.MessageHash[:]),
			balanceProof.ChannelIdentifier, typing.TokenNetworkAddress(balanceProof.TokenNetworkIdentifier),
			balanceProof.ChainId, balanceProof.Signature)

		var ourSignature typing.Signature
		var nonClosePubkey typing.PubKey

		ourSignature, err := channel.Account.Sign(nonClosingData)
		if err != nil {
			return
		}

		nonClosePubkey = utils.GetPublicKeyBuf(channel.Account.GetPublicKey())

		channelProxy.UpdateTransfer(
			balanceProof.Nonce, balanceHash, typing.AdditionalHash(balanceProof.MessageHash[:]),
			balanceProof.Signature, ourSignature,
			balanceProof.PublicKey, nonClosePubkey)
	}
}

func (self ChannelEventHandler) HandleContractSendChannelSettle(channel *ChannelService, channelSettleEvent *transfer.ContractSendChannelSettle) {
	var ourTransferredAmount typing.TokenAmount
	var ourLockedAmount typing.TokenAmount
	var ourLocksroot typing.Locksroot
	var partnerTransferredAmount typing.TokenAmount
	var partnerLockedAmount typing.TokenAmount
	var partnerLocksroot typing.Locksroot
	var ourBalanceProof *transfer.BalanceProofUnsignedState
	var partnerBalanceProof *transfer.BalanceProofSignedState

	var chainID typing.ChainID

	args := channel.GetPaymentChannelArgs(typing.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier), channelSettleEvent.ChannelIdentifier)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}
	channelProxy := channel.chain.PaymentChannel(typing.Address{}, channelSettleEvent.ChannelIdentifier, args)

	participanatsDetails := channelProxy.TokenNetwork.DetailParticipants(channelProxy.Participant1, channelProxy.Participant2, channelSettleEvent.ChannelIdentifier)

	// when ourDetails or PartnerDetails is nil, it means the channel has been settled already
	if participanatsDetails.OurDetails == nil || participanatsDetails.PartnerDetails == nil {
		return
	}

	ourBalanceHash := participanatsDetails.OurDetails.BalanceHash
	if len(ourBalanceHash) != 0 {
		ourBalanceProof = storage.GetLatestKnownBalanceProofFromEvents(
			channel.Wal.Storage, chainID, typing.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier),
			channelSettleEvent.ChannelIdentifier, ourBalanceHash)
	}

	if ourBalanceProof != nil {
		ourTransferredAmount = ourBalanceProof.TransferredAmount
		ourLockedAmount = ourBalanceProof.LockedAmount
		ourLocksroot = ourBalanceProof.LocksRoot
	}

	partnerBalanceHash := participanatsDetails.PartnerDetails.BalanceHash
	if len(partnerBalanceHash) != 0 {
		partnerBalanceProof = storage.GetLatestKnownBalanceProofFromStateChanges(
			channel.Wal.Storage, chainID, typing.TokenNetworkID(channelSettleEvent.TokenNetworkIdentifier),
			channelSettleEvent.ChannelIdentifier, partnerBalanceHash, participanatsDetails.PartnerDetails.Address)
	}

	if partnerBalanceProof != nil {
		partnerTransferredAmount = partnerBalanceProof.TransferredAmount
		partnerLockedAmount = partnerBalanceProof.LockedAmount
		partnerLocksroot = partnerBalanceProof.LocksRoot
	}

	channelProxy.Settle(
		ourTransferredAmount, ourLockedAmount, ourLocksroot,
		partnerTransferredAmount, partnerLockedAmount, partnerLocksroot)
}
