package channelservice

import (
	"reflect"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/transfer"
)

type MessageHandler struct {
}

func (self *MessageHandler) OnMessage(channel *ChannelService, message interface{}) {
	log.Debug("[OnMessage] message: ", reflect.TypeOf(message).String())
	switch message.(type) {
	case *messages.DirectTransfer:
		self.HandleMessageDirecttransfer(channel, message.(*messages.DirectTransfer))
	case *messages.Delivered:
		self.HandleMessageDelivered(channel, message.(*messages.Delivered))
	case *messages.Processed:
		self.HandleMessageProcessed(channel, message.(*messages.Processed))
	case *messages.SecretRequest:
		self.HandleMessageSecretRequest(channel, message.(*messages.SecretRequest))
	case *messages.RevealSecret:
		self.HandleMessageRevealSecret(channel, message.(*messages.RevealSecret))
	case *messages.Secret:
		self.HandleMessageSecret(channel, message.(*messages.Secret))
	case *messages.LockExpired:
		self.HandleMessageLockExpired(channel, message.(*messages.LockExpired))
	case *messages.RefundTransfer:
		self.HandleMessageRefundTransfer(channel, message.(*messages.RefundTransfer))
	case *messages.LockedTransfer:
		self.HandleMessageLockedTransfer(channel, message.(*messages.LockedTransfer))
	default:
		log.Warn("[OnMessage] Unkown message. ", reflect.TypeOf(message).String())
	}
}

func (self *MessageHandler) HandleMessageDirecttransfer(channel *ChannelService, message *messages.DirectTransfer) {
	var tokenNetworkIdentifier common.TokenNetworkID

	copy(tokenNetworkIdentifier[:], message.EnvelopeMessage.TokenNetworkAddress.TokenNetworkAddress[:20])
	//todo check

	//balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	directTransfer := &transfer.ReceiveTransferDirect{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{Sender: balanceProof.Sender},
		TokenNetworkIdentifier:         tokenNetworkIdentifier,
		MessageIdentifier:              common.MessageID(message.MessageIdentifier.MessageId),
		PaymentIdentifier:              common.PaymentID(message.PaymentIdentifier.PaymentId),
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
		MessageIdentifier: common.MessageID(message.MessageIdentifier.MessageId),
	}

	channel.HandleStateChange(processed)
}

func (self *MessageHandler) HandleMessageDelivered(channel *ChannelService, message *messages.Delivered) {
	delivered := &transfer.ReceiveDelivered{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: messages.ConvertAddress(message.Signature.Sender),
		},
		MessageIdentifier: common.MessageID(message.DeliveredMessageIdentifier.MessageId),
	}

	channel.HandleStateChange(delivered)
}

func (self *MessageHandler) HandleMessageSecretRequest(channel *ChannelService,
	message *messages.SecretRequest) {
	var secretHash [32]byte
	var sender [20]byte
	copy(secretHash[:], message.SecretHash.SecretHash)
	copy(sender[:], message.Signature.Sender.Address)

	secretRequest := &transfer.ReceiveSecretRequest{
		PaymentIdentifier: common.PaymentID(message.PaymentIdentifier.PaymentId),
		Amount:            common.PaymentAmount(message.Amount.TokenAmount),
		Expiration:        common.BlockExpiration(message.Expiration.BlockExpiration),
		SecretHash:        common.SecretHash(secretHash),
		Sender:            sender,
		MessageIdentifier: common.MessageID(message.MessageIdentifier.MessageId),
	}
	channel.HandleStateChange(secretRequest)
}

