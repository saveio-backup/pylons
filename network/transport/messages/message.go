package messages

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"crypto/sha256"

	proto "github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChannel/account"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniChannel/utils"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology/core/types"
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
	return Uint64ToBytes(this.DeliveredMessageIdentifier.MessageId)
}

// data to sign for envelopeMessage is the packed balance proof
func (this *EnvelopeMessage) DataToSign(data []byte) []byte {
	var addr [20]byte

	copy(addr[:], this.TokenNetworkAddress.TokenNetworkAddress)

	tokenNetworkAddr := typing.TokenNetworkAddress(addr)
	chainId := typing.ChainID(int(this.ChainId.ChainId))
	channelId := typing.ChannelID(int(this.ChannelIdentifier.ChannelId))

	balanceHash := transfer.HashBalanceData(
		typing.TokenAmount(this.TransferredAmount.TokenAmount),
		typing.TokenAmount(this.LockedAmount.TokenAmount),
		ConvertLocksroot(this.Locksroot),
	)

	nonce := typing.Nonce(this.Nonce)
	addtionalHash := MessageHash(data)

	return transfer.PackBalanceProof(nonce, balanceHash, addtionalHash, channelId, tokenNetworkAddr, chainId, 1)
}

func (this *DirectTransfer) DataToSign() []byte {
	return this.EnvelopeMessage.DataToSign(this.Pack())
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
		//TODO : mediated transfer related messages to be added
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

func Sign(account *account.Account, message SignedMessageInterface) error {
	data := message.DataToSign()

	sigData, err := account.Sign(data)
	if err != nil {
		return err
	}

	pubKey := keypair.SerializePublicKey(account.GetPublicKey())
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
	default:
		return fmt.Errorf("Unknow message type to sign")
	}

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

func GetSender(message SignedMessageInterface) (typing.Address, error) {
	var sender typing.Address
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

	if len(senderBuf) != typing.ADDR_LEN {
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
	if sender != typing.Address(address) {
		return fmt.Errorf("sender and public key not match")
	}

	err = utils.VerifySignature(pubKey, data, signature)
	if err != nil {
		return err
	}

	return nil
}
