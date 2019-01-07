package proxies

import (
	"bytes"
	"container/list"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/oniio/oniChannel/network/rpc"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

const settlementTimeoutMin int = 1000
const settlementTimeoutMax int = 100000

type ChannelData struct {
	ChannelIdentifier typing.ChannelID
	SettleBlockHeight typing.BlockHeight
	State             int
}

type ParticipantDetails struct {
	Address      typing.Address
	Deposit      typing.TokenAmount
	Withdrawn    typing.TokenAmount
	isCloser     bool
	BalanceHash  typing.BalanceHash
	Nonce        typing.Nonce
	Locksroot    typing.Locksroot
	LockedAmount typing.TokenAmount
}

type ParticipantsDetails struct {
	OurDetails     *ParticipantDetails
	PartnerDetails *ParticipantDetails
}

type ChannelDetails struct {
	chainId          typing.ChainID
	channelData      *ChannelData
	ParticipantsData *ParticipantsDetails
}

type TokenNetwork struct {
	Address                 typing.Address
	proxy                   *rpc.ContractProxy
	nodeAddress             typing.Address
	openLock                sync.Mutex
	openChannelTransactions map[typing.Address]*sync.Mutex
	opLock                  sync.Mutex
	channelOperationsLock   map[typing.Address]*sync.Mutex
	depositLock             sync.Mutex
}

func NewTokenNetwork(
	jsonrpcClient *rpc.RpcClient,
	ManagerAddress typing.Address) *TokenNetwork {
	self := new(TokenNetwork)

	self.Address = ManagerAddress
	self.proxy = rpc.NewContractProxy(jsonrpcClient)
	self.nodeAddress = typing.Address(jsonrpcClient.Account.Address)

	self.openChannelTransactions = make(map[typing.Address]*sync.Mutex)
	selfOperationsLock = make(map[typing.Address]*sync.Mutex)

	return self
}

func (self *TokenNetwork) getOperationLock(partner typing.Address) *sync.Mutex {
	var result *sync.Mutex

	self.opLock.Lock()
	defer self.opLock.Unlock()

	val, exist := selfOperationsLock[partner]

	if exist == false {
		newOpLock := new(sync.Mutex)

		selfOperationsLock[partner] = newOpLock
		result = newOpLock
	} else {
		result = val
	}

	return result

}

func (self *TokenNetwork) TokenAddress() typing.Address {
	var result typing.Address

	return result
}

func (self *TokenNetwork) NewNettingChannel(partner typing.Address, settleTimeout int) typing.ChannelID {
	var channelIdentifier typing.ChannelID

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

		txHash := self.newNettingChannel(partner, settleTimeout)
		fmt.Printf("new netting tx:%s\n", txHash)
		confirmed, err := self.proxy.JsonrpcClient.PollForTxConfirmed(time.Duration(60)*time.Second, txHash)
		if err != nil || !confirmed {
			log.Errorf("The new netting channel tx failed", err)
			return 0
		}

	} else {
		self.openLock.Unlock()

		val.Lock()
		defer val.Unlock()
	}

	channelCreated := selfExistsAndNotSettled(self.nodeAddress, partner, 0)
	if channelCreated == true {
		channelDetail := self.detailChannel(self.nodeAddress, partner, 0)
		channelIdentifier = channelDetail.ChannelIdentifier
	}

	return channelIdentifier
}

func (self *TokenNetwork) newNettingChannel(partner typing.Address, settleTimeout int) []byte {
	hash, err := self.proxy.OpenChannel(common.Address(self.nodeAddress), common.Address(partner), uint64(settleTimeout))
	if err != nil {
		log.Errorf("new netting channel err:%s", err)
	}
	return hash
}

func (self *TokenNetwork) inspectChannelIdentifier(participant1 typing.Address,
	participant2 typing.Address, caller string,
	channelIdentifier typing.ChannelID) typing.ChannelID {
	// need getChannelIdentifier api
	// result := self.callAndCheckResult("getChannelIdentifier", participant1, participant2)
	// return result.(typing.ChannelID)
	id, err := self.proxy.GetChannelIdentifier(common.Address(participant1), common.Address(participant2))
	if err != nil {
		log.Errorf("Get channelIdentifier err:%s", err)
		return 0
	}
	return typing.ChannelID(id)
}

