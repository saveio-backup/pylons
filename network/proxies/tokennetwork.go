package proxies

import (
	"bytes"
	"container/list"
	"errors"
	"strconv"
	"sync"
	"time"

	chainsdk "github.com/oniio/oniChain-go-sdk"
	chnsdk "github.com/oniio/oniChain-go-sdk/channel"
	comm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

type ChannelData struct {
	ChannelIdentifier common.ChannelID
	SettleBlockHeight common.BlockHeight
	State             int
}

type ParticipantDetails struct {
	Address      common.Address
	Deposit      common.TokenAmount
	Withdrawn    common.TokenAmount
	isCloser     bool
	BalanceHash  common.BalanceHash
	Nonce        common.Nonce
	Locksroot    common.Locksroot
	LockedAmount common.TokenAmount
}

type ParticipantsDetails struct {
	OurDetails     *ParticipantDetails
	PartnerDetails *ParticipantDetails
}

type ChannelDetails struct {
	ChainId          common.ChainID
	ChannelData      *ChannelData
	ParticipantsData *ParticipantsDetails
}

type TokenNetwork struct {
	Address                 common.Address
	ChainClient             *chainsdk.Chain
	ChannelClient           *chnsdk.Channel
	nodeAddress             common.Address
	openLock                sync.Mutex
	openChannelTransactions map[common.Address]*sync.Mutex
	opLock                  sync.Mutex
	channelOperationsLock   map[common.Address]*sync.Mutex
	depositLock             sync.Mutex
}

func NewTokenNetwork(
	chainClient *chainsdk.Chain,
	tokenAddress common.Address) *TokenNetwork {
	self := new(TokenNetwork)

	self.Address = tokenAddress
	self.ChainClient = chainClient
	self.ChannelClient = chainClient.Native.Channel
	self.nodeAddress = common.Address(self.ChannelClient.DefAcc.Address)

	self.openChannelTransactions = make(map[common.Address]*sync.Mutex)
	self.channelOperationsLock = make(map[common.Address]*sync.Mutex)

	return self
}

func (self *TokenNetwork) getOperationLock(partner common.Address) *sync.Mutex {
	var result *sync.Mutex

	self.opLock.Lock()
	defer self.opLock.Unlock()

	val, exist := self.channelOperationsLock[partner]

	if exist == false {
		newOpLock := new(sync.Mutex)

		self.channelOperationsLock[partner] = newOpLock
		result = newOpLock
	} else {
		result = val
	}

	return result

}

func (self *TokenNetwork) TokenAddress() common.Address {
	return self.Address
}

func (self *TokenNetwork) NewNettingChannel(partner common.Address, settleTimeout int) common.ChannelID {
	var channelIdentifier common.ChannelID

	if bytes.Compare(self.nodeAddress[:], partner[:]) == 0 {
		log.Errorf("The other peer must not have the same address as the client.")
		return 0
	}

	self.openLock.Lock()
	val, exist := self.openChannelTransactions[partner]

	if exist == false {
		newOpenChannelTransaction := new(sync.Mutex)
		newOpenChannelTransaction.Lock()
		defer newOpenChannelTransaction.Unlock()

		self.openChannelTransactions[partner] = newOpenChannelTransaction
		self.openLock.Unlock()

		txHash, err := self.newNettingChannel(partner, settleTimeout)
		if err != nil {
			log.Errorf("create channel txn failed", err.Error())
			return 0
		}
		confirmed, err := self.ChainClient.PollForTxConfirmed(time.Duration(60)*time.Second, txHash)
		if err != nil || !confirmed {
			log.Errorf("poll new netting channel tx failed", err.Error())
			return 0
		}

	} else {
		self.openLock.Unlock()

		val.Lock()
		defer val.Unlock()
	}

	channelCreated := self.channelExistsAndNotSettled(self.nodeAddress, partner, 0)
	if channelCreated == true {
		channelDetail := self.detailChannel(self.nodeAddress, partner, 0)
		channelIdentifier = channelDetail.ChannelIdentifier
	}

	return channelIdentifier
}

func (self *TokenNetwork) newNettingChannel(partner common.Address, settleTimeout int) ([]byte, error) {
	hash, err := self.ChannelClient.OpenChannel(comm.Address(self.nodeAddress), comm.Address(partner), uint64(settleTimeout))
	regAddr  := common.ToBase58(self.nodeAddress)
	patAddr := common.ToBase58(partner)
	if err != nil {
		log.Errorf("create new channel between failed:%s", regAddr, patAddr, err.Error())
		return nil, err
	}
	return hash, nil
}

func (self *TokenNetwork) inspectChannelIdentifier(participant1 common.Address,
	participant2 common.Address, caller string,
	channelIdentifier common.ChannelID) common.ChannelID {
	// need getChannelIdentifier api
	// result := self.callAndCheckResult("getChannelIdentifier", participant1, participant2)
	// return result.(common.ChannelID)
	id, err := self.ChannelClient.GetChannelIdentifier(comm.Address(participant1), comm.Address(participant2))
	if err != nil {
		log.Errorf("Get channelIdentifier err:%s", err)
		return 0
	}
	return common.ChannelID(id)
}

func (self *TokenNetwork) channelExistsAndNotSettled(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	var existsAndNotSettled bool

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)

	if channelState > micropayment.NonExistent && channelState < micropayment.Settled {
		existsAndNotSettled = true
	}

	return existsAndNotSettled
}

