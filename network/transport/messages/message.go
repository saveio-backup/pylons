package messages

import (
	"bytes"
	"fmt"

	"crypto/sha256"

	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
	sig "github.com/saveio/themis/core/signature"
	"github.com/saveio/themis/core/types"
	"github.com/saveio/themis/crypto/keypair"
)

type SignedMessageInterface interface {
	DataToSign() []byte
}

func (this *Processed) DataToSign() []byte {
	return transfer.Uint64ToBytes(this.MessageId.MessageId)
}

func (this *Delivered) DataToSign() []byte {
	if this.DeliveredMessageId == nil {
		log.Warn("this.DeliveredMessageId == nil")
	}
	return transfer.Uint64ToBytes(this.DeliveredMessageId.MessageId)
}

// data to sign for envelopeMessage is the packed balance proof
func (this *EnvelopeMessage) DataToSign(dataToSign []byte) []byte {
	var addr [20]byte
	copy(addr[:], this.TokenNetworkAddress.TokenNetworkAddress)

	nonce := common.Nonce(this.Nonce)
	tokenNetworkAddr := common.TokenNetworkAddress(addr)
	chainId := common.ChainID(int(this.ChainId.ChainId))
	channelId := common.ChannelID(int(this.ChannelId.ChannelId))

	balanceHash := transfer.HashBalanceData(
		common.TokenAmount(this.TransferredAmount.TokenAmount),
		common.TokenAmount(this.LockedAmount.TokenAmount),
		ConvertLocksRoot(this.LocksRoot),
	)
	var additionalHash []byte
	if dataToSign != nil {
		additionalHash = MessageHash(dataToSign)
	}

	log.Debug("[EnvelopeMessage DataToSign] balanceHash: ", balanceHash)
	log.Debug("[EnvelopeMessage DataToSign] additionalHash: ", additionalHash)
	return transfer.PackBalanceProof(nonce, balanceHash, additionalHash, channelId, tokenNetworkAddr, chainId, 1)
}

func (this *LockedTransfer) DataToSign() []byte {
	return this.BaseMessage.EnvelopeMessage.DataToSign(this.Pack())
}

func (this *LockedTransfer) Pack() []byte {
	var buf bytes.Buffer
	env := this.BaseMessage.EnvelopeMessage
	buf.Write(transfer.Uint64ToBytes(env.ChainId.ChainId))
	buf.Write(transfer.Uint64ToBytes(this.BaseMessage.MessageId.MessageId))
	buf.Write(transfer.Uint64ToBytes(this.BaseMessage.PaymentId.PaymentId))
	buf.Write(transfer.Uint64ToBytes(env.Nonce))
	buf.Write(this.BaseMessage.Token.Address)
	buf.Write(env.TokenNetworkAddress.TokenNetworkAddress)
	buf.Write(transfer.Uint64ToBytes(env.ChannelId.ChannelId))
	buf.Write(transfer.Uint64ToBytes(env.TransferredAmount.TokenAmount))
	buf.Write(transfer.Uint64ToBytes(env.LockedAmount.TokenAmount))
	buf.Write(this.BaseMessage.Recipient.Address)
	buf.Write(env.LocksRoot.LocksRoot)

	return buf.Bytes()
}

func (this *DirectTransfer) DataToSign() []byte {
	return this.EnvelopeMessage.DataToSign(this.Pack())
}

func (this *DirectTransfer) Pack() []byte {
	var buf bytes.Buffer
	env := this.EnvelopeMessage
	buf.Write(transfer.Uint64ToBytes(env.ChainId.ChainId))
	buf.Write(transfer.Uint64ToBytes(this.MessageId.MessageId))
	buf.Write(transfer.Uint64ToBytes(this.PaymentId.PaymentId))
	buf.Write(transfer.Uint64ToBytes(env.Nonce))
	buf.Write(this.Token.Address)
	buf.Write(env.TokenNetworkAddress.TokenNetworkAddress)
	buf.Write(transfer.Uint64ToBytes(env.ChannelId.ChannelId))
	buf.Write(transfer.Uint64ToBytes(env.TransferredAmount.TokenAmount))
	buf.Write(transfer.Uint64ToBytes(env.LockedAmount.TokenAmount))
	buf.Write(this.Recipient.Address)
	buf.Write(env.LocksRoot.LocksRoot)

	return buf.Bytes()
}