func (self *TokenNetwork) channelExistsAndNotSettled(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

	var existsAndNotSettled bool

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)

	if channelState > micropayment.NonExistent && channelState < micropayment.Settled {
		existsAndNotSettled = true
	}

	return existsAndNotSettled
}

func (self *TokenNetwork) detailParticipant(channelIdentifier typing.ChannelID,
	participant typing.Address, partner typing.Address) *ParticipantDetails {
	info, err := self.proxy.GetChannelParticipantInfo(uint64(channelIdentifier), common.Address(participant), common.Address(partner))
	if err != nil {
		log.Errorf("GetChannelParticipantInfo err:%s", err)
		return nil
	}
	result := &ParticipantDetails{
		Address:     typing.Address(info.WalletAddr),
		Deposit:     typing.TokenAmount(info.Deposit),
		Withdrawn:   typing.TokenAmount(info.WithDrawAmount),
		isCloser:    info.IsCloser,
		BalanceHash: typing.BalanceHash(info.BalanceHash),
		Nonce:       typing.Nonce(info.Nonce),

		// TODO: this two field is missing
		// Locksroot:    ,
		// LockedAmount: ,
	}
	return result
}

func (self *TokenNetwork) detailChannel(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) *ChannelData {

	if channelIdentifier == 0 {
		channelIdentifier = self.inspectChannelIdentifier(participant1, participant2,
			"detailChannel", channelIdentifier)
	}

	info, err := self.proxy.GetChannelInfo(uint64(channelIdentifier), common.Address(participant1), common.Address(participant2))
	if err != nil {
		log.Errorf("GetChannelInfo err:%s", err)
		return nil
	}
	channelData := &ChannelData{
		ChannelIdentifier: channelIdentifier,
		SettleBlockHeight: typing.BlockHeight(info.SettleBlockHeight),
		State:             int(info.ChannelState),
	}
	return channelData
}

func (self *TokenNetwork) DetailParticipants(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) *ParticipantsDetails {

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

func (self *TokenNetwork) detail(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) *ChannelDetails {

	if self.nodeAddress != participant1 && self.nodeAddress != participant2 {
		return nil
	}

	if self.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	channelData := self.detailChannel(participant1, participant2, channelIdentifier)
	participantsData := self.DetailParticipants(participant1, participant2, channelIdentifier)

	channelDetails := new(ChannelDetails)
	channelDetails.chainId = 0
	channelDetails.channelData = channelData
	channelDetails.ParticipantsData = participantsData

	return channelDetails
}

func (self *TokenNetwork) settlementTimeoutMin() int {
	//Hard code it, can support it in contract if needed
	return settlementTimeoutMin
}

func (self *TokenNetwork) settlementTimeoutMax() int {
	//Hard code it, can support it in contract if needed
	return settlementTimeoutMax
}

func (self *TokenNetwork) ChannelIsOpened(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Opened
}

func (self *TokenNetwork) ChannelIsClosed(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Closed
}

func (self *TokenNetwork) ChannelIsSettled(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelIdentifier)
	return channelState == micropayment.Settled
}

func (self *TokenNetwork) ClosingAddress(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) typing.Address {

	var result typing.Address

	channelData := self.detailChannel(participant1, participant2, channelIdentifier)
	participantsData := self.DetailParticipants(participant1, participant2, channelData.ChannelIdentifier)

	if participantsData.OurDetails.isCloser {
		return participantsData.OurDetails.Address
	} else if participantsData.PartnerDetails.isCloser {
		return participantsData.PartnerDetails.Address
	}

	return result

}