func (self *MessageHandler) HandleMessageRevealSecret(channel *ChannelService, message *messages.RevealSecret) {
	var sender [20]byte
	copy(sender[:], message.Signature.Sender.Address)
	stateChange := &transfer.ReceiveSecretReveal{
		Secret:            message.Secret.Secret,
		Sender:            sender,
		MessageIdentifier: common.MessageID(message.MessageIdentifier.MessageId),
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageSecret(channel *ChannelService, message *messages.Secret) {
	var sender [20]byte
	copy(sender[:], message.EnvelopeMessage.Signature.Sender.Address)
	secretHash := common.GetHash(message.Secret.Secret)
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	stateChange := &transfer.ReceiveUnlock{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: sender,
		},
		MessageIdentifier: common.MessageID(message.MessageIdentifier.MessageId),
		Secret:            message.Secret.Secret,
		SecretHash:        secretHash,
		BalanceProof:      balanceProof,
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageLockExpired(channel *ChannelService, message *messages.LockExpired) {
	var secretHash [32]byte
	copy(secretHash[:], message.SecretHash.SecretHash)
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	stateChange := &transfer.ReceiveLockExpired{
		BalanceProof:      balanceProof,
		SecretHash:        secretHash,
		MessageIdentifier: common.MessageID(message.MessageIdentifier.MessageId),
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageRefundTransfer(channel *ChannelService, message *messages.RefundTransfer) {
	var tokenNetworkAddress common.Address
	copy(tokenNetworkAddress[:], message.Refund.BaseMessage.Token.Address)

	var previousAddress common.Address
	copy(previousAddress[:], message.Refund.Initiator.Address)

	fromTransfer := LockedTransferSignedFromMessage(message.Refund)
	chainState := channel.StateFromChannel()
	routes, _ := GetBestRoutes(chainState, common.TokenNetworkID(tokenNetworkAddress),
		common.Address(channel.address), common.Address(fromTransfer.Target),
		fromTransfer.Lock.Amount, previousAddress)

	role := transfer.GetTransferRole(chainState, common.SecretHash(fromTransfer.Lock.SecretHash))
	if role == "initiator" {
		secret := common.SecretRandom(constants.SECRET_LEN)
		stateChange := &transfer.ReceiveTransferRefundCancelRoute{
			Routes:   routes,
			Transfer: fromTransfer,
			Secret:   secret,
		}
		channel.HandleStateChange(stateChange)
	} else {
		stateChange := &transfer.ReceiveTransferRefund{
			Transfer: fromTransfer,
			Routes:   routes,
		}
		channel.HandleStateChange(stateChange)
	}
}

func (self *MessageHandler) HandleMessageLockedTransfer(channel *ChannelService, message *messages.LockedTransfer) {
	//todo check message.BaseMessage.Lock.SecretHash registered

	var targetAddress common.Address
	copy(targetAddress[:], message.Target.Address)
	if targetAddress == channel.address {
		var initiator common.Address
		copy(initiator[:], message.Initiator.Address)
		channel.transport.StartHealthCheck(initiator)
		initTargetStateChange := channel.TargetInit(message)
		channel.HandleStateChange(initTargetStateChange)
	} else {
		initMediatorStateChange := channel.MediatorInit(message)
		channel.HandleStateChange(initMediatorStateChange)
	}
}

func BalanceProofFromEnvelope(message *messages.EnvelopeMessage, dataToSign []byte) *transfer.BalanceProofSignedState {
	var tokenNetworkIdentifier common.TokenNetworkID
	var messageHash common.Keccak256
	var locksRoot common.Locksroot

	tmpLocksRoot := message.Locksroot.Locksroot
	copy(locksRoot[:], tmpLocksRoot[:32])

	log.Debug("[BalanceProofFromEnvelope]: ", locksRoot)
	tmpMessageHash := common.GetHash(dataToSign)
	copy(messageHash[:], tmpMessageHash[:32])
	copy(tokenNetworkIdentifier[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])
	if message.Signature == nil {
		log.Warn("BalanceProofFromEnvelope message.Signature is nil")
	}
	log.Debug("[BalanceProofFromEnvelope] lockedAmount: ", message.LockedAmount.TokenAmount)
	balanceProof := &transfer.BalanceProofSignedState{
		Nonce:                  common.Nonce(message.Nonce),
		TransferredAmount:      common.TokenAmount(message.TransferredAmount.TokenAmount),
		LockedAmount:           common.TokenAmount(message.LockedAmount.TokenAmount),
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		ChannelIdentifier:      common.ChannelID(message.ChannelIdentifier.ChannelId),
		MessageHash:            messageHash,
		Signature:              message.Signature.Signature,
		Sender:                 messages.ConvertAddress(message.Signature.Sender),
		ChainId:                common.ChainID(message.ChainId.ChainId),
		PublicKey:              message.Signature.Publickey,
		LocksRoot:              locksRoot,
	}

	return balanceProof
}

func LockedTransferSignedFromMessage(message *messages.LockedTransfer) *transfer.LockedTransferSignedState {
	//""" Create LockedTransferSignedState from a LockedTransfer message. """
	balanceProof := BalanceProofFromEnvelope(message.BaseMessage.EnvelopeMessage, message.Pack())

	var keccaKHash [32]byte
	copy(keccaKHash[:], message.BaseMessage.Lock.SecretHash.SecretHash[:32])
	lock := &transfer.HashTimeLockState{
		Amount:     common.TokenAmount(message.BaseMessage.Lock.Amount.PaymentAmount),
		Expiration: common.BlockHeight(message.BaseMessage.Lock.Expiration.BlockExpiration),
		SecretHash: common.Keccak256(keccaKHash),
	}

	var tokenAddress [20]byte
	copy(tokenAddress[:], message.BaseMessage.Token.Address[:20])

	var initAddress [20]byte
	copy(initAddress[:], message.Initiator.Address[:20])

	var targetAddress [20]byte
	copy(targetAddress[:], message.Target.Address[:20])

	transferState := &transfer.LockedTransferSignedState{
		MessageIdentifier: common.MessageID(message.BaseMessage.MessageIdentifier.MessageId),
		PaymentIdentifier: common.PaymentID(message.BaseMessage.PaymentIdentifier.PaymentId),
		Token:             tokenAddress,
		BalanceProof:      balanceProof,
		Lock:              lock,
		Initiator:         initAddress,
		Target:            targetAddress,
	}
	return transferState
}