func (this *SecretRequest) DataToSign() []byte {
	var buf bytes.Buffer
	buf.Write(transfer.Uint64ToBytes(this.MessageId.MessageId))
	buf.Write(transfer.Uint64ToBytes(this.PaymentId.PaymentId))
	buf.Write(this.SecretHash.SecretHash[:])
	buf.Write(transfer.Uint64ToBytes(this.Amount.TokenAmount))
	buf.Write(transfer.Uint64ToBytes(this.Expiration.BlockExpiration))
	return buf.Bytes()
}

func (this *BalanceProof) DataToSign() []byte {
	return this.EnvelopeMessage.DataToSign(this.Pack())
}

func (this *BalanceProof) Pack() []byte {
	var buf bytes.Buffer
	env := this.EnvelopeMessage
	buf.Write(transfer.Uint64ToBytes(env.ChainId.ChainId))
	buf.Write(transfer.Uint64ToBytes(this.MessageId.MessageId))
	buf.Write(transfer.Uint64ToBytes(this.PaymentId.PaymentId))
	buf.Write(transfer.Uint64ToBytes(env.Nonce))
	buf.Write(env.TokenNetworkAddress.TokenNetworkAddress)
	buf.Write(transfer.Uint64ToBytes(env.ChannelId.ChannelId))
	buf.Write(transfer.Uint64ToBytes(env.TransferredAmount.TokenAmount))
	buf.Write(transfer.Uint64ToBytes(env.LockedAmount.TokenAmount))
	buf.Write(env.LocksRoot.LocksRoot)

	return buf.Bytes()
}

func (this *LockExpired) DataToSign() []byte {
	return this.EnvelopeMessage.DataToSign(this.Pack())
}

func (this *LockExpired) Pack() []byte {
	var buf bytes.Buffer
	env := this.EnvelopeMessage

	buf.Write(transfer.Uint64ToBytes(env.ChainId.ChainId))
	buf.Write(transfer.Uint64ToBytes(env.Nonce))
	buf.Write(transfer.Uint64ToBytes(this.MessageId.MessageId))
	buf.Write(env.TokenNetworkAddress.TokenNetworkAddress)
	buf.Write(transfer.Uint64ToBytes(env.ChannelId.ChannelId))
	buf.Write(transfer.Uint64ToBytes(env.TransferredAmount.TokenAmount))
	buf.Write(transfer.Uint64ToBytes(env.LockedAmount.TokenAmount))
	buf.Write(this.Recipient.Address)
	buf.Write(env.LocksRoot.LocksRoot)
	buf.Write(this.SecretHash.SecretHash[:])
	//packed.signature = self.signature

	return buf.Bytes()
}

func (this *RevealSecret) DataToSign() []byte {
	var buf bytes.Buffer
	buf.Write(transfer.Uint64ToBytes(this.MessageId.MessageId))
	buf.Write(this.Secret.Secret[:])
	return buf.Bytes()
}

func (this *WithdrawRequest) DataToSign() []byte {
	channelId := common.ChannelID(int(this.ChannelId.ChannelId))
	participant := ConvertAddress(this.Participant)
	withdrawAmount := common.TokenAmount(this.WithdrawAmount.TokenAmount)

	return transfer.PackWithdraw(channelId, participant, withdrawAmount)
}

func (this *Withdraw) DataToSign() []byte {
	channelId := common.ChannelID(int(this.ChannelId.ChannelId))
	participant := ConvertAddress(this.Participant)
	withdrawAmount := common.TokenAmount(this.WithdrawAmount.TokenAmount)

	return transfer.PackWithdraw(channelId, participant, withdrawAmount)
}

func (this *CooperativeSettleRequest) DataToSign() []byte {
	channelId := common.ChannelID(int(this.ChannelId.ChannelId))
	participant1 := ConvertAddress(this.Participant1)
	participant1Balance := common.TokenAmount(this.Participant1Balance.TokenAmount)
	participant2 := ConvertAddress(this.Participant2)
	participant2Balance := common.TokenAmount(this.Participant2Balance.TokenAmount)

	return transfer.PackCooperativeSettle(channelId, participant1, participant1Balance, participant2, participant2Balance)
}

