package proxies

import (
	"container/list"
	"errors"
	"sync"

	"github.com/oniio/oniChannel/common"
)

type PaymentChannel struct {
	TokenNetwork      *TokenNetwork
	channelIdentifier common.ChannelID
	Participant1      common.Address
	Participant2      common.Address
	openBlockHeight   common.BlockHeight
	settleTimeout     common.BlockHeight
	closeBlockHeight  common.BlockHeight
}

func NewPaymentChannel(tokenNetwork *TokenNetwork, channelIdentifier common.ChannelID, args map[string]interface{}) (*PaymentChannel, error) {
	var participant1, participant2 common.Address

	//[NOTE] only new opened channel will really execute this function.
	//existing PaymentChannel for existed channel should be found in BlockchainService

	self := new(PaymentChannel)
	if v, exist := args["participant1"]; exist {
		participant1 = v.(common.Address)
	}
	if v, exist := args["participant2"]; exist {
		participant2 = v.(common.Address)
	}

	if tokenNetwork.nodeAddress != participant1 && tokenNetwork.nodeAddress != participant2 {
		return nil, errors.New("One participant must be the node address")
	}

	if tokenNetwork.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	if v, exist := args["blockHeight"]; exist {
		self.openBlockHeight = v.(common.BlockHeight)
	}
	if v, exist := args["settleTimeout"]; exist {
		self.settleTimeout = common.BlockHeight(v.(common.BlockTimeout))
	}

	self.channelIdentifier = channelIdentifier

	self.Participant1 = participant1
	self.Participant2 = participant2
	self.TokenNetwork = tokenNetwork

	return self, nil
}

func (self *PaymentChannel) GetChannelId() common.ChannelID {
	return self.channelIdentifier
}

func (self *PaymentChannel) LockOrRaise() *sync.Mutex {
	opLock := self.TokenNetwork.getOperationLock(self.Participant2)

	return opLock
}

func (self *PaymentChannel) tokenAddress() common.Address {
	return self.TokenNetwork.TokenAddress()
}

func (self *PaymentChannel) Detail() *ChannelDetails {
	return self.TokenNetwork.detail(self.Participant1, self.Participant2, self.channelIdentifier)
}

//Should be set when open channel, should NOT get it by filter log
func (self PaymentChannel) SettleTimeout() common.BlockHeight {
	return self.settleTimeout
}

func (self *PaymentChannel) CloseBlockHeight() common.BlockHeight {

	//[NOTE] should not care about close block number!
	// can get it by fitler log if needed
	return 0
}

func (self *PaymentChannel) Opened() bool {
	return self.TokenNetwork.ChannelIsOpened(self.Participant1, self.Participant2,
		self.channelIdentifier)
}

func (self *PaymentChannel) Closed() bool {
	return self.TokenNetwork.ChannelIsClosed(self.Participant1, self.Participant2,
		self.channelIdentifier)
}

func (self *PaymentChannel) Settled() bool {
	return self.TokenNetwork.ChannelIsSettled(self.Participant1, self.Participant2,
		self.channelIdentifier)
}

func (self *PaymentChannel) ClosingAddress() common.Address {
	return self.TokenNetwork.ClosingAddress(self.Participant1, self.Participant2,
		self.channelIdentifier)
}

func (self *PaymentChannel) CanTransfer() bool {
	return self.TokenNetwork.CanTransfer(self.Participant1, self.Participant2,
		self.channelIdentifier)
}

func (self *PaymentChannel) SetTotalDeposit(totalDeposit common.TokenAmount) error {
	return self.TokenNetwork.SetTotalDeposit(self.channelIdentifier, totalDeposit,
		self.Participant2)
}

func (self *PaymentChannel) Close(nonce common.Nonce, balanceHash common.BalanceHash,
	additionalHash common.AdditionalHash, signature common.Signature, pubKey common.PubKey) {

	self.TokenNetwork.Close(self.channelIdentifier, self.Participant2,
		balanceHash, nonce, additionalHash, signature, pubKey)

	return
}

func (self *PaymentChannel) UpdateTransfer(nonce common.Nonce, balanceHash common.BalanceHash,
	additionalHash common.AdditionalHash, partnerSignature common.Signature,
	signature common.Signature, closePubkey, nonClosePubkey common.PubKey) {

	self.TokenNetwork.updateTransfer(self.channelIdentifier, self.Participant2,
		balanceHash, nonce, additionalHash, partnerSignature, signature, closePubkey, nonClosePubkey)

	return
}

func (self *PaymentChannel) Unlock(merkleTreeLeaves *list.List) {
	self.TokenNetwork.unlock(self.channelIdentifier, self.Participant2, merkleTreeLeaves)
	return
}

func (self *PaymentChannel) Settle(transferredAmount common.TokenAmount, lockedAmount common.TokenAmount, locksRoot common.Locksroot,
	partnerTransferredAmount common.TokenAmount, partnerLockedAmount common.TokenAmount, partnerLocksroot common.Locksroot) {

	self.TokenNetwork.settle(self.channelIdentifier, transferredAmount, lockedAmount,
		locksRoot, self.Participant2, partnerTransferredAmount, partnerLockedAmount, partnerLocksroot)

	return

}

func (self *PaymentChannel) GetGasBalance() (common.TokenAmount, error) {
	return self.TokenNetwork.GetGasBalance()
}

//Not needed
func (self PaymentChannel) AllEventsFilter(fromBlock common.BlockHeight,
	toBlock common.BlockHeight) {

	return
}
