package proxies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/saveio/themis/crypto/signature"
	"strconv"
	"sync"
	"time"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/transfer"
	chainsdk "github.com/saveio/themis-go-sdk"
	chnsdk "github.com/saveio/themis-go-sdk/channel"
	comm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
	"github.com/saveio/themis/smartcontract/service/native/micropayment"
)

type ChannelData struct {
	ChannelId         common.ChannelID
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
	LocksRoot    common.LocksRoot
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
	openChannelTransactions map[common.Address]chan bool
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

	self.openChannelTransactions = make(map[common.Address]chan bool)
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
	var channelId common.ChannelID

	if bytes.Compare(self.nodeAddress[:], partner[:]) == 0 {
		log.Errorf("The other peer must not have the same address as the client.")
		return 0
	}

	self.openLock.Lock()
	val, exist := self.openChannelTransactions[partner]

	if exist == false {
		newOpenChannelTransaction := make(chan bool)
		defer func() {
			close(newOpenChannelTransaction)
			delete(self.openChannelTransactions, partner)
		}()

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

		<-val
	}

	channelCreated := self.channelExistsAndNotSettled(self.nodeAddress, partner, 0)
	if channelCreated == true {
		channelDetail := self.detailChannel(self.nodeAddress, partner, 0)
		channelId = channelDetail.ChannelId
	}

	return channelId
}

func (self *TokenNetwork) newNettingChannel(partner common.Address, settleTimeout int) ([]byte, error) {
	pubKey := keypair.SerializePublicKey(self.ChannelClient.DefAcc.PublicKey)
	log.Infof("[newNettingChannel] PubKey: %s", hex.EncodeToString(pubKey))
	hash, err := self.ChannelClient.OpenChannel(comm.Address(self.nodeAddress), pubKey, comm.Address(partner), uint64(settleTimeout))
	regAddr := common.ToBase58(self.nodeAddress)
	patAddr := common.ToBase58(partner)
	if err != nil {
		log.Errorf("create new channel between failed:%s", regAddr, patAddr, err.Error())
		return nil, err
	}
	return hash, nil
}

func (self *TokenNetwork) inspectChannelId(participant1 common.Address,
	participant2 common.Address, caller string,
	channelId common.ChannelID) common.ChannelID {
	// need getChannelId api
	// result := self.callAndCheckResult("getChannelId", participant1, participant2)
	// return result.(common.ChannelID)
	id, err := self.ChannelClient.GetChannelIdentifier(comm.Address(participant1), comm.Address(participant2))
	if err != nil {
		log.Errorf("Get channelId err:%s", err)
		return 0
	}
	return common.ChannelID(id)
}

func (self *TokenNetwork) channelExistsAndNotSettled(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) bool {

	var existsAndNotSettled bool

	channelState := self.getChannelState(participant1, participant2, channelId)

	if channelState > micropayment.NonExistent && channelState < micropayment.Settled {
		existsAndNotSettled = true
	}

	return existsAndNotSettled
}

func (self *TokenNetwork) detailParticipant(channelId common.ChannelID,
	participant common.Address, partner common.Address) *ParticipantDetails {
	info, err := self.ChannelClient.GetChannelParticipantInfo(uint64(channelId), comm.Address(participant), comm.Address(partner))
	if err != nil {
		log.Errorf("GetChannelParticipantInfo err:%s", err)
		return nil
	}

	var balanceHash common.BalanceHash

	if len(info.BalanceHash) != 0 && len(info.BalanceHash) != constants.HashLen {
		log.Errorf("GetChannelParticipantInfo: length error for balance hash")
		return nil
	}

	if len(info.BalanceHash) == constants.HashLen {
		for index, value := range info.BalanceHash {
			balanceHash[index] = value
		}
	}

	var locksRoot common.LocksRoot

	if len(info.LocksRoot) != 0 && len(info.LocksRoot) != constants.HashLen {
		log.Errorf("GetChannelParticipantInfo: length error for balance hash")
		return nil
	}

	if len(info.LocksRoot) == constants.HashLen {
		for index, value := range info.LocksRoot {
			locksRoot[index] = value
		}
	}
	result := &ParticipantDetails{
		Address:      common.Address(info.WalletAddr),
		Deposit:      common.TokenAmount(info.Deposit),
		Withdrawn:    common.TokenAmount(info.WithDrawAmount),
		isCloser:     info.IsCloser,
		BalanceHash:  balanceHash,
		Nonce:        common.Nonce(info.Nonce),
		LocksRoot:    locksRoot,
		LockedAmount: common.TokenAmount(info.LockedAmount),

		// TODO: this two field is missing
		// locksRoot:    ,
		// LockedAmount: ,
	}
	return result
}