func (self *TokenNetwork) detailParticipant(channelIdentifier common.ChannelID,
	participant common.Address, partner common.Address) *ParticipantDetails {
	info, err := self.ChannelClient.GetChannelParticipantInfo(uint64(channelIdentifier), comm.Address(participant), comm.Address(partner))
	if err != nil {
		log.Errorf("GetChannelParticipantInfo err:%s", err)
		return nil
	}
	result := &ParticipantDetails{
		Address:     common.Address(info.WalletAddr),
		Deposit:     common.TokenAmount(info.Deposit),
		Withdrawn:   common.TokenAmount(info.WithDrawAmount),
		isCloser:    info.IsCloser,
		BalanceHash: common.BalanceHash(info.BalanceHash),
		Nonce:       common.Nonce(info.Nonce),

		// TODO: this two field is missing
		// Locksroot:    ,
		// LockedAmount: ,
	}
	return result
}

func (self *TokenNetwork) detailChannel(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) *ChannelData {

	if channelIdentifier == 0 {
		channelIdentifier = self.inspectChannelIdentifier(participant1, participant2,
			"detailChannel", channelIdentifier)
	}

	info, err := self.ChannelClient.GetChannelInfo(uint64(channelIdentifier), comm.Address(participant1), comm.Address(participant2))
	if err != nil {
		log.Errorf("GetChannelInfo err:%s", err)
		return nil
	}
	channelData := &ChannelData{
		ChannelIdentifier: channelIdentifier,
		SettleBlockHeight: common.BlockHeight(info.SettleBlockHeight),
		State:             int(info.ChannelState),
	}
	return channelData
}

func (self *TokenNetwork) DetailParticipants(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) *ParticipantsDetails {

	if self.nodeAddress != participant1 && self.nodeAddress != participant2 {
		return nil
	}

	if self.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	channelIdentifier = self.inspectChannelIdentifier(participant1, participant2,
		"detailParticipants", channelIdentifier)

	ourData := self.detailParticipant(channelIdentifier, participant1, participant2)
	partnerData := self.detailParticipant(channelIdentifier, participant2, participant1)

	participantsDetails := new(ParticipantsDetails)
	participantsDetails.OurDetails = ourData
	participantsDetails.PartnerDetails = partnerData
	return participantsDetails
}

func (self *TokenNetwork) detail(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) *ChannelDetails {

	if self.nodeAddress != participant1 && self.nodeAddress != participant2 {
		return nil
	}

	if self.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	channelData := self.detailChannel(participant1, participant2, channelIdentifier)
	participantsData := self.DetailParticipants(participant1, participant2, channelIdentifier)

	channelDetails := new(ChannelDetails)
	id, err := self.ChainClient.GetNetworkId()
	if err != nil {
		log.Warn("get network id failed,id set to 0", err)
		id = 0
	}
	channelDetails.ChainId = common.ChainID(id)
	channelDetails.ChannelData = channelData
	channelDetails.ParticipantsData = participantsData

	return channelDetails
}

func (self *TokenNetwork) ChannelIsOpened(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Opened
}

