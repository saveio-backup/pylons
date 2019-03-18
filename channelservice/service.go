package channelservice

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bytes"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChain-go-sdk/ong"
	"github.com/oniio/oniChain/account"
	comm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	scUtils "github.com/oniio/oniChain/smartcontract/service/native/utils"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport"
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/storage"
	"github.com/oniio/oniChannel/transfer"
)

type ChannelService struct {
	chain           *network.BlockchainService
	queryStartBlock common.BlockHeight

	Account                       *account.Account
	channelEventHandler           *ChannelEventHandler
	messageHandler                *MessageHandler
	config                        map[string]string
	transport                     *transport.Transport
	targetsToIndentifierToStatues map[common.Address]*sync.Map
	ReceiveNotificationChannels   map[chan *transfer.EventPaymentReceivedSuccess]struct{}

	address            common.Address
	mircoAddress       common.Address
	dispatchEventsLock sync.Mutex
	eventPollLock      sync.Mutex
	databasePath       string
	databaseDir        string
	lockFile           string

	alarm         *AlarmTask
	Wal           *storage.WriteAheadLog
	snapshotGroup int

	tokennetworkidsToConnectionmanagers map[common.TokenNetworkID]*ConnectionManager
	lastFilterBlock                     common.BlockHeight
}

type PaymentStatus struct {
	paymentType            common.PaymentType
	paymentIdentifier      common.PaymentID
	amount                 common.TokenAmount
	tokenNetworkIdentifier common.TokenNetworkID
	paymentDone            chan bool //used to notify send success/fail
}

func (self *PaymentStatus) Match(paymentType common.PaymentType, tokenNetworkIdentifier common.TokenNetworkID, amount common.TokenAmount) bool {
	if self.paymentType == paymentType && self.tokenNetworkIdentifier == tokenNetworkIdentifier && self.amount == amount {
		return true
	} else {
		return false
	}
}