func (self *TokenNetwork) CanTransfer(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

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

func (self *TokenNetwork) GetBalance() (typing.Balance, error) {

	bal, err := self.proxy.GetOntBalance(common.Address(self.nodeAddress))
	currentBalance := typing.Balance(bal)
	if err != nil {
		log.Errorf("GetOntBalance err:%s", err)
		return currentBalance, err
	}

	return currentBalance, nil
}

func (self *TokenNetwork) SetTotalDeposit(channelIdentifier typing.ChannelID,
	totalDeposit typing.TokenAmount, partner typing.Address) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
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

	if totalDeposit < currentDeposit {
		log.Errorf("SetTotalDeposit failed, Current total deposit %d is already larger than the requested total deposit amount %d",
			currentDeposit, totalDeposit)
		return
	} else if amountToDeposit <= 0 {
		log.Errorf("SetTotalDeposit failed, new_total_deposit - previous_total_deposit must be greater than 0",
			currentDeposit, totalDeposit)
		return
	}

	//call ont api to get balance of wallet!
	var currentBalance typing.TokenAmount
	bal, err := self.proxy.GetOntBalance(common.Address(self.nodeAddress))
	currentBalance = typing.TokenAmount(bal)
	if err != nil {
		log.Errorf("GetOntBalance err:%s", err)
		return
	}
	if currentBalance < amountToDeposit {
		log.Errorf("SetTotalDeposit failed, new_total_deposit - previous_total_deposit = %d can not be larger than the available balance %d",
			amountToDeposit, currentBalance)
		return
	}

	txHash, err := self.proxy.SetTotalDeposit(uint64(channelIdentifier), common.Address(self.nodeAddress), common.Address(partner), uint64(totalDeposit))
	if err != nil {
		log.Errorf("SetTotalDeposit err:%s", err)
		return
	}
	log.Infof("SetTotalDeposit tx hash:%v\n", txHash)
	_, err = self.proxy.JsonrpcClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SetTotalDeposit  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForDeposit(self.nodeAddress, partner, channelIdentifier, totalDeposit)
		return
	}

	return
}

func (self *TokenNetwork) Close(channelIdentifier typing.ChannelID, partner typing.Address,
	balanceHash typing.BalanceHash, nonce typing.Nonce, additionalHash typing.AdditionalHash,
	signature typing.Signature, partnerPubKey typing.PubKey) {

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

	txHash, err := self.proxy.CloseChannel(uint64(channelIdentifier), common.Address(self.nodeAddress), common.Address(partner), []byte(balanceHash), uint64(nonce), []byte(additionalHash), []byte(signature), []byte(partnerPubKey))
	if err != nil {
		log.Errorf("CloseChannel err:%s", err)
		return
	}
	log.Infof("CloseChannel tx hash:%s\n", txHash)
	_, err = self.proxy.JsonrpcClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("CloseChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForClose(self.nodeAddress, partner, channelIdentifier)

		return
	}

	return
}

func (self *TokenNetwork) updateTransfer(channelIdentifier typing.ChannelID, partner typing.Address,
	balanceHash typing.BalanceHash, nonce typing.Nonce, additionalHash typing.AdditionalHash,
	closingSignature typing.Signature, nonClosingSignature typing.Signature, closePubKey typing.PubKey, nonClosePubKey typing.PubKey) {

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelIdentifier) == false {
		return
	}

	// need pubkey
	txHash, err := self.proxy.UpdateNonClosingBalanceProof(uint64(channelIdentifier), common.Address(partner), common.Address(self.nodeAddress), []byte(balanceHash), uint64(nonce), []byte(additionalHash), []byte(closingSignature), []byte(nonClosingSignature), closePubKey, nonClosePubKey)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof err:%s", err)
		return
	}
	log.Infof("UpdateNonClosingBalanceProof tx hash:%s\n", txHash)
	_, err = self.proxy.JsonrpcClient.PollForTxConfirmed(time.Minute, txHash)
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

func (self *TokenNetwork) withDraw(channelIdentifier typing.ChannelID, partner typing.Address,
	totalWithdraw typing.TokenAmount, partnerSignature typing.Signature, partnerPubKey typing.PubKey, signature typing.Signature, pubKey typing.PubKey) {

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

	txHash, err := self.proxy.SetTotalWithdraw(uint64(channelIdentifier), common.Address(self.nodeAddress), common.Address(partner), uint64(totalWithdraw), []byte(signature), pubKey, []byte(partnerSignature), partnerPubKey)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof err:%s", err)
		return
	}
	log.Infof("UpdateNonClosingBalanceProof tx hash:%s\n", txHash)
	_, err = self.proxy.JsonrpcClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForWithdraw(self.nodeAddress, partner, channelIdentifier, totalWithdraw)

		return
	}

	return
}