func (self *TokenNetwork) ChannelIsClosed(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Closed
}

func (self *TokenNetwork) ChannelIsSettled(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Settled
}

func (self *TokenNetwork) ClosingAddress(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) common.Address {

	var result common.Address

	channelData := self.detailChannel(participant1, participant2, channelIdentifier)
	participantsData := self.DetailParticipants(participant1, participant2, channelData.ChannelIdentifier)

	if participantsData.OurDetails.isCloser {
		return participantsData.OurDetails.Address
	} else if participantsData.PartnerDetails.isCloser {
		return participantsData.PartnerDetails.Address
	}

	return result

}

func (self *TokenNetwork) CanTransfer(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	if self.ChannelIsOpened(participant1, participant2, channelIdentifier) == false {
		return false
	}

	details := self.detailParticipant(channelIdentifier, participant1, participant2)
	if details.Deposit > 0 {
		return true
	} else {
		return false
	}

}

//token balance should divided by Decimals of token
func (self *TokenNetwork) GetGasBalance() (common.TokenAmount, error) {

	bal, err := self.ChainClient.Native.Ong.BalanceOf(comm.Address(self.nodeAddress))
	currentBalance := common.TokenAmount(bal)
	if err != nil {
		log.Errorf("GetGasBalance err:%s", err)
		return currentBalance, err
	}

	return currentBalance, nil
}

func (self *TokenNetwork) SetTotalDeposit(channelIdentifier common.ChannelID,
	totalDeposit common.TokenAmount, partner common.Address) error {

	var opLock *sync.Mutex

	if !self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) {
		return errors.New("channel outdated")
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	//Avoid double spend!
	self.depositLock.Lock()
	defer self.depositLock.Unlock()

	detail := self.detailParticipant(channelIdentifier, self.nodeAddress, partner)

	currentDeposit := detail.Deposit
	amountToDeposit := totalDeposit - currentDeposit

	if totalDeposit <= currentDeposit {
		log.Warnf("current total deposit %d is already larger than the requested total deposit amount %d, skip",
			currentDeposit, totalDeposit)
		return nil
	}

	//call native api to get balance of wallet!
	currentBalance, err := self.GetGasBalance()
	if err != nil {
		return err
	}
	if currentBalance < amountToDeposit {
		log.Errorf("SetTotalDeposit failed, new_total_deposit - previous_total_deposit = %d can not be larger than the available balance %d",
			amountToDeposit, currentBalance)
		return errors.New("current balance not enough")
	}

	txHash, err := self.ChannelClient.SetTotalDeposit(uint64(channelIdentifier), comm.Address(self.nodeAddress), comm.Address(partner), uint64(totalDeposit))
	if err != nil {
		log.Errorf("SetTotalDeposit err:%s", err)
		return err
	}
	log.Infof("SetTotalDeposit tx hash:%v\n", txHash)
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SetTotalDeposit  WaitForGenerateBlock err:%s", err)
		ret, nerr := self.checkChannelStateForDeposit(self.nodeAddress, partner, channelIdentifier, totalDeposit)
		if !ret {
			return nerr
		}
		return err
	}

	return nil
}

func (self *TokenNetwork) Close(channelIdentifier common.ChannelID, partner common.Address,
	balanceHash common.BalanceHash, nonce common.Nonce, additionalHash common.AdditionalHash,
	signature common.Signature, partnerPubKey common.PubKey) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	if self.checkChannelStateForClose(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	txHash, err := self.ChannelClient.CloseChannel(uint64(channelIdentifier), comm.Address(self.nodeAddress), comm.Address(partner), []byte(balanceHash), uint64(nonce), []byte(additionalHash), []byte(signature), []byte(partnerPubKey))
	if err != nil {
		log.Errorf("CloseChannel err:%s", err)
		return
	}
	log.Infof("CloseChannel tx hash:%s\n", txHash)
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("CloseChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForClose(self.nodeAddress, partner, channelIdentifier)

		return
	}

	return
}