func NewChannelService(chain *network.BlockchainService,
	queryStartBlock common.BlockHeight,
	transport *transport.Transport,
	//defaultSecretRegistry SecretRegistry,
	defaultRegistryAddress common.Address,
	channel_event_handler *ChannelEventHandler,
	messageHandler *MessageHandler,
	config map[string]string) *ChannelService {
	if chain == nil {
		log.Error("error in create new channel service: chain service not available")
		return nil
	}
	self := new(ChannelService)

	self.chain = chain
	self.queryStartBlock = queryStartBlock
	self.mircoAddress = defaultRegistryAddress
	//self.defaultSecretRegistry = defaultSecretRegistry
	self.config = config
	self.Account = chain.GetAccount()
	self.targetsToIndentifierToStatues = make(map[common.Address]*sync.Map)
	self.tokennetworkidsToConnectionmanagers = make(map[common.TokenNetworkID]*ConnectionManager)
	self.ReceiveNotificationChannels = make(map[chan *transfer.EventPaymentReceivedSuccess]struct{})

	// address in the blockchain service is set when import wallet
	self.address = chain.Address
	self.channelEventHandler = new(ChannelEventHandler)
	self.messageHandler = messageHandler

	self.transport = transport
	self.alarm = NewAlarmTask(chain)

	if _, exist := config["DBPath"]; exist == false {
		self.setDefaultDBPath()
	} else {
		self.databasePath = config["DBPath"]
		if self.databasePath == "." {
			self.setDefaultDBPath()
		}
	}

	return self
}
func (self *ChannelService) setDefaultDBPath() {
	fullname, err := GetFullDatabasePath()
	if err == nil {
		databaseDir := filepath.Dir(self.databasePath)
		os.Mkdir(databaseDir, os.ModeDir)
		self.databasePath = fullname
		self.databaseDir = databaseDir
		log.Info("database set to", fullname)
	} else {
		self.databasePath = ":memory:"
		self.databaseDir = ""
		log.Warn("get full db path failed,use memory database")
	}
}
func (self *ChannelService) Start() error {
	// register to Endpoint contract
	var addr comm.Address
	var err error
	if addr, err = comm.AddressParseFromBytes(self.address[:]); err != nil {
		log.Fatal("address format invalid", err)
		return err
	}

	info, err := self.chain.ChannelClient.GetEndpointByAddress(addr)
	if err != nil {
		log.Fatal("check endpoint info failed:", err)
		return err
	}

	if info == nil {
		log.Info("this account haven`t registered, begin registering...")
		txHash, err := self.chain.ChannelClient.RegisterPaymentEndPoint([]byte(self.config["protocol"]), []byte(self.config["host"]), []byte(self.config["port"]), addr)
		if err != nil {
			log.Fatal("register endpoint service failed:", err)
			return err
		}
		log.Info("wait for the confirmation of transaction...")
		_, err = self.chain.ChainClient.PollForTxConfirmed(time.Duration(constants.POLL_FOR_COMFIRMED)*time.Second, txHash)
		if err != nil {
			log.Error("poll transaction failed:", err)
			return err
		}
		log.Info("endpoint register succesful")
	}
	log.Info("account been registered")
	sqliteStorage, err := storage.NewSQLiteStorage(self.databasePath)
	if err != nil {
		log.Error("create db failed:", err)
		return err
	}
	stateChangeQty := sqliteStorage.CountStateChanges()
	self.snapshotGroup = stateChangeQty / constants.SNAPSHOT_STATE_CHANGE_COUNT

	var lastLogBlockHeight common.BlockHeight
	self.Wal = storage.RestoreToStateChange(transfer.StateTransition, sqliteStorage, "latest")
	log.Info("channel service start: [RestoreToStateChange] finished")

	if self.Wal.StateManager.CurrentState == nil {
		var stateChange transfer.StateChange
		networkId, err := self.chain.ChainClient.GetNetworkId()
		if err != nil {
			log.Error("get network id failed:", err)
			return err
		}
		currentHeight, err := self.chain.ChainClient.GetCurrentBlockHeight()
		if err != nil {
			log.Error("get current block height failed:", err)
			return err
		}
		chainNetworkId := common.ChainID(networkId)
		stateChange = &transfer.ActionInitChain{
			BlockHeight: common.BlockHeight(currentHeight),
			OurAddress:  self.address,
			ChainId:     chainNetworkId,
		}
		self.HandleStateChange(stateChange)

		paymentNetwork := transfer.NewPaymentNetworkState()
		paymentNetwork.Address = common.PaymentNetworkID(scUtils.MicroPayContractAddress)
		stateChange = &transfer.ContractReceiveNewPaymentNetwork{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: common.TransactionHash{},
				BlockHeight:     lastLogBlockHeight,
			},
			PaymentNetwork: paymentNetwork,
		}
		self.HandleStateChange(stateChange)
		self.InitializeTokenNetwork()
	} else {
		lastLogBlockHeight = transfer.GetBlockHeight(self.StateFromChannel())
		log.Infof("Restored state from WAL,last log BlockHeight=%d", lastLogBlockHeight)
	}

	log.Info("db setup done")
	//set filter start block 	number
	self.lastFilterBlock = lastLogBlockHeight

	self.alarm.RegisterCallback(self.CallbackNewBlock)
	err = self.alarm.FirstRun()
	if err != nil {
		log.Error("run alarm call back failed:", err)
		return err
	}

	//reset neighbor networkStates
	channelState := self.StateFromChannel()
	channelState.NodeAddressesToNetworkStates = make(map[common.Address]string)

	// start the transport layer, pass channel service for message handling and signing
	err = self.transport.Start(self)
	if err != nil {
		log.Error("transport layer start failed:", err)
		return err
	}
	chainState := self.StateFromChannel()

	self.InitializeTransactionsQueues(chainState)

	self.alarm.Start()

	self.InitializeMessagesQueues(chainState)

	self.StartNeighboursHealthCheck()
	log.Info("channel service started")
	return nil
}

func (self *ChannelService) Stop() {
	self.alarm.Stop()
	self.transport.Stop()
	log.Info("channel service stopped")
}

func (self *ChannelService) AddPendingRoutine() {

}

func (self *ChannelService) HandleStateChange(stateChange transfer.StateChange) []transfer.Event {
	self.dispatchEventsLock.Lock()
	defer self.dispatchEventsLock.Unlock()

	log.Debug("[HandleStateChange]", reflect.TypeOf(stateChange).String())
	eventList := self.Wal.LogAndDispatch(stateChange)
	for _, e := range eventList {
		log.Debug("[HandleStateChange] Range Events: ", reflect.TypeOf(e).String())
		self.channelEventHandler.OnChannelEvent(self, e.(transfer.Event))
	}
	//take snapshot

	newSnapShotGroup := self.Wal.StateChangeId / constants.SNAPSHOT_STATE_CHANGE_COUNT
	if newSnapShotGroup > self.snapshotGroup {
		log.Info("storing snapshot, snapshot id = ", newSnapShotGroup)
		self.Wal.Snapshot()
		self.snapshotGroup = newSnapShotGroup
	}
	return eventList
}