func (self *TokenNetwork) detailChannel(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) *ChannelData {

	if channelId == 0 {
		channelId = self.inspectChannelId(participant1, participant2,
			"detailChannel", channelId)
	}

	info, err := self.ChannelClient.GetChannelInfo(uint64(channelId), comm.Address(participant1), comm.Address(participant2))
	if err != nil {
		log.Errorf("GetChannelInfo err:%s", err)
		return nil
	}
	channelData := &ChannelData{
		ChannelId:         channelId,
		SettleBlockHeight: common.BlockHeight(info.SettleBlockHeight),
		State:             int(info.ChannelState),
	}
	return channelData
}

func (self *TokenNetwork) DetailParticipants(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) *ParticipantsDetails {

	if self.nodeAddress != participant1 && self.nodeAddress != participant2 {
		return nil
	}

	if self.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	// only query when there channel id is 0, after channel is settled, inspectChannelId returns 0
	// but we need to get the correct info for unlock when channel id is provided correctly
	if channelId == 0 {
		channelId = self.inspectChannelId(participant1, participant2,
			"detailParticipants", channelId)
	}

	ourData := self.detailParticipant(channelId, participant1, participant2)
	partnerData := self.detailParticipant(channelId, participant2, participant1)

	participantsDetails := new(ParticipantsDetails)
	participantsDetails.OurDetails = ourData
	participantsDetails.PartnerDetails = partnerData
	return participantsDetails
}

func (self *TokenNetwork) detail(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) *ChannelDetails {

	if self.nodeAddress != participant1 && self.nodeAddress != participant2 {
		return nil
	}

	if self.nodeAddress == participant2 {
		participant1, participant2 = participant2, participant1
	}

	channelData := self.detailChannel(participant1, participant2, channelId)
	participantsData := self.DetailParticipants(participant1, participant2, channelId)

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
	participant2 common.Address, channelId common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelId)
	return channelState == micropayment.Opened
}

func (self *TokenNetwork) ChannelIsClosed(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelId)
	return channelState == micropayment.Closed
}

func (self *TokenNetwork) ChannelIsSettled(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelId)
	return channelState == micropayment.Settled
}

func (self *TokenNetwork) ClosingAddress(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) common.Address {

	var result common.Address

	channelData := self.detailChannel(participant1, participant2, channelId)
	participantsData := self.DetailParticipants(participant1, participant2, channelData.ChannelId)

	if participantsData.OurDetails.isCloser {
		return participantsData.OurDetails.Address
	} else if participantsData.PartnerDetails.isCloser {
		return participantsData.PartnerDetails.Address
	}

	return result

}

func (self *TokenNetwork) CanTransfer(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) bool {

	if self.ChannelIsOpened(participant1, participant2, channelId) == false {
		return false
	}

	details := self.detailParticipant(channelId, participant1, participant2)
	if details.Deposit > 0 {
		return true
	} else {
		return false
	}
}