func (self *TokenNetwork) unlock(channelIdentifier typing.ChannelID, partner typing.Address,
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

func (self *TokenNetwork) settle(channelIdentifier typing.ChannelID, transferredAmount typing.TokenAmount,
	lockedAmount typing.TokenAmount, locksroot typing.Locksroot, partner typing.Address,
	partnerTransferredAmount typing.TokenAmount, partnerLockedAmount typing.TokenAmount, partnerLocksroot typing.Locksroot) {

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
	var locksrootSlice []byte
	var partnerLocksrootSlice []byte

	if typing.LocksrootEmpty(locksroot) {
		locksrootSlice = nil
	} else {
		locksrootSlice = locksroot[:]
	}

	if typing.LocksrootEmpty(partnerLocksroot) {
		partnerLocksrootSlice = nil
	} else {
		partnerLocksrootSlice = partnerLocksroot[:]
	}

	if ourBpIsLarger {

		txHash, err = self.proxy.SettleChannel(uint64(channelIdentifier),
			common.Address(partner),
			uint64(partnerTransferredAmount),
			uint64(partnerLockedAmount),
			partnerLocksrootSlice,
			common.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksrootSlice,
		)

	} else {

		txHash, err = self.proxy.SettleChannel(uint64(channelIdentifier),
			common.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksrootSlice,
			common.Address(partner),
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
	_, err = self.proxy.JsonrpcClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SettleChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForSettle(self.nodeAddress, partner, channelIdentifier)
		return
	}

	return
}

// func (self *TokenNetwork) eventsFilter(topics *list.List, fromBlock typing.BlockHeight,
// 	toBlock typing.BlockHeight) *utils.StatelessFilter {

// 	//[TODO] call self.client.new_filter
// 	result := new(utils.StatelessFilter)

// 	return result
// }

// func (self *TokenNetwork) AllEventsFilter(fromBlock typing.BlockHeight,
// 	toBlock typing.BlockHeight) *utils.StatelessFilter {

// 	return self.eventsFilter(nil, fromBlock, toBlock)
// }

func (self *TokenNetwork) CheckForOutdatedChannel(participant1 typing.Address, participant2 typing.Address,
	channelIdentifier typing.ChannelID) bool {

	onchainChannelDetails := self.detailChannel(participant1, participant2, 0)
	onchainChannelIdentifier := onchainChannelDetails.ChannelIdentifier
	if onchainChannelIdentifier != channelIdentifier {
		return false
	} else {
		return true
	}
}

func (self *TokenNetwork) getChannelState(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) int {

	channelDate := self.detailChannel(participant1, participant2, channelIdentifier)
	if channelDate != nil {
		return channelDate.State
	}

	// channelData is nil when call contract with proxy failed, actually we are not sure about the real channel state in the contract
	return micropayment.NonExistent
}

func (self *TokenNetwork) checkChannelStateForClose(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

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

func (self *TokenNetwork) checkChannelStateForDeposit(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID,
	depositAmount typing.TokenAmount) bool {

	participantDetails := self.DetailParticipants(participant1, participant2,
		channelIdentifier)

	channelState := self.getChannelState(self.nodeAddress, participant2, channelIdentifier)
	if channelState == micropayment.NonExistent || channelState == micropayment.Removed {
		return false
	} else if channelState == micropayment.Settled {
		return false
	} else if channelState == micropayment.Closed {
		return false
	} else if participantDetails.OurDetails.Deposit < depositAmount {
		return false
	}

	return true
}

func (self *TokenNetwork) checkChannelStateForWithdraw(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID,
	withdrawAmount typing.TokenAmount) bool {

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

func (self *TokenNetwork) checkChannelStateForSettle(participant1 typing.Address,
	participant2 typing.Address, channelIdentifier typing.ChannelID) bool {

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
		h, err := self.proxy.JsonrpcClient.GetCurrentBlockHeight()
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
