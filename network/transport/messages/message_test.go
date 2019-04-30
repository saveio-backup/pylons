package messages

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChain/account"
	"github.com/saveio/pylons/common"
)

func BuildDirectTransfer(chainID common.ChainID, nonce common.Nonce, amount common.TokenAmount,
	lockedAmount common.TokenAmount, locksRoot common.Locksroot, channelID common.ChannelID,
	tokenNetworkID common.TokenNetworkID, messageID common.MessageID,
	paymentID common.PaymentID, tokenAddress common.TokenAddress, recipient common.Address) proto.Message {
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
	account := account.NewAccount("")

	chainID := common.ChainID(0)
	nonce := common.Nonce(1)
	amount := common.TokenAmount(100)
	lockedAmount := common.TokenAmount(200)

	var locksRoot common.Locksroot
	var tokenAddress common.TokenAddress
	var recipient common.Address
	var tokenNetworkID common.TokenNetworkID

	channelID := common.ChannelID(1)
	messageID := common.MessageID(3)
	paymentID := common.PaymentID(4)

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