//token balance should divided by Decimals of token
func (self *TokenNetwork) GetGasBalance() (common.TokenAmount, error) {

	bal, err := self.ChainClient.Native.Usdt.BalanceOf(comm.Address(self.nodeAddress))
	currentBalance := common.TokenAmount(bal)
	if err != nil {
		log.Errorf("GetGasBalance err:%s", err)
		return currentBalance, err
	}

	return currentBalance, nil
}

func (self *TokenNetwork) SetTotalDeposit(channelId common.ChannelID,
	totalDeposit common.TokenAmount, partner common.Address) error {

	var opLock *sync.Mutex

	if !self.CheckForOutdatedChannel(self.nodeAddress, partner, channelId) {
		return errors.New("channel outdated")
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	//Avoid double spend!
	self.depositLock.Lock()
	defer self.depositLock.Unlock()

	detail := self.detailParticipant(channelId, self.nodeAddress, partner)

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

	txHash, err := self.ChannelClient.SetTotalDeposit(uint64(channelId), comm.Address(self.nodeAddress), comm.Address(partner), uint64(totalDeposit))
	if err != nil {
		log.Errorf("SetTotalDeposit err:%s", err)
		return err
	}
	log.Infof("SetTotalDeposit tx hash:%v\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SetTotalDeposit  WaitForGenerateBlock err:%s", err)
		ret, nerr := self.checkChannelStateForDeposit(self.nodeAddress, partner, channelId, totalDeposit)
		if !ret {
			return nerr
		}
		return err
	}

	return nil
}

func (self *TokenNetwork) Close(channelId common.ChannelID, partner common.Address,
	balanceHash common.BalanceHash, nonce common.Nonce, additionalHash common.AdditionalHash,
	signature common.Signature, partnerPubKey common.PubKey) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelId) == false {
		return
	}

	if self.checkChannelStateForClose(self.nodeAddress, partner, channelId) == false {
		return
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	txHash, err := self.ChannelClient.CloseChannel(uint64(channelId), comm.Address(self.nodeAddress), comm.Address(partner), balanceHash[:], uint64(nonce), []byte(additionalHash), []byte(signature), []byte(partnerPubKey))
	if err != nil {
		log.Errorf("CloseChannel err:%s", err)
		return
	}
	log.Infof("CloseChannel tx hash:%s\n", getTxHashString(txHash))

	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("CloseChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForClose(self.nodeAddress, partner, channelId)
		return
	}

	return
}

func (self *TokenNetwork) updateTransfer(channelId common.ChannelID, partner common.Address,
	balanceHash common.BalanceHash, nonce common.Nonce, additionalHash common.AdditionalHash,
	closingSignature common.Signature, nonClosingSignature common.Signature, closePubKey common.PubKey, nonClosePubKey common.PubKey) {

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelId) == false {
		return
	}

	// need pubkey
	txHash, err := self.ChannelClient.UpdateNonClosingBalanceProof(uint64(channelId), comm.Address(partner), comm.Address(self.nodeAddress), balanceHash[:], uint64(nonce), []byte(additionalHash), []byte(closingSignature), []byte(nonClosingSignature), closePubKey, nonClosePubKey)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof err:%s", err)
		return
	}
	log.Infof("UpdateNonClosingBalanceProof tx hash:%s\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("UpdateNonClosingBalanceProof  WaitForGenerateBlock err:%s", err)
		channelClosed := self.ChannelIsClosed(self.nodeAddress, partner, channelId)
		if channelClosed == false {
			return
		}
		return
	}

	return
}

func (self *TokenNetwork) withDraw(channelId common.ChannelID, partner common.Address,
	totalWithdraw common.TokenAmount, partnerSignature common.Signature, partnerPubKey common.PubKey, signature common.Signature, pubKey common.PubKey) error {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelId) == false {
		return errors.New("channel out dated")
	}

	opLock = self.getOperationLock(partner)
	opLock.Lock()
	defer opLock.Unlock()

	txHash, err := self.ChannelClient.SetTotalWithdraw(uint64(channelId), comm.Address(self.nodeAddress), comm.Address(partner), uint64(totalWithdraw), []byte(signature), pubKey, []byte(partnerSignature), partnerPubKey)
	if err != nil {
		log.Errorf("SetTotoalWithdraw err:%s", err)
		return err
	}
	log.Infof("SetTotalWithdraw tx hash:%s\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SetTotoalWithdraw WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForWithdraw(self.nodeAddress, partner, channelId, totalWithdraw)

		return err
	}

	return nil
}