func (this *CooperativeSettle) DataToSign() []byte {
	channelId := common.ChannelID(int(this.ChannelId.ChannelId))
	participant1 := ConvertAddress(this.Participant1)
	participant1Balance := common.TokenAmount(this.Participant1Balance.TokenAmount)
	participant2 := ConvertAddress(this.Participant2)
	participant2Balance := common.TokenAmount(this.Participant2Balance.TokenAmount)

	return transfer.PackCooperativeSettle(channelId, participant1, participant1Balance, participant2, participant2Balance)
}

func (this *RefundTransfer) DataToSign() []byte {
	return this.Refund.BaseMessage.EnvelopeMessage.DataToSign(this.Pack())
}

func (this *RefundTransfer) Pack() []byte {
	return this.Refund.Pack()
}

func MessageHash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

func MessageFromSendEvent(event interface{}) proto.Message {
	log.Debug("[MessageFromSendEvent] Event type: ", reflect.TypeOf(event).String())
	switch event.(type) {
	case *transfer.SendDirectTransfer:
		return DirectTransferFromEvent(event.(*transfer.SendDirectTransfer))
	case *transfer.SendProcessed:
		return ProcessedFromEvent(event.(*transfer.SendProcessed))
	case *transfer.SendLockedTransfer:
		return LockedTransferFromEvent(event.(*transfer.SendLockedTransfer))
	case *transfer.SendSecretReveal:
		return RevealSecretFromEvent(event.(*transfer.SendSecretReveal))
	case *transfer.SendBalanceProof:
		return UnlockFromEvent(event.(*transfer.SendBalanceProof))
	case *transfer.SendSecretRequest:
		return SecretRequestFromEvent(event.(*transfer.SendSecretRequest))
	case *transfer.SendRefundTransfer:
		return RefundTransferFromEvent(event.(*transfer.SendRefundTransfer))
	case *transfer.SendLockExpired:
		return LockExpiredFromEvent(event.(*transfer.SendLockExpired))
	case *transfer.SendWithdrawRequest:
		return WithdrawRequestFromEvent(event.(*transfer.SendWithdrawRequest))
	case *transfer.SendWithdraw:
		return WithdrawFromEvent(event.(*transfer.SendWithdraw))
	case *transfer.SendCooperativeSettleRequest:
		return CooperativeSettleRequestFromEvent(event.(*transfer.SendCooperativeSettleRequest))
	case *transfer.SendCooperativeSettle:
		return CooperativeSettleFromEvent(event.(*transfer.SendCooperativeSettle))
	default:
		log.Debug("Unknown event type: ", reflect.TypeOf(event).String())
	}

	return nil
}

func ProcessedFromEvent(event *transfer.SendProcessed) proto.Message {
	msg := &Processed{
		MessageId: &MessageID{MessageId: (uint64)(event.MessageId)},
	}

	return msg
}

