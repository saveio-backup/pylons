package service

import (
	"reflect"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

type MessageHandler struct {}

func (self *MessageHandler) OnMessage(channelSrv *ChannelService, message interface{}) {
	log.Debug("[OnMessage] message: ", reflect.TypeOf(message).String())
	switch msg := message.(type) {
	case *messages.DirectTransfer:
		self.HandleMessageDirectTransfer(channelSrv, msg)
	case *messages.Delivered:
		self.HandleMessageDelivered(channelSrv, msg)
	case *messages.Processed:
		self.HandleMessageProcessed(channelSrv, msg)
	case *messages.SecretRequest:
		self.HandleMessageSecretRequest(channelSrv, msg)
	case *messages.RevealSecret:
		self.HandleMessageRevealSecret(channelSrv, msg)
	case *messages.BalanceProof:
		self.HandleMessageSecret(channelSrv, msg)
	case *messages.LockExpired:
		self.HandleMessageLockExpired(channelSrv, msg)
	case *messages.RefundTransfer:
		self.HandleMessageRefundTransfer(channelSrv, msg)
	case *messages.LockedTransfer:
		self.HandleMessageLockedTransfer(channelSrv, msg)
	case *messages.WithdrawRequest:
		self.HandleMessageWithdrawRequest(channelSrv, msg)
	case *messages.Withdraw:
		self.HandleMessageWithdraw(channelSrv, msg)
	case *messages.CooperativeSettleRequest:
		self.HandleMessageCooperativeSettleRequest(channelSrv, msg)
	case *messages.CooperativeSettle:
		self.HandleMessageCooperativeSettle(channelSrv, msg)
	default:
		log.Warn("[OnMessage] Unknown message. ", reflect.TypeOf(message).String())
	}
}

func (self *MessageHandler) HandleMessageDirectTransfer(channel *ChannelService, message *messages.DirectTransfer) {
	var tokenNetworkId common.TokenNetworkID

	copy(tokenNetworkId[:], message.EnvelopeMessage.TokenNetworkAddress.TokenNetworkAddress[:20])
	//todo check

	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	directTransfer := &transfer.ReceiveTransferDirect{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{Sender: balanceProof.Sender},
		TokenNetworkId:                 tokenNetworkId,
		MessageId:                      common.MessageID(message.MessageId.MessageId),
		PaymentId:                      common.PaymentID(message.PaymentId.PaymentId),
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
		MessageId: common.MessageID(message.MessageId.MessageId),
	}

	channel.HandleStateChange(processed)
}

func (self *MessageHandler) HandleMessageDelivered(channel *ChannelService, message *messages.Delivered) {
	delivered := &transfer.ReceiveDelivered{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: messages.ConvertAddress(message.Signature.Sender),
		},
		MessageId: common.MessageID(message.DeliveredMessageId.MessageId),
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
		PaymentId:  common.PaymentID(message.PaymentId.PaymentId),
		Amount:     common.PaymentAmount(message.Amount.TokenAmount),
		Expiration: common.BlockExpiration(message.Expiration.BlockExpiration),
		SecretHash: common.SecretHash(secretHash),
		Sender:     sender,
		MessageId:  common.MessageID(message.MessageId.MessageId),
	}
	channel.HandleStateChange(secretRequest)
}