func (self *TokenNetwork) cooperativeSettle(channelId common.ChannelID, participant1 common.Address,
	participant1Balance common.TokenAmount, participant2 common.Address, participant2Balance common.TokenAmount,
	participant1Signature common.Signature, participant1PubKey common.PubKey, participant2Signature common.Signature, participant2PubKey common.PubKey) error {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, participant2, channelId) == false {
		return errors.New("channel out dated")
	}

	opLock = self.getOperationLock(participant2)
	opLock.Lock()
	defer opLock.Unlock()

	txHash, err := self.ChannelClient.CooperativeSettle(uint64(channelId), comm.Address(participant1), uint64(participant1Balance),
		comm.Address(participant2), uint64(participant2Balance), []byte(participant1Signature), participant1PubKey, []byte(participant2Signature), participant2PubKey)
	if err != nil {
		log.Errorf("CooperativeSettle err:%s", err)
		return err
	}
	log.Infof("CooperativeSettle tx hash:%s\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("CooperativeSettle WaitForGenerateBlock err:%s", err)
		return err
	}

	return nil
}
func (self *TokenNetwork) unlock(channelId common.ChannelID, partner common.Address,
	merkleTreeLeaves []*transfer.HashTimeLockState) {

	if len(merkleTreeLeaves) == 0 {
		return
	}

	var leavesPacked []byte

	for _, lock := range merkleTreeLeaves {
		leavesPacked = append(leavesPacked, lock.PackData()...)
	}

	txHash, err := self.ChannelClient.Unlock(uint64(channelId), comm.Address(self.nodeAddress), comm.Address(partner), leavesPacked)
	if err != nil {
		log.Errorf("Unlock err:%s", err)
		return
	}
	log.Infof("Unlock tx hash:%s\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("CooperativeSettle WaitForGenerateBlock err:%s", err)
		return
	}

	/*
		var receiptOrNone bool

		if receiptOrNone {
			channelSettled := self.ChannelIsSettled(self.nodeAddress, partner, channelId)
			if channelSettled == false {
				return
			}
		}
	*/

	return
}

