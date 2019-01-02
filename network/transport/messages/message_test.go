package messages

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChannel/account"
	"github.com/oniio/oniChannel/typing"
)

func BuildDirectTransfer(chainID typing.ChainID, nonce typing.Nonce, amount typing.TokenAmount,
	lockedAmount typing.TokenAmount, locksRoot typing.Locksroot, channelID typing.ChannelID,
	tokenNetworkID typing.TokenNetworkID, messageID typing.MessageID,
	paymentID typing.PaymentID, tokenAddress typing.TokenAddress, recipient typing.Address) proto.Message {
	env := &EnvelopeMessage{
		ChainId:             &ChainID{uint64(chainID)},
		Nonce:               uint64(nonce),
		TransferredAmount:   &TokenAmount{uint64(amount)},
		LockedAmount:        &TokenAmount{uint64(lockedAmount)},
		Locksroot:           &Locksroot{[]byte(locksRoot[:])},
		ChannelIdentifier:   &ChannelID{uint64(channelID)},
		TokenNetworkAddress: &TokenNetworkAddress{tokenNetworkID[:]},
	}

	msg := &DirectTransfer{
		EnvelopeMessage:   env,
		MessageIdentifier: &MessageID{uint64(messageID)},
		PaymentIdentifier: &PaymentID{uint64(paymentID)},
		Token:             &Address{tokenAddress[:]},
		Recipient:         &Address{recipient[:]},
	}

	return msg
}
func TestVerifySiganature(t *testing.T) {
	account := account.NewAccount()

	chainID := typing.ChainID(0)
	nonce := typing.Nonce(1)
	amount := typing.TokenAmount(100)
	lockedAmount := typing.TokenAmount(200)

	var locksRoot typing.Locksroot
	var tokenAddress typing.TokenAddress
	var recipient typing.Address
	var tokenNetworkID typing.TokenNetworkID

	channelID := typing.ChannelID(1)
	messageID := typing.MessageID(3)
	paymentID := typing.PaymentID(4)

	message := BuildDirectTransfer(chainID, nonce, amount, lockedAmount, locksRoot, channelID, tokenNetworkID, messageID, paymentID, tokenAddress, recipient)

	err := Sign(account, message.(SignedMessageInterface))
	if err != nil {
		t.Fatal("error signing the message")
	}

	directTransfer := message.(*DirectTransfer)
	err = Verify(directTransfer.DataToSign(), directTransfer)
	if err != nil {
		t.Fatal("error verify the message")
	}
}
