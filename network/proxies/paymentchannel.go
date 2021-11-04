package proxies

import (
	"errors"
	"sync"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

type PaymentChannel struct {
	TokenNetwork     *TokenNetwork
	channelId        common.ChannelID
	Participant1     common.Address
	Participant2     common.Address
	openBlockHeight  common.BlockHeight
	settleTimeout    common.BlockHeight
	closeBlockHeight common.BlockHeight
}

func NewPaymentChannel(tokenNetwork *TokenNetwork, channelId common.ChannelID, args map[string]interface{}) (*PaymentChannel, error) {
	var participant1, participant2 common.Address

	//[NOTE] only new opened channel will really execute this function.
	//existing PaymentChannel for existed channel should be found in BlockChainService

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
		switch v.(type) {
		case common.BlockHeight:
			self.settleTimeout = v.(common.BlockHeight)
		case common.BlockTimeout:
			self.settleTimeout = common.BlockHeight(v.(common.BlockTimeout))
		}
	}

	self.channelId = channelId

	self.Participant1 = participant1
	self.Participant2 = participant2
	self.TokenNetwork = tokenNetwork

	return self, nil
}

func (self *PaymentChannel) GetChannelId() common.ChannelID {
	return self.channelId
}

func (self *PaymentChannel) LockOrRaise() *sync.Mutex {
	opLock := self.TokenNetwork.getOperationLock(self.Participant2)

	return opLock
}

func (self *PaymentChannel) tokenAddress() common.Address {
	return self.TokenNetwork.TokenAddress()
}

func (self *PaymentChannel) Detail() *ChannelDetails {
	return self.TokenNetwork.detail(self.Participant1, self.Participant2, self.channelId)
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
		self.channelId)
}

func (self *PaymentChannel) Closed() bool {
	return self.TokenNetwork.ChannelIsClosed(self.Participant1, self.Participant2,
		self.channelId)
}

func (self *PaymentChannel) Settled() bool {
	return self.TokenNetwork.ChannelIsSettled(self.Participant1, self.Participant2,
		self.channelId)
}

func (self *PaymentChannel) ClosingAddress() common.Address {
	return self.TokenNetwork.ClosingAddress(self.Participant1, self.Participant2,
		self.channelId)
}

func (self *PaymentChannel) CanTransfer() bool {
	return self.TokenNetwork.CanTransfer(self.Participant1, self.Participant2,
		self.channelId)
}

func (self *PaymentChannel) SetTotalDeposit(totalDeposit common.TokenAmount) error {
	return self.TokenNetwork.SetTotalDeposit(self.channelId, totalDeposit, self.Participant2)
}

func (self *PaymentChannel) Close(nonce common.Nonce, balanceHash common.BalanceHash,
	additionalHash common.AdditionalHash, signature common.Signature, pubKey common.PubKey) {

	self.TokenNetwork.Close(self.channelId, self.Participant2,
		balanceHash, nonce, additionalHash, signature, pubKey)

	return
}

func (self *PaymentChannel) Withdraw(partner common.Address, totalWithdraw common.TokenAmount,
	partnerSignature common.Signature, partnerPubKey common.PubKey, signature common.Signature, pubKey common.PubKey) error {

	return self.TokenNetwork.withDraw(self.channelId, partner, totalWithdraw, partnerSignature, partnerPubKey, signature, pubKey)
}

func (self *PaymentChannel) CooperativeSettle(participant1 common.Address, participant1Balance common.TokenAmount, participant2 common.Address, participant2Balance common.TokenAmount,
	participant1Signature common.Signature, participant1PubKey common.PubKey, participant2Signature common.Signature, participant2PubKey common.PubKey) error {

	return self.TokenNetwork.cooperativeSettle(self.channelId, participant1, participant1Balance,
		participant2, participant2Balance, participant1Signature, participant1PubKey, participant2Signature, participant2PubKey)
}

func (self *PaymentChannel) UpdateTransfer(nonce common.Nonce, balanceHash common.BalanceHash,
	additionalHash common.AdditionalHash, partnerSignature common.Signature,
	signature common.Signature, closePubkey, nonClosePubkey common.PubKey) {

	self.TokenNetwork.updateTransfer(self.channelId, self.Participant2,
		balanceHash, nonce, additionalHash, partnerSignature, signature, closePubkey, nonClosePubkey)

	return
}

func (self *PaymentChannel) Unlock(merkleTreeLeaves []*transfer.HashTimeLockState) {
	self.TokenNetwork.unlock(self.channelId, self.Participant2, merkleTreeLeaves)
	return
}

func (self *PaymentChannel) Settle(transferredAmount common.TokenAmount, lockedAmount common.TokenAmount, locksRoot common.LocksRoot,
	partnerTransferredAmount common.TokenAmount, partnerLockedAmount common.TokenAmount, partnerLocksroot common.LocksRoot) {

	self.TokenNetwork.settle(self.channelId, transferredAmount, lockedAmount,
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