func DirectTransferFromEvent(event *transfer.SendDirectTransfer) proto.Message {
	bp := event.BalanceProof

	env := &EnvelopeMessage{
		ChainId:             &ChainID{ChainId: uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{TokenAmount: uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{TokenAmount: uint64(bp.LockedAmount)},
		LocksRoot:           &LocksRoot{LocksRoot: []byte(bp.LocksRoot[:])},
		ChannelId:           &ChannelID{ChannelId: uint64(bp.ChannelId)},
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: bp.TokenNetworkId[:]},
	}

	msg := &DirectTransfer{
		EnvelopeMessage: env,
		MessageId:       &MessageID{MessageId: uint64(event.MessageId)},
		PaymentId:       &PaymentID{PaymentId: uint64(event.PaymentId)},
		Token:           &Address{Address: event.TokenAddress[:]},
		Recipient:       &Address{Address: event.Recipient[:]},
	}

	return msg
}

func LockedTransferFromEvent(event *transfer.SendLockedTransfer) proto.Message {
	bp := event.Transfer.BalanceProof
	log.Debug("[LockedTransferFromEvent] lockedAmount: ", bp.LockedAmount)
	env := &EnvelopeMessage{
		ChainId:             &ChainID{ChainId: uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{TokenAmount: uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{TokenAmount: uint64(bp.LockedAmount)},
		LocksRoot:           &LocksRoot{LocksRoot: []byte(bp.LocksRoot[:])},
		ChannelId:           &ChannelID{ChannelId: uint64(bp.ChannelId)},
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: bp.TokenNetworkId[:]},
	}
	lock := &HashTimeLock{
		Amount:     &PaymentAmount{PaymentAmount: uint64(event.Transfer.Lock.Amount)},
		Expiration: &BlockExpiration{BlockExpiration: uint64(event.Transfer.Lock.Expiration)},
		SecretHash: &SecretHash{SecretHash: event.Transfer.Lock.SecretHash[:]},
	}

	transferBase := &LockedTransferBase{
		Lock:            lock,
		EnvelopeMessage: env,
		MessageId:       &MessageID{MessageId: uint64(event.MessageId)},
		PaymentId:       &PaymentID{PaymentId: uint64(event.Transfer.PaymentId)},
		Token:           &Address{Address: event.Transfer.Token[:]},
		Recipient:       &Address{Address: event.Recipient[:]},
	}

	mediators := make([]*Address, 0, len(event.Transfer.Mediators))
	for _, mediator := range event.Transfer.Mediators {
		log.Infof("event transfer mediator %s", common.ToBase58(mediator))
		var addr common.Address
		copy(addr[:], mediator[:])
		mediators = append(mediators, &Address{Address: addr[:]})
	}
	msg := &LockedTransfer{
		BaseMessage: transferBase,
		Target:      &Address{Address: event.Transfer.Target[:]},
		Initiator:   &Address{Address: event.Transfer.Initiator[:]},
		EncSecret:   &EncSecret{EncSecret: event.Transfer.EncSecret[:]},
		Fee:         0,
		Mediators:   mediators,
	}
	return msg
}

func SecretRequestFromEvent(event *transfer.SendSecretRequest) proto.Message {
	msg := &SecretRequest{
		MessageId:  &MessageID{MessageId: (uint64)(event.MessageId)},
		PaymentId:  &PaymentID{PaymentId: uint64(event.PaymentId)},
		SecretHash: &SecretHash{SecretHash: event.SecretHash[:]},
		Amount:     &TokenAmount{TokenAmount: uint64(event.Amount)},
		Expiration: &BlockExpiration{BlockExpiration: uint64(event.Expiration)},
	}
	return msg
}

func RefundTransferFromEvent(event *transfer.SendRefundTransfer) proto.Message {
	lockedTransferEvent := &transfer.SendLockedTransfer{
		SendMessageEvent: event.SendMessageEvent,
		Transfer:         event.Transfer,
	}

	lockedTransfer := LockedTransferFromEvent(lockedTransferEvent)
	msg := &RefundTransfer{
		Refund: lockedTransfer.(*LockedTransfer),
	}

	return msg
}

func UnlockFromEvent(event *transfer.SendBalanceProof) proto.Message {
	bp := event.BalanceProof
	log.Debug("[UnlockFromEvent] lockedAmount: ", bp.LockedAmount)
	env := &EnvelopeMessage{
		ChainId:             &ChainID{ChainId: uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{TokenAmount: uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{TokenAmount: uint64(bp.LockedAmount)},
		LocksRoot:           &LocksRoot{LocksRoot: []byte(bp.LocksRoot[:])},
		ChannelId:           &ChannelID{ChannelId: uint64(bp.ChannelId)},
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: bp.TokenNetworkId[:]},
	}

	msg := &BalanceProof{
		EnvelopeMessage: env,
		MessageId:       &MessageID{MessageId: (uint64)(event.MessageId)},
		PaymentId:       &PaymentID{PaymentId: uint64(event.PaymentId)},
		Secret:          &SecretType{Secret: event.Secret},
	}
	return msg
}

func RevealSecretFromEvent(event *transfer.SendSecretReveal) proto.Message {
	msg := &RevealSecret{
		MessageId: &MessageID{MessageId: (uint64)(event.MessageId)},
		Secret:    &SecretType{Secret: event.Secret},
	}
	return msg
}

func LockExpiredFromEvent(event *transfer.SendLockExpired) proto.Message {
	bp := event.BalanceProof
	log.Debug("[LockExpiredFromEvent] lockedAmount: ", bp.LockedAmount)
	env := &EnvelopeMessage{
		ChainId:             &ChainID{ChainId: uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{TokenAmount: uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{TokenAmount: uint64(bp.LockedAmount)},
		LocksRoot:           &LocksRoot{LocksRoot: []byte(bp.LocksRoot[:])},
		ChannelId:           &ChannelID{ChannelId: uint64(bp.ChannelId)},
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: bp.TokenNetworkId[:]},
	}

	msg := &LockExpired{
		EnvelopeMessage: env,
		MessageId:       &MessageID{MessageId: (uint64)(event.MessageId)},
		Recipient:       &Address{Address: event.Recipient[:]},
		SecretHash:      &SecretHash{SecretHash: event.SecretHash[:]},
	}
	return msg
}

// fwtoo : refacotr to remove duplicate code
func WithdrawRequestFromEvent(event *transfer.SendWithdrawRequest) proto.Message {
	msg := &WithdrawRequest{
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: event.TokenNetworkId[:]},
		MessageId:           &MessageID{MessageId: (uint64)(event.MessageId)},
		ChannelId:           &ChannelID{ChannelId: uint64(event.ChannelId)},
		Participant:         &Address{Address: event.Participant[:]},
		WithdrawAmount:      &TokenAmount{TokenAmount: uint64(event.WithdrawAmount)},
	}
	return msg
}

func WithdrawFromEvent(event *transfer.SendWithdraw) proto.Message {
	msg := &Withdraw{
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: event.TokenNetworkId[:]},
		MessageId:           &MessageID{MessageId: (uint64)(event.MessageId)},
		ChannelId:           &ChannelID{ChannelId: uint64(event.ChannelId)},
		Participant:         &Address{Address: event.Participant[:]},
		WithdrawAmount:      &TokenAmount{TokenAmount: uint64(event.WithdrawAmount)},
		ParticipantSignature: &SignedMessage{
			Signature: event.ParticipantSignature,
			Sender:    &Address{Address: event.ParticipantAddress[:]},
			Publickey: event.ParticipantPublicKey,
		},
	}
	return msg
}

func CooperativeSettleRequestFromEvent(event *transfer.SendCooperativeSettleRequest) proto.Message {
	msg := &CooperativeSettleRequest{
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: event.TokenNetworkId[:]},
		MessageId:           &MessageID{MessageId: (uint64)(event.MessageId)},
		ChannelId:           &ChannelID{ChannelId: uint64(event.ChannelId)},
		Participant1:        &Address{Address: event.Participant1[:]},
		Participant1Balance: &TokenAmount{TokenAmount: uint64(event.Participant1Balance)},
		Participant2:        &Address{Address: event.Participant2[:]},
		Participant2Balance: &TokenAmount{TokenAmount: uint64(event.Participant2Balance)},
	}
	return msg
}

func CooperativeSettleFromEvent(event *transfer.SendCooperativeSettle) proto.Message {
	msg := &CooperativeSettle{
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: event.TokenNetworkId[:]},
		MessageId:           &MessageID{MessageId: (uint64)(event.MessageId)},
		ChannelId:           &ChannelID{ChannelId: uint64(event.ChannelId)},
		Participant1:        &Address{Address: event.Participant1[:]},
		Participant1Balance: &TokenAmount{TokenAmount: uint64(event.Participant1Balance)},
		Participant2:        &Address{Address: event.Participant2[:]},
		Participant2Balance: &TokenAmount{TokenAmount: uint64(event.Participant2Balance)},
		Participant1Signature: &SignedMessage{
			Signature: event.Participant1Signature,
			Sender:    &Address{Address: event.Participant1Address[:]},
			Publickey: event.Participant1PublicKey,
		},
	}
	return msg
}