func (self *ChannelService) SetNodeNetworkState(nodeAddress common.Address,
	networkState string) {

	return
}

func (self *ChannelService) StartNeighboursHealthCheck() {
	neighbours := transfer.GetNeighbours(self.StateFromChannel())

	for _, v := range neighbours {
		log.Debug("[StartNeighboursHealthCheck] Neighbour: %s", common.ToBase58(v))
		self.transport.StartHealthCheck(v)
	}

	return
}

func (self *ChannelService) InitializeTransactionsQueues(chainState *transfer.ChainState) {
	pendingTransactions := transfer.GetPendingTransactions(chainState)

	for _, transaction := range pendingTransactions {
		self.channelEventHandler.OnChannelEvent(self, transaction)
	}
}

func (self *ChannelService) RegisterPaymentStatus(target common.Address, identifier common.PaymentID, paymentType common.PaymentType,
	amount common.TokenAmount, tokenNetworkIdentifier common.TokenNetworkID) *PaymentStatus {
	status := &PaymentStatus{
		paymentType:            paymentType,
		paymentIdentifier:      identifier,
		amount:                 amount,
		tokenNetworkIdentifier: tokenNetworkIdentifier,
		paymentDone:            make(chan bool, 1),
	}

	if payments, exist := self.targetsToIndentifierToStatues[common.Address(target)]; exist {
		payments.Store(identifier, status)
	} else {
		payments := new(sync.Map)

		payments.Store(identifier, status)
		self.targetsToIndentifierToStatues[common.Address(target)] = payments
	}
	return status
}

func (self *ChannelService) GetPaymentStatus(target common.Address, identifier common.PaymentID) (status *PaymentStatus, exist bool) {
	payments, exist := self.targetsToIndentifierToStatues[common.Address(target)]
	if exist {
		paymentStatus, ok := payments.Load(identifier)
		if ok {
			return paymentStatus.(*PaymentStatus), true
		}
	}

	return nil, false
}

func (self *ChannelService) RemovePaymentStatus(target common.Address, identifier common.PaymentID) (ok bool) {
	payments, exist := self.targetsToIndentifierToStatues[common.Address(target)]
	if exist {
		_, ok := payments.Load(identifier)
		if ok {
			payments.Delete(identifier)
			return true
		}
	}

	return false
}

func (self *ChannelService) InitializeMessagesQueues(chainState *transfer.ChainState) {
	eventsQueues := transfer.GetAllMessageQueues(chainState)

	for queueIdentifier, eventQueue := range *eventsQueues {
		self.transport.StartHealthCheck(queueIdentifier.Recipient)

		for _, event := range eventQueue {

			switch event.(type) {
			case transfer.SendDirectTransfer:
				e := event.(transfer.SendDirectTransfer)
				self.RegisterPaymentStatus(common.Address(e.Recipient), e.PaymentIdentifier, common.PAYMENT_DIRECT, e.BalanceProof.TransferredAmount, e.BalanceProof.TokenNetworkIdentifier)
			}

			message := messages.MessageFromSendEvent(&event)
			if message != nil {
				err := self.Sign(message)
				if err != nil {
					log.Error("[InitializeMessagesQueues] Sign: ", err.Error())
				}
				err = self.transport.SendAsync(&queueIdentifier, message)
				if err != nil {
					log.Error("[InitializeMessagesQueues] SendAsync: ", err.Error())
				}
			}
		}
	}
}

func (self *ChannelService) CallbackNewBlock(latestBlock common.BlockHeight, blockHash common.BlockHash) {
	var events []map[string]interface{}

	fromBlock := self.lastFilterBlock + 1
	toBlock := latestBlock

	events, err := self.chain.ChannelClient.GetFilterArgsForAllEventsFromChannel(0, uint32(fromBlock), uint32(toBlock))
	if err != nil {
		return
	}

	numEvents := len(events)
	for i := 0; i < numEvents; i++ {
		OnBlockchainEvent(self, events[i])
	}

	block := new(transfer.Block)
	block.BlockHeight = toBlock
	block.BlockHash = blockHash

	self.HandleStateChange(block)

	self.lastFilterBlock = toBlock
	return
}

func (self *ChannelService) OnMessage(message proto.Message, from string) {
	self.messageHandler.OnMessage(self, message)
	return
}

