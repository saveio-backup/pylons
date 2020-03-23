package messages

import (
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
)

func BuildDirectTransfer(chainID common.ChainID, nonce common.Nonce, amount common.TokenAmount,
	lockedAmount common.TokenAmount, locksRoot common.LocksRoot, channelID common.ChannelID,
	tokenNetworkID common.TokenNetworkID, messageID common.MessageID,
	paymentID common.PaymentID, tokenAddress common.TokenAddress, recipient common.Address) proto.Message {
	env := &EnvelopeMessage{
		ChainId:             &ChainID{ChainId: uint64(chainID)},
		Nonce:               uint64(nonce),
		TransferredAmount:   &TokenAmount{TokenAmount: uint64(amount)},
		LockedAmount:        &TokenAmount{TokenAmount: uint64(lockedAmount)},
		LocksRoot:           &LocksRoot{LocksRoot: []byte(locksRoot[:])},
		ChannelId:           &ChannelID{ChannelId: uint64(channelID)},
		TokenNetworkAddress: &TokenNetworkAddress{TokenNetworkAddress: tokenNetworkID[:]},
	}

	msg := &DirectTransfer{
		EnvelopeMessage: env,
		MessageId:       &MessageID{MessageId: uint64(messageID)},
		PaymentId:       &PaymentID{PaymentId: uint64(paymentID)},
		Token:           &Address{Address: tokenAddress[:]},
		Recipient:       &Address{Address: recipient[:]},
	}

	return msg
}
func TestVerifySignature(t *testing.T) {
	account := account.NewAccount("")

	chainID := common.ChainID(0)
	nonce := common.Nonce(1)
	amount := common.TokenAmount(100)
	lockedAmount := common.TokenAmount(200)

	var locksRoot common.LocksRoot
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

func TestAppendMediator(t *testing.T) {
	testAddr := make([]common.Address, 0)
	addr1, _ := common.FromBase58("AaH1pqFW3YskUDwpg7XqrqxgviBb57JyK1")
	addr2, _ := common.FromBase58("AanMAAHLVdpsGFo5QMbQDkfpMTcNGAFXNz")
	testAddr = append(testAddr, addr1)
	testAddr = append(testAddr, addr2)
	mediators := make([]*Address, 0, len(testAddr))
	for _, mediator := range testAddr {
		log.Infof("event transfer mediator %s", common.ToBase58(mediator))
		var addr common.Address
		copy(addr[:], mediator[:])
		mediators = append(mediators, &Address{Address: addr[:]})
	}
	log.Infof("event transfer mediator %v", mediators)
}
