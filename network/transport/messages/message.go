package messages

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"crypto/sha256"

	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChain/account"
	sig "github.com/oniio/oniChain/core/signature"
	"github.com/oniio/oniChain/core/types"
	"github.com/oniio/oniChain/crypto/keypair"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/transfer"
	"reflect"
	"github.com/oniio/oniChain/common/log"
)

type SignedMessageInterface interface {
	DataToSign() []byte
}

func BytesToUint64(data []byte) uint64 {
	var n uint64
	bytesBuffer := bytes.NewBuffer(data)
	binary.Read(bytesBuffer, binary.BigEndian, &n)
	return n
}
func Uint64ToBytes(n uint64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func (this *Processed) DataToSign() []byte {
	return Uint64ToBytes(this.MessageIdentifier.MessageId)
}

func (this *Delivered) DataToSign() []byte {
	if this.DeliveredMessageIdentifier == nil {
		log.Warn("this.DeliveredMessageIdentifier == nil")
	}
	return Uint64ToBytes(this.DeliveredMessageIdentifier.MessageId)
}

func (this *LockedTransfer) DataToSign() []byte {
	message := this.BaseMessage.EnvelopeMessage
	var locksRoot [32]byte
	copy(locksRoot[:], message.Locksroot.Locksroot[:32])

	log.Debug("[DataToSign]: ", common.TokenAmount(message.TransferredAmount.TokenAmount),
		common.TokenAmount(message.LockedAmount.TokenAmount), locksRoot)

	messageHash := transfer.HashBalanceData(common.TokenAmount(message.TransferredAmount.TokenAmount),
		common.TokenAmount(message.LockedAmount.TokenAmount), locksRoot)

	return message.DataToSign(messageHash)
}

// data to sign for envelopeMessage is the packed balance proof
func (this *EnvelopeMessage) DataToSign(additionalHash []byte) []byte {
	var addr [20]byte

	copy(addr[:], this.TokenNetworkAddress.TokenNetworkAddress)

	tokenNetworkAddr := common.TokenNetworkAddress(addr)
	chainId := common.ChainID(int(this.ChainId.ChainId))
	channelId := common.ChannelID(int(this.ChannelIdentifier.ChannelId))

	balanceHash := transfer.HashBalanceData(
		common.TokenAmount(this.TransferredAmount.TokenAmount),
		common.TokenAmount(this.LockedAmount.TokenAmount),
		ConvertLocksroot(this.Locksroot),
	)

	nonce := common.Nonce(this.Nonce)
	log.Debug("[LockedTransfer DataToSign] balanceHash: ", balanceHash)
	log.Debug("[LockedTransfer DataToSign] addtionalHash: ", additionalHash)
	return transfer.PackBalanceProof(nonce, balanceHash, additionalHash, channelId, tokenNetworkAddr, chainId, 1)
}


func (this *DirectTransfer) DataToSign() []byte {
	messageHash := MessageHash(this.Pack())
	return this.EnvelopeMessage.DataToSign(messageHash[:])
}

func (this *DirectTransfer) Pack() []byte {
	var buf bytes.Buffer

	env := this.EnvelopeMessage

	buf.Write(Uint64ToBytes(env.ChainId.ChainId))
	buf.Write(Uint64ToBytes(this.MessageIdentifier.MessageId))
	buf.Write(Uint64ToBytes(this.PaymentIdentifier.PaymentId))
	buf.Write(Uint64ToBytes(env.Nonce))
	buf.Write(this.Token.Address)
	buf.Write(env.TokenNetworkAddress.TokenNetworkAddress)
	buf.Write(Uint64ToBytes(env.ChannelIdentifier.ChannelId))
	buf.Write(Uint64ToBytes(env.TransferredAmount.TokenAmount))
	buf.Write(Uint64ToBytes(env.LockedAmount.TokenAmount))
	buf.Write(this.Recipient.Address)
	buf.Write(env.Locksroot.Locksroot)

	return buf.Bytes()
}

func (this *SecretRequest) DataToSign() []byte {
	var buf bytes.Buffer
	buf.Write(Uint64ToBytes(this.MessageIdentifier.MessageId))
	buf.Write(Uint64ToBytes(this.PaymentIdentifier.PaymentId))

	buf.Write(this.SecretHash.SecretHash[:])
	buf.Write(Uint64ToBytes(this.Amount.TokenAmount))
	buf.Write(Uint64ToBytes(this.Expiration.BlockExpiration))
	return buf.Bytes()
}

func (this *Secret) DataToSign() []byte {
	message := this.EnvelopeMessage
	var locksRoot [32]byte
	copy(locksRoot[:], message.Locksroot.Locksroot[:32])
	messageHash := transfer.HashBalanceData(common.TokenAmount(message.TransferredAmount.TokenAmount),
		common.TokenAmount(message.LockedAmount.TokenAmount), locksRoot)

	return message.DataToSign(messageHash)
}

func (this *RevealSecret) DataToSign() []byte {
	var buf bytes.Buffer
	buf.Write(Uint64ToBytes(this.MessageIdentifier.MessageId))
	buf.Write(this.Secret.Secret[:])
	return buf.Bytes()
}

func MessageHash(data []byte) []byte {
	sum := sha256.Sum256(data)

	return sum[:]
}

func MessageFromSendEvent(event interface{}) proto.Message {
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
	default:
		log.Debug("Unknown event type: ", reflect.TypeOf(event).String())
	}

	return nil
}