func (self *ChannelService) Sign(message interface{}) error {
	_, ok := message.(messages.SignedMessageInterface)
	if !ok {
		return errors.New("invalid message to sign")
	}

	err := messages.Sign(self.Account, message.(messages.SignedMessageInterface))
	if err != nil {
		return err
	}

	return nil
}

func (self *ChannelService) ConnectionManagerForTokenNetwork(tokenNetworkIdentifier common.TokenNetworkID) *ConnectionManager {
	var manager *ConnectionManager
	var exist bool
	manager, exist = self.tokennetworkidsToConnectionmanagers[tokenNetworkIdentifier]
	if exist == false {
		manager = NewConnectionManager(self, tokenNetworkIdentifier)
		self.tokennetworkidsToConnectionmanagers[tokenNetworkIdentifier] = manager
	}
	return manager
}

func (self *ChannelService) LeaveAllTokenNetwork() {

	stateChange := &transfer.ActionLeaveAllNetworks{}
	self.HandleStateChange(stateChange)

	return
}

func (self *ChannelService) CloseAndSettle() {

	self.LeaveAllTokenNetwork()

	if len(self.tokennetworkidsToConnectionmanagers) > 0 {
		WaitForSettleAllChannels(self, self.alarm.Getinterval())
	}

	return
}

func CreateDefaultIdentifier() common.PaymentID {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	num := r.Uint64()

	return common.PaymentID(num)
}

func (self *ChannelService) StateFromChannel() *transfer.ChainState {
	var result *transfer.ChainState

	state := self.Wal.StateManager.CurrentState
	if state != nil {
		result, _ = state.(*transfer.ChainState)
	}

	//[TODO] call self.Wal.DeepCopy when optimize for writer. This behavior can
	//be controled by new config option or make default after stable.
	return result
}

func (self *ChannelService) InitializeTokenNetwork() {

	tokenNetworkState := transfer.NewTokenNetworkState(self.address)
	tokenNetworkState.Address = common.TokenNetworkID(ong.ONG_CONTRACT_ADDRESS)
	tokenNetworkState.TokenAddress = common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	newTokenNetwork := &transfer.ContractReceiveNewTokenNetwork{PaymentNetworkIdentifier: common.PaymentNetworkID(scUtils.MicroPayContractAddress),
		TokenNetwork: tokenNetworkState}

	self.HandleStateChange(newTokenNetwork)
}

func (self *ChannelService) GetPaymentChannelArgs(tokenNetworkId common.TokenNetworkID, channelId common.ChannelID) map[string]interface{} {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState, tokenNetworkId, channelId)
	if channelState == nil {
		return nil
	}

	return self.GetPaymentArgs(channelState)
}

func (self *ChannelService) GetPaymentArgs(channelState *transfer.NettingChannelState) map[string]interface{} {
	if channelState == nil {
		return nil
	}

	ourState := channelState.GetChannelEndState(0)
	partnerState := channelState.GetChannelEndState(1)

	if ourState == nil || partnerState == nil {
		return nil
	}

	args := map[string]interface{}{
		"participant1":  ourState.GetAddress(),
		"participant2":  partnerState.GetAddress(),
		"blockHeight":   channelState.OpenTransaction.FinishedBlockHeight,
		"settleTimeout": channelState.SettleTimeout,
	}

	return args
}