func (self *TokenNetwork) updateTransfer(channelIdentifier common.ChannelID, partner common.Address,
	balanceHash common.BalanceHash, nonce common.Nonce, additionalHash common.AdditionalHash,
	closingSignature common.Signature, nonClosingSignature common.Signature, closePubKey common.PubKey, nonClosePubKey common.PubKey) {

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	// need pubkey
	txHash, err := self.ChannelClient.UpdateNonClosingBalanceProof(uint64(channelIdentifier), comm.Address(partner), comm.Address(self.nodeAddress), []byte(balanceHash), uint64(nonce), []byte(additionalHash), []byte(closingSignature), []byte(nonClosingSignature), closePubKey, nonClosePubKey)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof err:%s", err)
		return
	}
	log.Infof("UpdateNonClosingBalanceProof tx hash:%s\n", txHash)
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof  WaitForGenerateBlock err:%s", err)
		channelClosed := self.ChannelIsClosed(self.nodeAddress, partner, channelIdentifier)
		if channelClosed == false {
			return
		}
		return
	}

	return
}

func (self *TokenNetwork) withDraw(channelIdentifier common.ChannelID, partner common.Address,
	totalWithdraw common.TokenAmount, partnerSignature common.Signature, partnerPubKey common.PubKey, signature common.Signature, pubKey common.PubKey) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	details := self.detailParticipant(channelIdentifier, self.nodeAddress, partner)
	currentWithdraw := details.Withdrawn
	amountToWithdraw := totalWithdraw - currentWithdraw

	if totalWithdraw < currentWithdraw {
		return
	}

	if amountToWithdraw <= 0 {
		return
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	txHash, err := self.ChannelClient.SetTotalWithdraw(uint64(channelIdentifier), comm.Address(self.nodeAddress), comm.Address(partner), uint64(totalWithdraw), []byte(signature), pubKey, []byte(partnerSignature), partnerPubKey)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof err:%s", err)
		return
	}
	log.Infof("UpdateNonClosingBalanceProof tx hash:%s\n", txHash)
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForWithdraw(self.nodeAddress, partner, channelIdentifier, totalWithdraw)

		return
	}

	return
}

func (self *TokenNetwork) unlock(channelIdentifier common.ChannelID, partner common.Address,
	merkleTreeLeaves *list.List) {

	if merkleTreeLeaves.Len() == 0 {
		return
	}

	var leavesPacked []byte

	for e := merkleTreeLeaves.Front(); e != nil; e = e.Next() {
		hashTimeLockState := e.Value.(*transfer.HashTimeLockState)
		leavesPacked = append(leavesPacked, hashTimeLockState.Encoded...)
	}

	//[TODO] call transaction_hash = self.proxy.transact("unlock"
	//[TODO] call client.poll
	// support this when support secret and route

	var receiptOrNone bool

	if receiptOrNone {
		channelSettled := self.ChannelIsSettled(self.nodeAddress, partner, channelIdentifier)
		if channelSettled == false {
			return
		}
	}

	return
}

func (self *TokenNetwork) settle(channelIdentifier common.ChannelID, transferredAmount common.TokenAmount,
	lockedAmount common.TokenAmount, locksRoot common.Locksroot, partner common.Address,
	partnerTransferredAmount common.TokenAmount, partnerLockedAmount common.TokenAmount, partnerLocksroot common.Locksroot) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	ourMaximum := transferredAmount + lockedAmount
	partnerMaximum := partnerTransferredAmount + partnerLockedAmount

	ourBpIsLarger := ourMaximum > partnerMaximum
	var txHash []byte
	var err error
	var locksRootSlice []byte
	var partnerLocksrootSlice []byte

	if common.LocksrootEmpty(locksRoot) {
		locksRootSlice = nil
	} else {
		locksRootSlice = locksRoot[:]
	}

	if common.LocksrootEmpty(partnerLocksroot) {
		partnerLocksrootSlice = nil
	} else {
		partnerLocksrootSlice = partnerLocksroot[:]
	}

	if ourBpIsLarger {

		txHash, err = self.ChannelClient.SettleChannel(uint64(channelIdentifier),
			comm.Address(partner),
			uint64(partnerTransferredAmount),
			uint64(partnerLockedAmount),
			partnerLocksrootSlice,
			comm.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksRootSlice,
		)

	} else {

		txHash, err = self.ChannelClient.SettleChannel(uint64(channelIdentifier),
			comm.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksRootSlice,
			comm.Address(partner),
			uint64(partnerTransferredAmount),
			uint64(partnerLockedAmount),
			partnerLocksrootSlice,
		)
	}

	if err != nil {
		log.Errorf("SettleChannel err:%s", err)
		return
	}
	log.Infof("SettleChannel tx hash:%s\n", txHash)
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SettleChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForSettle(self.nodeAddress, partner, channelIdentifier)
		return
	}

	return
}

