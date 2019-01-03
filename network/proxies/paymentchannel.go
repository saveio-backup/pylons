package proxies

import (
	"container/list"
	"errors"
	"sync"

	"github.com/oniio/oniChannel/typing"
)

type PaymentChannel struct {
	TokenNetwork      *TokenNetwork
	channelIdentifier typing.ChannelID
	Participant1      typing.Address
	Participant2      typing.Address
	openBlockHeight   typing.BlockHeight
	settleTimeout     typing.BlockHeight
	closeBlockHeight  typing.BlockHeight
}

func NewPaymentChannel(tokenNetwork *TokenNetwork, channelIdentifier typing.ChannelID, args map[string]interface{}) (*PaymentChannel, error) {
	var participant1, participant2 typing.Address

	//[NOTE] only new opened channel will really execute this function.
	//existing PaymentChannel for existed channel should be found in BlockchainService

	self := new(PaymentChannel)
	if v, exist := args["participant1"]; exist {
		participant1 = v.(typing.Address)
	}
	if v, exist := args["participant2"]; exist {
		participant2 = v.(typing.Address)
	}

	if tokenNetwork.nodeAddress != participant1 && tokenNetwork.nodeAddress != participant2 {
		return nil, errors.New("One participant must be the node address")
	}

	if tokenNetwork.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	if v, exist := args["blockHeight"]; exist {
		self.openBlockHeight = v.(typing.BlockHeight)
	}
	if v, exist := args["settleTimeout"]; exist {
		self.settleTimeout = v.(typing.BlockHeight)
	}

	selfIdentifier = channelIdentifier

	self.Participant1 = participant1
	self.Participant2 = participant2
	self.TokenNetwork = tokenNetwork

	return self, nil
}

func (self *PaymentChannel) GetChannelId() typing.ChannelID {
	return selfIdentifier
}

func (self *PaymentChannel) LockOrRaise() *sync.Mutex {
	opLock := self.TokenNetwork.getOperationLock(self.Participant2)

	return opLock
}

func (self *PaymentChannel) tokenAddress() typing.Address {
	return self.TokenNetwork.TokenAddress()
}

func (self *PaymentChannel) Detail() *ChannelDetails {
	return self.TokenNetwork.detail(self.Participant1, self.Participant2, selfIdentifier)
}

//Should be set when open channel, should NOT get it by filter log
func (self PaymentChannel) SettleTimeout() typing.BlockHeight {
	return self.settleTimeout
}

func (self *PaymentChannel) CloseBlockHeight() typing.BlockHeight {

	//[NOTE] should not care about close block number!
	// can get it by fitler log if needed
	return 0
}

func (self *PaymentChannel) Opened() bool {
	return self.TokenNetwork.ChannelIsOpened(self.Participant1, self.Participant2,
		selfIdentifier)
}

func (self *PaymentChannel) Closed() bool {
	return self.TokenNetwork.ChannelIsClosed(self.Participant1, self.Participant2,
		selfIdentifier)
}

func (self *PaymentChannel) Settled() bool {
	return self.TokenNetwork.ChannelIsSettled(self.Participant1, self.Participant2,
		selfIdentifier)
}

func (self *PaymentChannel) ClosingAddress() typing.Address {
	return self.TokenNetwork.ClosingAddress(self.Participant1, self.Participant2,
		selfIdentifier)
}

func (self *PaymentChannel) CanTransfer() bool {
	return self.TokenNetwork.CanTransfer(self.Participant1, self.Participant2,
		selfIdentifier)
}

func (self *PaymentChannel) SetTotalDeposit(totalDeposit typing.TokenAmount) {

	self.TokenNetwork.SetTotalDeposit(selfIdentifier, totalDeposit,
		self.Participant2)

	return
}

func (self *PaymentChannel) Close(nonce typing.Nonce, balanceHash typing.BalanceHash,
	additionalHash typing.AdditionalHash, signature typing.Signature, pubKey typing.PubKey) {

	self.TokenNetwork.Close(selfIdentifier, self.Participant2,
		balanceHash, nonce, additionalHash, signature, pubKey)

	return
}

func (self *PaymentChannel) UpdateTransfer(nonce typing.Nonce, balanceHash typing.BalanceHash,
	additionalHash typing.AdditionalHash, partnerSignature typing.Signature,
	signature typing.Signature, closePubkey, nonClosePubkey typing.PubKey) {

	self.TokenNetwork.updateTransfer(selfIdentifier, self.Participant2,
		balanceHash, nonce, additionalHash, partnerSignature, signature, closePubkey, nonClosePubkey)

	return
}

func (self *PaymentChannel) Unlock(merkleTreeLeaves *list.List) {
	self.TokenNetwork.unlock(selfIdentifier, self.Participant2, merkleTreeLeaves)
	return
}

func (self *PaymentChannel) Settle(transferredAmount typing.TokenAmount, lockedAmount typing.TokenAmount, locksroot typing.Locksroot,
	partnerTransferredAmount typing.TokenAmount, partnerLockedAmount typing.TokenAmount, partnerLocksroot typing.Locksroot) {

	self.TokenNetwork.settle(selfIdentifier, transferredAmount, lockedAmount,
		locksroot, self.Participant2, partnerTransferredAmount, partnerLockedAmount, partnerLocksroot)

	return

}

func (self *PaymentChannel) GetBalance() (typing.Balance, error) {
	return self.TokenNetwork.GetBalance()
}

//Not needed
func (self PaymentChannel) AllEventsFilter(fromBlock typing.BlockHeight,
	toBlock typing.BlockHeight) {

	return
}