func Sign(account *account.Account, message SignedMessageInterface) error {
	log.Debug("[Sign]: ", reflect.TypeOf(message).String())
	data := message.DataToSign()

	sigData, err := sig.Sign(account, data)
	if err != nil {
		return err
	}

	pubKey := keypair.SerializePublicKey(account.PubKey())
	switch message.(type) {
	case *DirectTransfer:
		msg := message.(*DirectTransfer)
		msg.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *Processed:
		msg := message.(*Processed)
		msg.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *Delivered:
		msg := message.(*Delivered)
		msg.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *LockedTransfer:
		msg := message.(*LockedTransfer)
		msg.BaseMessage.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *RefundTransfer:
		msg := message.(*RefundTransfer)
		msg.Refund.BaseMessage.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *SecretRequest:
		msg := message.(*SecretRequest)
		msg.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *RevealSecret:
		msg := message.(*RevealSecret)
		msg.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *BalanceProof:
		msg := message.(*BalanceProof)
		msg.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *LockExpired:
		msg := message.(*LockExpired)
		msg.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *WithdrawRequest:
		msg := message.(*WithdrawRequest)
		msg.ParticipantSignature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *Withdraw:
		msg := message.(*Withdraw)
		msg.PartnerSignature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *CooperativeSettleRequest:
		msg := message.(*CooperativeSettleRequest)
		msg.Participant1Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	case *CooperativeSettle:
		msg := message.(*CooperativeSettle)
		msg.Participant2Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	default:
		return fmt.Errorf("[Sign] Unknow message type to sign %v", reflect.TypeOf(message).String())
	}
	//log.Debug("Sign [PubKey]: ", pubKey)
	//log.Debug("Sign [Data]: ", data)
	//log.Debug("Sign [Signature]: ", sigData)
	return nil
}
func GetPublicKeyFromEnvelope(message *EnvelopeMessage) (keypair.PublicKey, error) {
	pubKeyBuf := message.Signature.Publickey

	pubKey, err := keypair.DeserializePublicKey(pubKeyBuf)
	if err != nil {
		return nil, fmt.Errorf("deserialize publickey error")
	}

	return pubKey, nil
}
func GetPublicKey(message SignedMessageInterface) (keypair.PublicKey, error) {
	var pubKeyBuf []byte

	switch message.(type) {
	case *DirectTransfer:
		msg := message.(*DirectTransfer)
		pubKeyBuf = msg.EnvelopeMessage.Signature.Publickey
	case *Processed:
		msg := message.(*Processed)
		pubKeyBuf = msg.Signature.Publickey
	default:
		return nil, fmt.Errorf("Unknow message type to GetPublicKey")
	}

	pubKey, err := keypair.DeserializePublicKey(pubKeyBuf)
	if err != nil {
		return nil, fmt.Errorf("deserialize publickey error")
	}

	return pubKey, nil
}