// func (self *TokenNetwork) eventsFilter(topics *list.List, fromBlock common.BlockHeight,
// 	toBlock common.BlockHeight) *utils.StatelessFilter {

// 	//[TODO] call self.client.new_filter
// 	result := new(utils.StatelessFilter)

// 	return result
// }

// func (self *TokenNetwork) AllEventsFilter(fromBlock common.BlockHeight,
// 	toBlock common.BlockHeight) *utils.StatelessFilter {

// 	return self.eventsFilter(nil, fromBlock, toBlock)
// }

func (self *TokenNetwork) CheckForOutdatedChannel(participant1 common.Address, participant2 common.Address,
	channelIdentifier common.ChannelID) bool {

	onchainChannelDetails := self.detailChannel(participant1, participant2, 0)
	onchainChannelIdentifier := onchainChannelDetails.ChannelIdentifier
	if onchainChannelIdentifier != channelIdentifier {
		return false
	} else {
		return true
	}
}

func (self *TokenNetwork) getChannelState(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) int {

	channelDate := self.detailChannel(participant1, participant2, channelIdentifier)
	if channelDate != nil {
		return channelDate.State
	}

	// channelData is nil when call contract with proxy failed, actually we are not sure about the real channel state in the contract
	return micropayment.NonExistent
}

func (self *TokenNetwork) checkChannelStateForClose(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	if channelState == micropayment.NonExistent || channelState == micropayment.Removed {
		return false
	} else if channelState == micropayment.Settled {
		return false
	} else if channelState == micropayment.Closed {
		return false
	}

	return true
}

func (self *TokenNetwork) checkChannelStateForDeposit(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID,
	depositAmount common.TokenAmount) (bool, error) {

	participantDetails := self.DetailParticipants(participant1, participant2,
		channelIdentifier)

	channelState := self.getChannelState(self.nodeAddress, participant2, channelIdentifier)
	if channelState == micropayment.NonExistent || channelState == micropayment.Removed {
		return false, errors.New("channel does not exist")
	} else if channelState == micropayment.Settled {
		return false, errors.New("deposit is not possible due to channel being settled")
	} else if channelState == micropayment.Closed {
		return false, errors.New("channel is already closed")
	} else if participantDetails.OurDetails.Deposit < depositAmount {
		return false, errors.New("deposit amount did not increase after deposit transaction")
	}

	return false, nil
}

func (self *TokenNetwork) checkChannelStateForWithdraw(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID,
	withdrawAmount common.TokenAmount) bool {

	participantDetails := self.DetailParticipants(participant1, participant2,
		channelIdentifier)

	if participantDetails.OurDetails.Withdrawn > withdrawAmount {
		return false
	}

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	if channelState == micropayment.NonExistent || channelState == micropayment.Removed {
		return false
	} else if channelState == micropayment.Settled {
		return false
	} else if channelState == micropayment.Closed {
		return false
	}

	return true
}

func (self *TokenNetwork) checkChannelStateForSettle(participant1 common.Address,
	participant2 common.Address, channelIdentifier common.ChannelID) bool {

	channelData := self.detailChannel(participant1, participant2, channelIdentifier)
	if channelData.State == micropayment.Settled {
		return false
	} else if channelData.State == micropayment.Removed {
		return false
	} else if channelData.State == micropayment.Opened {
		return false
	} else if channelData.State == micropayment.Closed {
		//[TODO] use rpc client to get block number
		//self.client.block_number() < channel_data.settle_block_numbe
		h, err := self.ChainClient.GetCurrentBlockHeight()
		if err != nil {
			log.Errorf("GetCurrentBlockHeight err :%s", err)
			return false
		}
		height, err := strconv.ParseUint(string(h), 10, 64)
		if err != nil {
			log.Errorf("ParseUint height :%s err :%s", h, err)
			return false
		}
		if height < uint64(channelData.SettleBlockHeight) {
			return false
		}
	}

	return true
}