func EventFilterForPayments(event transfer.Event, tokenNetworkIdentifier common.TokenNetworkID,
	partnerAddress common.Address) bool {

	var result bool

	result = false
	emptyAddress := common.Address{}
	switch event.(type) {
	case *transfer.EventPaymentSentSuccess:
		eventPaymentSentSuccess := event.(*transfer.EventPaymentSentSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentSuccess.Target == common.Address(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentReceivedSuccess:
		eventPaymentReceivedSuccess := event.(*transfer.EventPaymentReceivedSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentReceivedSuccess.Initiator == common.InitiatorAddress(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentSentFailed:
		eventPaymentSentFailed := event.(*transfer.EventPaymentSentFailed)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentFailed.Target == common.Address(partnerAddress) {
			result = true
		}
	}

	return result
}

func (self *ChannelService) Address() common.Address {
	return self.address
}

func (self *ChannelService) GetChannel(registryAddress common.PaymentNetworkID, tokenAddress *common.TokenAddress,
	partnerAddress *common.Address) *transfer.NettingChannelState {

	var result *transfer.NettingChannelState
	channelList := self.GetChannelList(registryAddress, tokenAddress, partnerAddress)
	if channelList.Len() != 0 {
		result = channelList.Back().Value.(*transfer.NettingChannelState)
	} else {
		log.Info("[GetChannelList] Len = 0")
	}

	return result
}

func (self *ChannelService) tokenNetworkConnect(
	registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress,
	funds common.TokenAmount,
	initialChannelTarget int,
	joinableFundsTarget float32) {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	connectionManager.connect(funds, initialChannelTarget, joinableFundsTarget)

	return
}

func (self *ChannelService) tokenNetworkLeave(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)
	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	return connectionManager.Leave(registryAddress)
}

func (self *ChannelService) OpenChannel(tokenAddress common.TokenAddress,
	partnerAddress common.Address) common.ChannelID {

	id, err := self.chain.ChannelClient.GetChannelIdentifier(comm.Address(self.address), comm.Address(partnerAddress))
	if err != nil {
		log.Error("get channel identifier failed ", err)
		return 0
	}
	regAddr, _ := comm.AddressParseFromBytes(self.address[:])
	patAddr, _ := comm.AddressParseFromBytes(partnerAddress[:])
	if id != 0 {
		log.Infof("channel between %s and %s already setup", regAddr.ToBase58(), patAddr.ToBase58())
		return common.ChannelID(id)
	}
	if id == 0 {
		log.Infof("channel between %s and %s haven`t setup. start to create new one", regAddr.ToBase58(), patAddr.ToBase58())
	}

	tokenNetwork := self.chain.NewTokenNetwork(common.Address(ong.ONG_CONTRACT_ADDRESS))
	channelId := tokenNetwork.NewNettingChannel(partnerAddress, constants.SETTLE_TIMEOUT)
	if channelId == 0 {
		log.Error("open channel failed")
		return 0
	}
	log.Info("wait for new channel ... ")
	channelState := WaitForNewChannel(self, common.PaymentNetworkID(self.mircoAddress), common.TokenAddress(ong.ONG_CONTRACT_ADDRESS), partnerAddress,
		float32(constants.OPEN_CHANNEL_RETRY_TIMEOUT), constants.OPEN_CHANNEL_RETRY_TIMES)
	if channelState == nil {
		log.Error("setup channel timeout")
		return 0
	}
	log.Infof("new channel between %s and %s has setup, channel ID = %d", regAddr.ToBase58(), patAddr.ToBase58(), channelState.Identifier)

	chainState := self.StateFromChannel()

	tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, channelState.TokenNetworkIdentifier)
	tokenNetworkState.AddRoute(self.Address(), partnerAddress, channelState.Identifier)
	return channelState.Identifier

}

func (self *ChannelService) SetTotalChannelDeposit(tokenAddress common.TokenAddress, partnerAddress common.Address, totalDeposit common.TokenAmount) error {

	chainState := self.StateFromChannel()
	partAddr, _ := comm.AddressParseFromBytes(partnerAddress[:])
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.mircoAddress), tokenAddress, partnerAddress)
	if channelState == nil {
		log.Errorf("deposit failed, can not find specific channel with %s", partAddr.ToBase58())
		return errors.New("can not find specific channel")
	}

	args := self.GetPaymentArgs(channelState)
	if args == nil {
		log.Error("can not get payment channel args")
		return errors.New("can not get payment channel args")
	}

	channelProxy := self.chain.PaymentChannel(common.Address(tokenAddress), channelState.Identifier, args)

	balance, err := channelProxy.GetGasBalance()
	if err != nil {
		return err
	}

	addedNum := totalDeposit - channelState.OurState.ContractBalance
	if balance < addedNum {
		return errors.New("gas balance not enough")
	}

	err = channelProxy.SetTotalDeposit(totalDeposit)
	if err != nil {
		return err
	}
	targetAddress := self.address
	log.Info("wait for balance updated...")
	WaitForParticipantNewBalance(self, common.PaymentNetworkID(self.mircoAddress), tokenAddress, partnerAddress,
		targetAddress, totalDeposit, constants.DEPOSIT_RETRY_TIMEOUT)

	return nil
}

func (self *ChannelService) ChannelClose(tokenAddress common.TokenAddress, partnerAddress common.Address,
	retryTimeout common.NetworkTimeout) {

	addressList := list.New()
	addressList.PushBack(partnerAddress)

	self.ChannelBatchClose(tokenAddress, addressList, retryTimeout)
	return
}