func ProcessedFromEvent(event *transfer.SendProcessed) proto.Message {
	msg := &Processed{
		MessageIdentifier: &MessageID{MessageId: (uint64)(event.MessageIdentifier)},
	}

	return msg
}
func DirectTransferFromEvent(event *transfer.SendDirectTransfer) proto.Message {
	bp := event.BalanceProof

	env := &EnvelopeMessage{
		ChainId:             &ChainID{uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{uint64(bp.LockedAmount)},
		Locksroot:           &Locksroot{[]byte(bp.LocksRoot[:])},
		ChannelIdentifier:   &ChannelID{uint64(bp.ChannelIdentifier)},
		TokenNetworkAddress: &TokenNetworkAddress{bp.TokenNetworkIdentifier[:]},
	}

	msg := &DirectTransfer{
		EnvelopeMessage:   env,
		MessageIdentifier: &MessageID{uint64(event.MessageIdentifier)},
		PaymentIdentifier: &PaymentID{uint64(event.PaymentIdentifier)},
		Token:             &Address{event.TokenAddress[:]},
		Recipient:         &Address{event.Recipient[:]},
	}

	return msg
}

func LockedTransferFromEvent(event *transfer.SendLockedTransfer) proto.Message {
	bp := event.Transfer.BalanceProof
	log.Debug("[LockedTransferFromEvent] lockedAmount: ", bp.LockedAmount)
	env := &EnvelopeMessage{
		ChainId:             &ChainID{uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{uint64(bp.LockedAmount)},
		Locksroot:           &Locksroot{[]byte(bp.LocksRoot[:])},
		ChannelIdentifier:   &ChannelID{uint64(bp.ChannelIdentifier)},
		TokenNetworkAddress: &TokenNetworkAddress{bp.TokenNetworkIdentifier[:]},
	}
	lock := &HashTimeLock{
		Amount:     &PaymentAmount{uint64(event.Transfer.Lock.Amount)},
		Expiration: &BlockExpiration{uint64(event.Transfer.Lock.Expiration)},
		SecretHash: &SecretHash{event.Transfer.Lock.SecretHash[:]},
	}

	transferBase := &LockedTransferBase{
		Lock:              lock,
		EnvelopeMessage:   env,
		MessageIdentifier: &MessageID{uint64(event.MessageIdentifier)},
		PaymentIdentifier: &PaymentID{uint64(event.Transfer.PaymentIdentifier)},
		Token:             &Address{event.Transfer.Token[:]},
		Recipient:         &Address{event.Recipient[:]},
	}

	msg := &LockedTransfer{
		BaseMessage: transferBase,
		Target:      &Address{event.Transfer.Target[:]},
		Initiator:   &Address{event.Transfer.Initiator[:]},
		Fee:         0,
	}
	return msg
}


func SecretRequestFromEvent(event *transfer.SendSecretRequest) proto.Message {
	msg := &SecretRequest{
		MessageIdentifier: &MessageID{MessageId: (uint64)(event.MessageIdentifier)},
		PaymentIdentifier: &PaymentID{PaymentId: uint64(event.PaymentIdentifier)},
		SecretHash:        &SecretHash{SecretHash: event.SecretHash[:]},
		Amount:            &TokenAmount{TokenAmount: uint64(event.Amount)},
		Expiration:        &BlockExpiration{BlockExpiration: uint64(event.Expiration)},
	}
	return msg
}

func UnlockFromEvent(event *transfer.SendBalanceProof) proto.Message {
	bp := event.BalanceProof
	log.Debug("[UnlockFromEvent] lockedAmount: ", bp.LockedAmount)
	env := &EnvelopeMessage{
		ChainId:             &ChainID{uint64(bp.ChainId)},
		Nonce:               uint64(bp.Nonce),
		TransferredAmount:   &TokenAmount{uint64(bp.TransferredAmount)},
		LockedAmount:        &TokenAmount{uint64(bp.LockedAmount)},
		Locksroot:           &Locksroot{[]byte(bp.LocksRoot[:])},
		ChannelIdentifier:   &ChannelID{uint64(bp.ChannelIdentifier)},
		TokenNetworkAddress: &TokenNetworkAddress{bp.TokenNetworkIdentifier[:]},
	}

	msg := &Secret{
		EnvelopeMessage:   env,
		MessageIdentifier: &MessageID{MessageId: (uint64)(event.MessageIdentifier)},
		PaymentIdentifier: &PaymentID{PaymentId: uint64(event.PaymentIdentifier)},
		Secret:            &SecretType{Secret: event.Secret},
	}
	return msg
}

func RevealSecretFromEvent(event *transfer.SendSecretReveal) proto.Message {
	msg := &RevealSecret{
		MessageIdentifier: &MessageID{MessageId: (uint64)(event.MessageIdentifier)},
		Secret:            &SecretType{Secret: event.Secret},
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
	case *Secret:
		msg := message.(*Secret)
		msg.EnvelopeMessage.Signature = &SignedMessage{
			Signature: sigData,
			Sender:    &Address{Address: account.Address[:]},
			Publickey: pubKey,
		}
	default:
		return fmt.Errorf("[Sign] Unknow message type to sign ", reflect.TypeOf(message).String())
	}
	log.Debug("[PubKey]: ", pubKey)
	log.Debug("[Data]: ", data)
	log.Debug("[SigObj]: ", sigData)
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

	if len(senderBuf) != constants.ADDR_LEN {
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