func (self *TokenNetwork) settle(channelId common.ChannelID, transferredAmount common.TokenAmount,
	lockedAmount common.TokenAmount, locksRoot common.LocksRoot, partner common.Address,
	partnerTransferredAmount common.TokenAmount, partnerLockedAmount common.TokenAmount, partnerLocksroot common.LocksRoot) {

	var opLock *sync.Mutex

	if self.CheckForOutdatedChannel(self.nodeAddress, partner, channelId) == false {
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

	if ourBpIsLarger {
		txHash, err = self.ChannelClient.SettleChannel(uint64(channelId),
			comm.Address(partner),
			uint64(partnerTransferredAmount),
			uint64(partnerLockedAmount),
			partnerLocksroot[:],
			comm.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksRoot[:],
		)
	} else {

		txHash, err = self.ChannelClient.SettleChannel(uint64(channelId),
			comm.Address(self.nodeAddress),
			uint64(transferredAmount),
			uint64(lockedAmount),
			locksRoot[:],
			comm.Address(partner),
			uint64(partnerTransferredAmount),
			uint64(partnerLockedAmount),
			partnerLocksroot[:],
		)
	}

	if err != nil {
		log.Errorf("SettleChannel err:%s", err)
		return
	}
	log.Infof("SettleChannel tx hash:%s\n", getTxHashString(txHash))
	_, err = self.ChainClient.PollForTxConfirmed(time.Minute, txHash)
	if err != nil {
		log.Errorf("SettleChannel  WaitForGenerateBlock err:%s", err)
		self.checkChannelStateForSettle(self.nodeAddress, partner, channelId)
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
	channelId common.ChannelID) bool {
	onchainChannelDetails := self.detailChannel(participant1, participant2, 0)
	onchainChannelId := onchainChannelDetails.ChannelId
	if onchainChannelId != channelId {
		log.Debugf("onchainId %d channelId: %d", onchainChannelId, channelId)
		return false
	} else {
		return true
	}
}

func (self *TokenNetwork) getChannelState(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) int {

	channelDate := self.detailChannel(participant1, participant2, channelId)
	if channelDate != nil {
		return channelDate.State
	}

	// channelData is nil when call contract with proxy failed, actually we are not sure about the real channel state in the contract
	return micropayment.NonExistent
}

func (self *TokenNetwork) checkChannelStateForClose(participant1 common.Address,
	participant2 common.Address, channelId common.ChannelID) bool {

	channelState := self.getChannelState(participant1, participant2, channelId)
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
	participant2 common.Address, channelId common.ChannelID,
	depositAmount common.TokenAmount) (bool, error) {

	participantDetails := self.DetailParticipants(participant1, participant2,
		channelId)

	channelState := self.getChannelState(self.nodeAddress, participant2, channelId)
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
	participant2 common.Address, channelId common.ChannelID,
	withdrawAmount common.TokenAmount) bool {

	participantDetails := self.DetailParticipants(participant1, participant2,
		channelId)

	if participantDetails.OurDetails.Withdrawn > withdrawAmount {
		return false
	}

	channelState := self.getChannelState(participant1, participant2, channelId)
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
	participant2 common.Address, channelId common.ChannelID) bool {

	channelData := self.detailChannel(participant1, participant2, channelId)
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

func getTxHashString(txHash []byte) string {
	hash, err := comm.Uint256ParseFromBytes(txHash)
	if err != nil {
		log.Errorf("parse tx hash error")
		return "error parsing tx hash"
	}
	return hash.ToHexString()
}

func (t *TokenNetwork) GetFee(wa comm.Address, ta comm.Address) (uint64, error) {
	info, err := t.ChannelClient.GetFeeInfo(wa, ta)
	if err != nil {
		log.Errorf("%s\n", err.Error())
		return 0, err
	}
	return info.Flat, nil
}

func (t *TokenNetwork) SetFee(wa common.Address, ta common.Address, flat common.FeeAmount) ([]byte, error) {
	walletAddr :=comm.Address(wa)
	tokenAddr :=comm.Address(ta)
	msgHash := micropayment.FeeInfoMessageBundleHash(walletAddr, tokenAddr)
	sign, err := signature.Sign(t.ChannelClient.DefAcc.SigScheme, t.ChannelClient.DefAcc.PrivateKey, msgHash[:], nil)
	if err != nil {
		log.Errorf("%s\n", err.Error())
		return nil, err
	}
	serialize, err := signature.Serialize(sign)
	if err != nil {
		log.Errorf("%s\n", err.Error())
		return nil, err
	}
	pkHex := comm.PubKeyToHex(t.ChannelClient.DefAcc.PubKey())
	pkByte, err := comm.HexToBytes(pkHex)
	if err != nil {
		log.Errorf("%s\n", err.Error())
		return nil, err
	}
	feeInfo := micropayment.FeeInfo{
		WalletAddr: walletAddr,
		TokenAddr:  tokenAddr,
		Flat: 		uint64(flat),
		PublicKey:  pkByte,
		Signature:  serialize,
	}
	tx, err := t.ChannelClient.SetFeeInfo(feeInfo)
	if err != nil {
		log.Errorf("%s\n", err.Error())
		return nil, err
	}
	log.Info("Set fee tx:", hex.EncodeToString(tx))
	return tx, nil
}