func (self *ChannelService) ChannelBatchClose(tokenAddress common.TokenAddress, partnerAddress *list.List, retryTimeout common.NetworkTimeout) {

	chainState := self.StateFromChannel()
	registryAddress := common.PaymentNetworkID(self.mircoAddress)
	channelsToClose := transfer.FilterChannelsByPartnerAddress(chainState, registryAddress,
		tokenAddress, partnerAddress)

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		chainState, registryAddress, tokenAddress)

	//[TODO] use BlockchainService.PaymentChannel to get PaymentChannels and
	// get the lock!!

	channelIds := list.New()
	for e := channelsToClose.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*transfer.NettingChannelState)

		identifier := channelState.GetIdentifier()
		//channel := self.chain.PaymentChannel(common.Address{}, identifier, nil)
		/*
			lock := channel.LockOrRaise()
			lock.Lock()
			defer lock.Unlock()
		*/

		channelIds.PushBack(&identifier)

		channelClose := new(transfer.ActionChannelClose)
		channelClose.TokenNetworkIdentifier = tokenNetworkIdentifier
		channelClose.ChannelIdentifier = identifier

		self.HandleStateChange(channelClose)

		tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, channelState.TokenNetworkIdentifier)
		tokenNetworkState.DelRoute(channelState.Identifier)
	}

	WaitForClose(self, registryAddress, tokenAddress, channelIds, float32(retryTimeout))

	return
}

func (self *ChannelService) GetChannelList(registryAddress common.PaymentNetworkID,
	tokenAddress *common.TokenAddress, partnerAddress *common.Address) *list.List {

	result := list.New()
	chainState := self.StateFromChannel()

	if tokenAddress != nil && partnerAddress != nil {
		channelState := transfer.GetChannelStateFor(chainState, registryAddress,
			*tokenAddress, *partnerAddress)

		if channelState != nil {
			result.PushBack(channelState)
		} else {
			log.Info("[GetChannelList] channelState == nil")
		}
	} else if tokenAddress != nil {
		result = transfer.ListChannelStateForTokenNetwork(chainState, registryAddress,
			*tokenAddress)
	} else {
		result = transfer.ListAllChannelState(chainState)
	}

	return result
}

func (self *ChannelService) GetNodeNetworkState(nodeAddress common.Address) string {
	return transfer.GetNodeNetworkStatus(self.StateFromChannel(), nodeAddress)
}

func (self *ChannelService) GetTokensList(registryAddress common.PaymentNetworkID) *list.List {
	tokensList := transfer.GetTokenNetworkAddressesFor(self.StateFromChannel(),
		registryAddress)

	return tokensList
}

func (self *ChannelService) TransferAndWait(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress, amount common.TokenAmount, target common.Address,
	identifier common.PaymentID, transferTimeout int) {

	asyncResult := self.TransferAsync(registryAddress, tokenAddress, amount,
		target, identifier)

	select {
	case <-*asyncResult:
		break
	case <-time.After(time.Duration(transferTimeout) * time.Second):
		break
	}

	return
}

func (self *ChannelService) TransferAsync(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress, amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) *chan int {

	asyncResult := new(chan int)

	//[TODO] Adding Graph class to hold route information, support async transfer
	// by calling channel mediated_transfer_async
	return asyncResult
}

func (self *ChannelService) CanTransfer(target common.Address, amount common.TokenAmount) bool {
	chainState := self.StateFromChannel()
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	paymentNetworkIdentifier := common.PaymentNetworkID(self.mircoAddress)
	channelState := transfer.GetChannelStateFor(chainState, paymentNetworkIdentifier, tokenAddress, target)
	if channelState.OurState.ContractBalance < amount {
		return false
	} else {
		return true
	}
}

