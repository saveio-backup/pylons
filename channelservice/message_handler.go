package channelservice

import (
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

type MessageHandler struct {
}

func (self *MessageHandler) OnMessage(channel *ChannelService, message interface{}) {

	switch message.(type) {
	case *messages.DirectTransfer:
		self.HandleMessageDirecttransfer(channel, message.(*messages.DirectTransfer))
	case *messages.Delivered:
		self.HandleMessageDelivered(channel, message.(*messages.Delivered))
	case *messages.Processed:
		self.HandleMessageProcessed(channel, message.(*messages.Processed))
	default:

	}
}

func (self *MessageHandler) HandleMessageDirecttransfer(channel *ChannelService, message *messages.DirectTransfer) {
	var tokenNetworkIdentifier typing.TokenNetworkID

	copy(tokenNetworkIdentifier[:], message.EnvelopeMessage.TokenNetworkAddress.TokenNetworkAddress[:20])
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())

	directTransfer := &transfer.ReceiveTransferDirect{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{Sender: balanceProof.Sender},
		TokenNetworkIdentifier:         tokenNetworkIdentifier,
		MessageIdentifier:              typing.MessageID(message.MessageIdentifier.MessageId),
		PaymentIdentifier:              typing.PaymentID(message.PaymentIdentifier.PaymentId),
		BalanceProof:                   balanceProof,
	}

	channel.HandleStateChange(directTransfer)
	return
}

func (self *MessageHandler) HandleMessageProcessed(channel *ChannelService, message *messages.Processed) {
	processed := &transfer.ReceiveProcessed{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: messages.ConvertAddress(message.Signature.Sender),
		},
		MessageIdentifier: typing.MessageID(message.MessageIdentifier.MessageId),
	}

	channel.HandleStateChange(processed)
}

func (self *MessageHandler) HandleMessageDelivered(channel *ChannelService, message *messages.Delivered) {
	delivered := &transfer.ReceiveDelivered{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: messages.ConvertAddress(message.Signature.Sender),
		},
		MessageIdentifier: typing.MessageID(message.DeliveredMessageIdentifier.MessageId),
	}

	channel.HandleStateChange(delivered)
}

func BalanceProofFromEnvelope(message *messages.EnvelopeMessage, dataToSign []byte) *transfer.BalanceProofSignedState {
	var tokenNetworkIdentifier typing.TokenNetworkID
	var messageHash typing.Keccak256

	copy(tokenNetworkIdentifier[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])
	copy(messageHash[:], messages.MessageHash(dataToSign)[:32])

	balanceProof := &transfer.BalanceProofSignedState{
		Nonce:                  typing.Nonce(message.Nonce),
		TransferredAmount:      typing.TokenAmount(message.TransferredAmount.TokenAmount),
		LockedAmount:           typing.TokenAmount(message.LockedAmount.TokenAmount),
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		ChannelIdentifier:      typing.ChannelID(message.ChannelIdentifier.ChannelId),
		MessageHash:            messageHash,
		Signature:              message.Signature.Signature,
		Sender:                 messages.ConvertAddress(message.Signature.Sender),
		ChainId:                typing.ChainID(message.ChainId.ChainId),
		PublicKey:              message.Signature.Publickey,
	}

	return balanceProof
}