func GetSender(message SignedMessageInterface) (common.Address, error) {
	var sender common.Address
	var senderBuf []byte

	switch message.(type) {
	case *DirectTransfer:
		msg := message.(*DirectTransfer)
		senderBuf = msg.EnvelopeMessage.Signature.Sender.Address
	case *Processed:
		msg := message.(*Processed)
		senderBuf = msg.Signature.Sender.Address
	default:
		return sender, fmt.Errorf("Unknow message type to GetSender")
	}

	if len(senderBuf) != constants.AddrLen {
		return sender, fmt.Errorf("invalid sender length")
	}

	copy(sender[:], senderBuf[:20])

	return sender, nil
}

func GetSignature(message SignedMessageInterface) ([]byte, error) {
	var signature []byte

	switch message.(type) {
	case *DirectTransfer:
		msg := message.(*DirectTransfer)
		signature = msg.EnvelopeMessage.Signature.Signature
	case *Processed:
		msg := message.(*Processed)
		signature = msg.Signature.Signature
	default:
		return nil, fmt.Errorf("Unknow message type to GetSignature")
	}

	return signature, nil
}

func Verify(data []byte, message SignedMessageInterface) error {
	pubKey, err := GetPublicKey(message)
	if err != nil {
		return nil
	}

	sender, err := GetSender(message)
	if err != nil {
		return nil
	}

	signature, err := GetSignature(message)
	if err != nil {
		return err
	}

	address := types.AddressFromPubKey(pubKey)
	if sender != common.Address(address) {
		return fmt.Errorf("sender and public key not match")
	}

	err = common.VerifySignature(pubKey, data, signature)
	if err != nil {
		return err
	}

	return nil
}