func (self *ChannelService) DirectTransferAsync(amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) (chan bool, error) {
	if target == common.ADDRESS_EMPTY {
		log.Error("target address is invalid:", target)
		return nil, fmt.Errorf("target address is invalid")
	}
	chainState := self.StateFromChannel()
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	paymentNetworkIdentifier := common.PaymentNetworkID(self.mircoAddress)
	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(chainState,
		paymentNetworkIdentifier, tokenAddress)

	if !self.CanTransfer(target, amount) {
		return nil, fmt.Errorf("contract balance small than transfer amount")
	}

	self.transport.StartHealthCheck(common.Address(target))
	if identifier == common.PaymentID(0) {
		identifier = CreateDefaultIdentifier()
	}

	paymentStatus, exist := self.GetPaymentStatus(common.Address(target), identifier)
	if exist {
		if !paymentStatus.Match(common.PAYMENT_DIRECT, tokenNetworkIdentifier, amount) {
			return nil, errors.New("Another payment with same id is in flight. ")
		}
		log.Warn("payment already existed:")
		return paymentStatus.paymentDone, nil
	}

	directTransfer := &transfer.ActionTransferDirect{
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		ReceiverAddress:        common.Address(target),
		PaymentIdentifier:      identifier,
		Amount:                 amount,
	}

	paymentStatus = self.RegisterPaymentStatus(target, identifier, common.PAYMENT_DIRECT, amount, tokenNetworkIdentifier)
	//paymentStatus, _ = self.GetPaymentStatus(common.Address(target), identifier)

	self.HandleStateChange(directTransfer)
	return paymentStatus.paymentDone, nil
}

func (self *ChannelService) MediaTransfer(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress, amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) (chan bool, error) {

	if target == common.ADDRESS_EMPTY {
		log.Error("target address is invalid:", target)
		return nil, fmt.Errorf("target address is invalid")
	}
	if amount <= 0 {
		log.Error("amount negative:", amount)
		return nil, fmt.Errorf("amount negative ")
	}

	chainState := self.StateFromChannel()

	//TODO: check validTokens
	paymentNetworkIdentifier := common.PaymentNetworkID(self.mircoAddress)
	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		chainState, paymentNetworkIdentifier, tokenAddress)
	secret := common.SecretRandom(constants.SECRET_LEN)
	log.Debug("[MediaTransfer] Secret: ", secret)
	//TODO: check secret used

	//asyncResult, err := self.StartMediatedTransferWithSecret(tokenNetworkIdentifier,
	//	amount, target, identifier, secret)

	self.transport.StartHealthCheck(target)
	if identifier == common.PaymentID(0) {
		identifier = CreateDefaultIdentifier()
	}

	//with self.payment_identifier_lock:
	paymentStatus, exist := self.GetPaymentStatus(target, identifier)
	if exist {
		paymentStatusMatches := paymentStatus.Match(common.PAYMENT_MEDIATED, tokenNetworkIdentifier, amount)
		if !paymentStatusMatches {
			return nil, fmt.Errorf("Another payment with the same id is in flight. ")
		}
		return paymentStatus.paymentDone, nil
	}

	asyncDone := make(chan bool)
	paymentStatus = &PaymentStatus{
		paymentType:            common.PAYMENT_MEDIATED,
		paymentIdentifier:      identifier,
		amount:                 amount,
		tokenNetworkIdentifier: tokenNetworkIdentifier,
		paymentDone:            asyncDone,
	}

	payments := new(sync.Map)
	payments.Store(identifier, paymentStatus)
	self.targetsToIndentifierToStatues[target] = payments

	actionInitInitiator, err := self.InitiatorInit(identifier, amount,
		secret, tokenNetworkIdentifier, target)
	if err != nil {
		log.Error("[MediaTransfer]:", err.Error())
		return nil, err
	}
	if actionInitInitiator.TransferDescription == nil {
		log.Warn("MediaTransfer transferDescription == nil ")
	}
	//# Dispatch the state change even if there are no routes to create the
	//# wal entry.
	self.HandleStateChange(actionInitInitiator)
	return paymentStatus.paymentDone, nil
}

func (self *ChannelService) InitiatorInit(transferIdentifier common.PaymentID,
	transferAmount common.TokenAmount, transferSecret common.Secret,
	tokenNetworkIdentifier common.TokenNetworkID,
	targetAddress common.Address) (*transfer.ActionInitInitiator, error) {
	if 0 == bytes.Compare(transferSecret, common.EmptySecretHash[:]) {
		return nil, fmt.Errorf("Should never end up initiating transfer with Secret 0x0 ")
	}

	secretHash := common.GetHash(transferSecret)
	log.Debug("[InitiatorInit] secretHash: ", secretHash)
	transferState := &transfer.TransferDescriptionWithSecretState{
		PaymentNetworkIdentifier: common.PaymentNetworkID{},
		PaymentIdentifier:        transferIdentifier,
		Amount:                   transferAmount,
		TokenNetworkIdentifier:   tokenNetworkIdentifier,
		Initiator:                self.address,
		Target:                   targetAddress,
		Secret:                   transferSecret,
		SecretHash:               secretHash,
	}

	var previousAddress common.Address
	chainState := self.StateFromChannel()
	routes, err := GetBestRoutes(chainState, tokenNetworkIdentifier, self.address,
		targetAddress, transferAmount, previousAddress)
	if err != nil {
		return nil, err
	}
	initInitiatorStateChange := &transfer.ActionInitInitiator{
		TransferDescription: transferState,
		Routes:              routes,
	}
	return initInitiatorStateChange, nil
}