func (self *MessageHandler) HandleMessageRevealSecret(channel *ChannelService, message *messages.RevealSecret) {
	var sender [20]byte
	copy(sender[:], message.Signature.Sender.Address)
	stateChange := &transfer.ReceiveSecretReveal{
		Secret:    message.Secret.Secret,
		Sender:    sender,
		MessageId: common.MessageID(message.MessageId.MessageId),
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageSecret(channel *ChannelService, message *messages.BalanceProof) {
	var sender [20]byte
	copy(sender[:], message.EnvelopeMessage.Signature.Sender.Address)
	secretHash := common.GetHash(message.Secret.Secret)
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	stateChange := &transfer.ReceiveUnlock{
		AuthenticatedSenderStateChange: transfer.AuthenticatedSenderStateChange{
			Sender: sender,
		},
		MessageId:    common.MessageID(message.MessageId.MessageId),
		Secret:       message.Secret.Secret,
		SecretHash:   secretHash,
		BalanceProof: balanceProof,
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageLockExpired(channel *ChannelService, message *messages.LockExpired) {
	var secretHash [32]byte
	copy(secretHash[:], message.SecretHash.SecretHash)
	balanceProof := BalanceProofFromEnvelope(message.EnvelopeMessage, message.Pack())
	stateChange := &transfer.ReceiveLockExpired{
		BalanceProof: balanceProof,
		SecretHash:   secretHash,
		MessageId:    common.MessageID(message.MessageId.MessageId),
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageRefundTransfer(channel *ChannelService, message *messages.RefundTransfer) {
	var tokenNetworkAddress common.Address
	copy(tokenNetworkAddress[:], message.Refund.BaseMessage.Token.Address)

	var previousAddress common.Address
	copy(previousAddress[:], message.Refund.BaseMessage.EnvelopeMessage.Signature.Sender.Address)

	fromTransfer := LockedTransferSignedFromMessage(message.Refund)
	chainState := channel.StateFromChannel()
	routes, _ := GetBestRoutes(channel, common.TokenNetworkID(tokenNetworkAddress),
		channel.address, fromTransfer.Target, fromTransfer.Lock.Amount, []common.Address{previousAddress})

	role := transfer.GetTransferRole(chainState, common.SecretHash(fromTransfer.Lock.SecretHash))
	if role == "initiator" {
		secret := common.SecretRandom(constants.SecretLen)
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
		initTargetStateChange := channel.TargetInit(message)
		channel.HandleStateChange(initTargetStateChange)
	} else {
		initMediatorStateChange := channel.MediatorInit(message)
		channel.HandleStateChange(initMediatorStateChange)
	}
}

func (self *MessageHandler) HandleMessageWithdrawRequest(channel *ChannelService, message *messages.WithdrawRequest) {
	var tokenNetworkId common.TokenNetworkID

	copy(tokenNetworkId[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])

	stateChange := &transfer.ReceiveWithdrawRequest{
		TokenNetworkId:       tokenNetworkId,
		MessageId:            common.MessageID(message.MessageId.MessageId),
		ChannelId:            common.ChannelID(message.ChannelId.ChannelId),
		Participant:          messages.ConvertAddress(message.Participant),
		TotalWithdraw:        common.TokenAmount(message.WithdrawAmount.TokenAmount),
		ParticipantSignature: message.ParticipantSignature.Signature,
		ParticipantAddress:   messages.ConvertAddress(message.ParticipantSignature.Sender),
		ParticipantPublicKey: message.ParticipantSignature.Publickey,
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageWithdraw(channel *ChannelService, message *messages.Withdraw) {
	var tokenNetworkId common.TokenNetworkID

	channelId := common.ChannelID(message.ChannelId.ChannelId)

	copy(tokenNetworkId[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])

	channelState := transfer.GetChannelStateByTokenNetworkId(channel.StateFromChannel(),
		tokenNetworkId, channelId)
	if channelState == nil {
		return
	}

	// only process incoming withdraw message if we sent a withdraw request before
	withdrawTx := transfer.GetWithdrawTransaction(channelState)
	if withdrawTx != nil {
		stateChange := &transfer.ReceiveWithdraw{
			ReceiveWithdrawRequest: transfer.ReceiveWithdrawRequest{
				TokenNetworkId:       tokenNetworkId,
				ChannelId:            channelId,
				Participant:          messages.ConvertAddress(message.Participant),
				TotalWithdraw:        common.TokenAmount(message.WithdrawAmount.TokenAmount),
				ParticipantSignature: message.ParticipantSignature.Signature,
				ParticipantAddress:   messages.ConvertAddress(message.ParticipantSignature.Sender),
				ParticipantPublicKey: message.ParticipantSignature.Publickey,
			},
			PartnerSignature: message.PartnerSignature.Signature,
			PartnerAddress:   messages.ConvertAddress(message.PartnerSignature.Sender),
			PartnerPublicKey: message.PartnerSignature.Publickey,
		}
		channel.HandleStateChange(stateChange)
	}
}

func (self *MessageHandler) HandleMessageCooperativeSettleRequest(channel *ChannelService, message *messages.CooperativeSettleRequest) {
	var tokenNetworkId common.TokenNetworkID

	copy(tokenNetworkId[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])

	stateChange := &transfer.ReceiveCooperativeSettleRequest{
		TokenNetworkId:        tokenNetworkId,
		MessageId:             common.MessageID(message.MessageId.MessageId),
		ChannelId:             common.ChannelID(message.ChannelId.ChannelId),
		Participant1:          messages.ConvertAddress(message.Participant1),
		Participant1Balance:   common.TokenAmount(message.Participant1Balance.TokenAmount),
		Participant2:          messages.ConvertAddress(message.Participant2),
		Participant2Balance:   common.TokenAmount(message.Participant2Balance.TokenAmount),
		Participant1Signature: message.Participant1Signature.Signature,
		Participant1Address:   messages.ConvertAddress(message.Participant1Signature.Sender),
		Participant1PublicKey: message.Participant1Signature.Publickey,
	}
	channel.HandleStateChange(stateChange)
}

func (self *MessageHandler) HandleMessageCooperativeSettle(channel *ChannelService, message *messages.CooperativeSettle) {
	var tokenNetworkId common.TokenNetworkID

	channelId := common.ChannelID(message.ChannelId.ChannelId)

	copy(tokenNetworkId[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])

	// ne need to check here?
	channelState := transfer.GetChannelStateByTokenNetworkId(channel.StateFromChannel(),
		tokenNetworkId, channelId)
	if channelState == nil {
		return
	}

	// todo : need to check if we send a cooperative settle
	//withdrawTx := transfer.GetWithdrawTransaction(channelState)
	//if withdrawTx != nil {
	//}

	stateChange := &transfer.ReceiveCooperativeSettle{
		TokenNetworkId:        tokenNetworkId,
		MessageId:             common.MessageID(message.MessageId.MessageId),
		ChannelId:             common.ChannelID(message.ChannelId.ChannelId),
		Participant1:          messages.ConvertAddress(message.Participant1),
		Participant1Balance:   common.TokenAmount(message.Participant1Balance.TokenAmount),
		Participant2:          messages.ConvertAddress(message.Participant2),
		Participant2Balance:   common.TokenAmount(message.Participant2Balance.TokenAmount),
		Participant1Signature: message.Participant1Signature.Signature,
		Participant1Address:   messages.ConvertAddress(message.Participant1Signature.Sender),
		Participant1PublicKey: message.Participant1Signature.Publickey,
		Participant2Signature: message.Participant2Signature.Signature,
		Participant2Address:   messages.ConvertAddress(message.Participant2Signature.Sender),
		Participant2PublicKey: message.Participant2Signature.Publickey,
	}
	channel.HandleStateChange(stateChange)
}

func BalanceProofFromEnvelope(message *messages.EnvelopeMessage, dataToSign []byte) *transfer.BalanceProofSignedState {
	var tokenNetworkId common.TokenNetworkID
	var messageHash common.Keccak256
	var locksRoot common.LocksRoot

	tmpLocksRoot := message.LocksRoot.LocksRoot
	copy(locksRoot[:], tmpLocksRoot[:32])

	log.Debug("[BalanceProofFromEnvelope]: ", locksRoot)
	tmpMessageHash := common.GetHash(dataToSign)
	copy(messageHash[:], tmpMessageHash[:32])
	copy(tokenNetworkId[:], message.TokenNetworkAddress.TokenNetworkAddress[:20])
	if message.Signature == nil {
		log.Warn("BalanceProofFromEnvelope message.Signature is nil")
	}
	log.Debug("[BalanceProofFromEnvelope] lockedAmount: ", message.LockedAmount.TokenAmount)
	balanceProof := &transfer.BalanceProofSignedState{
		Nonce:             common.Nonce(message.Nonce),
		TransferredAmount: common.TokenAmount(message.TransferredAmount.TokenAmount),
		LockedAmount:      common.TokenAmount(message.LockedAmount.TokenAmount),
		TokenNetworkId:    tokenNetworkId,
		ChannelId:         common.ChannelID(message.ChannelId.ChannelId),
		MessageHash:       messageHash,
		Signature:         message.Signature.Signature,
		Sender:            messages.ConvertAddress(message.Signature.Sender),
		ChainId:           common.ChainID(message.ChainId.ChainId),
		PublicKey:         message.Signature.Publickey,
		LocksRoot:         locksRoot,
	}

	balanceProof.BalanceHash = transfer.HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

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

	mediators := make([]common.Address, 0, len(message.Mediators))
	for _, addr := range message.Mediators {
		var mediator [constants.AddrLen]byte
		copy(mediator[:], addr.Address[:constants.AddrLen])
		mediators = append(mediators, mediator)
		log.Infof("append mediator %s", common.ToBase58(mediator))
	}

	transferState := &transfer.LockedTransferSignedState{
		MessageId:    common.MessageID(message.BaseMessage.MessageId.MessageId),
		PaymentId:    common.PaymentID(message.BaseMessage.PaymentId.PaymentId),
		Token:        tokenAddress,
		BalanceProof: balanceProof,
		Lock:         lock,
		Initiator:    initAddress,
		Target:       targetAddress,
		EncSecret:    common.EncSecret(message.EncSecret.EncSecret),
		Mediators:    mediators,
	}
	return transferState
}