func (self *ChannelService) MediatorInit(lockedTransfer *messages.LockedTransfer) *transfer.ActionInitMediator {
	var initiatorAddr common.Address
	copy(initiatorAddr[:], lockedTransfer.Initiator.Address)

	fromTransfer := LockedTransferSignedFromMessage(lockedTransfer)

	chainState := self.StateFromChannel()
	routes, _ := GetBestRoutes(chainState, fromTransfer.BalanceProof.TokenNetworkIdentifier,
		self.address, common.Address(fromTransfer.Target), fromTransfer.Lock.Amount, initiatorAddr)
	fromRoute := &transfer.RouteState{
		NodeAddress:       initiatorAddr,
		ChannelIdentifier: fromTransfer.BalanceProof.ChannelIdentifier,
	}
	initMediatorStateChange := &transfer.ActionInitMediator{
		Routes:       routes,
		FromRoute:    fromRoute,
		FromTransfer: fromTransfer,
	}
	return initMediatorStateChange
}

func (self *ChannelService) TargetInit(lockedTransfer *messages.LockedTransfer) *transfer.ActionInitTarget {
	var sender common.Address
	copy(sender[:], lockedTransfer.BaseMessage.EnvelopeMessage.Signature.Sender.Address)

	fromTransfer := LockedTransferSignedFromMessage(lockedTransfer)
	fromRoute := &transfer.RouteState{
		NodeAddress:       sender,
		ChannelIdentifier: fromTransfer.BalanceProof.ChannelIdentifier,
	}
	initTargetStateChange := &transfer.ActionInitTarget{
		Route:    fromRoute,
		Transfer: fromTransfer,
	}
	return initTargetStateChange
}

func (self *ChannelService) GetEventsPaymentHistoryWithTimestamps(tokenAddress common.TokenAddress,
	targetAddress common.Address, limit int, offset int) *list.List {

	result := list.New()

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(self.StateFromChannel(),
		common.PaymentNetworkID{}, tokenAddress)
	events := self.Wal.Storage.GetEventsWithTimestamps(limit, offset)
	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		if EventFilterForPayments(event.WrappedEvent, tokenNetworkIdentifier, targetAddress) == true {
			result.PushBack(event)
		}

	}

	return result

}

func (self *ChannelService) GetEventsPaymentHistory(tokenAddress common.TokenAddress,
	targetAddress common.Address, limit int, offset int) *list.List {
	result := list.New()

	events := self.GetEventsPaymentHistoryWithTimestamps(tokenAddress, targetAddress,
		limit, offset)

	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		result.PushBack(event.WrappedEvent)

	}

	return result

}

func (self *ChannelService) GetInternalEventsWithTimestamps(limit int, offset int) *list.List {
	return self.Wal.Storage.GetEventsWithTimestamps(limit, offset)

}

func (self *ChannelService) SetHostAddr(nodeAddress common.Address, hostAddr string) {
	self.transport.SetHostAddr(nodeAddress, hostAddr)
}

func (self *ChannelService) GetHostAddr(nodeAddress common.Address) (string, error) {
	return self.transport.GetHostAddr(nodeAddress)
}

//func (self *ChannelService) Get(nodeAddress common.Address) string {
//	info, err := self.chain.ChannelClient.GetEndpointByAddress(comm.Address(nodeAddress))
//	regAddr, _ := comm.AddressParseFromBytes(nodeAddress[:])
//	if err != nil {
//		log.Warnf("get %s reg info err: %s", regAddr.ToBase58(), err.Error())
//		return ""
//	}
//	if info == nil {
//		log.Warnf("node %s haven`t been registed", regAddr.ToBase58())
//		return ""
//	}
//	nodeAddr := string(info.Protocol) + "://" + string(info.IP) + ":" + string(info.Port)
//	log.Infof("peer %s registe address: %s", regAddr.ToBase58(), nodeAddr)
//	return nodeAddr
//}

func GetFullDatabasePath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0:i+1]) + "channel.db", nil
}
