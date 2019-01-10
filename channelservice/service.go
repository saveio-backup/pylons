package channelservice

import (
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport"
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/storage"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

const SnapshotStateChangesCount int = 500

type ChannelService struct {
	chain           *network.BlockchainService
	queryStartBlock typing.BlockHeight

	Account                       *account.Account
	channelEventHandler           *ChannelEventHandler
	messageHandler                *MessageHandler
	config                        map[string]string
	discovery                     *network.ContractDiscovery
	transport                     *transport.Transport
	targetsToIndentifierToStatues map[typing.Address]*sync.Map
	ReceiveNotificationChannels   map[chan *transfer.EventPaymentReceivedSuccess]struct{}

	address typing.Address

	dispatchEventsLock sync.Mutex
	eventPollLock      sync.Mutex
	databasePath       string
	databaseDir        string
	lockFile           string

	alarm         *AlarmTask
	Wal           *storage.WriteAheadLog
	snapshotGroup int

	tokennetworkidsToConnectionmanagers map[typing.TokenNetworkID]*ConnectionManager
	lastFilterBlock                     typing.BlockHeight
}

type PaymentType int

const (
	PAYMENT_DIRECT PaymentType = iota
	PAYMENT_MEDIATED
)

type PaymentStatus struct {
	paymentType            PaymentType
	paymentIdentifier      typing.PaymentID
	amount                 typing.TokenAmount
	tokenNetworkIdentifier typing.TokenNetworkID
	paymentDone            chan bool //used to notify send success/fail
}

func (self *PaymentStatus) Match(paymentType PaymentType, tokenNetworkIdentifier typing.TokenNetworkID, amount typing.TokenAmount) bool {
	if self.paymentType == paymentType && self.tokenNetworkIdentifier == tokenNetworkIdentifier && self.amount == amount {
		return true
	} else {
		return false
	}
}

func NewChannelService(chain *network.BlockchainService,
	queryStartBlock typing.BlockHeight,
	transport *transport.Transport,
	//defaultSecretRegistry SecretRegistry,
	channel_event_handler *ChannelEventHandler,
	messageHandler *MessageHandler,
	config map[string]string,
	discovery *network.ContractDiscovery) *ChannelService {
	if chain == nil {
		fmt.Println("error in create new channel service: chain service not available")
		return nil
	}
	self := new(ChannelService)

	self.chain = chain
	self.queryStartBlock = queryStartBlock
	//self.defaultSecretRegistry = defaultSecretRegistry
	self.config = config
	self.Account = chain.GetAccount()
	self.targetsToIndentifierToStatues = make(map[typing.Address]*sync.Map)
	self.tokennetworkidsToConnectionmanagers = make(map[typing.TokenNetworkID]*ConnectionManager)
	self.ReceiveNotificationChannels = make(map[chan *transfer.EventPaymentReceivedSuccess]struct{})

	// address in the blockchain service is set when import wallet
	self.address = chain.Address
	self.discovery = discovery

	self.channelEventHandler = new(ChannelEventHandler)
	self.messageHandler = messageHandler

	if _, exist := config["database_path"]; exist == false {
		self.databasePath = ":memory:"
	} else {
		self.databasePath = config["database_path"]
		if self.databasePath == "." {
			fullname, err := GetFullDatabasePath()
			if err == nil {
				self.databasePath = fullname
			} else {
				self.databasePath = ":memory:"
			}
		}
	}

	self.transport = transport

	self.alarm = NewAlarmTask(chain)

	if self.databasePath != ":memory:" {
		databaseDir := filepath.Dir(config["database_path"])
		os.Mkdir(databaseDir, os.ModeDir)
		self.databaseDir = databaseDir
	} else {
		self.databasePath = ":memory:"
		self.databaseDir = ""
	}

	return self
}

func (self *ChannelService) Start() {
	// register to Endpoint contract
	port, _ := strconv.Atoi(self.config["port"])
	go self.discovery.Register(self.address, self.config["host"], port)

	var lastLogBlockHeight typing.BlockHeight

	sqliteStorage := storage.NewSQLiteStorage(self.databasePath)
	self.Wal = storage.RestoreToStateChange(transfer.StateTransition, sqliteStorage, "latest")
	if self.Wal.StateManager.CurrentState == nil {

		var stateChange transfer.StateChange

		rand := rand.New(rand.NewSource(time.Now().UnixNano()))
		stateChange = &transfer.ActionInitChain{
			PseudoRandomGenerator: rand,
			BlockHeight:           lastLogBlockHeight,
			OurAddress:            typing.Address{},
			ChainId:               0}
		self.HandleStateChange(stateChange)

		paymentNetwork := transfer.NewPaymentNetworkState()
		stateChange = &transfer.ContractReceiveNewPaymentNetwork{
			transfer.ContractReceiveStateChange{typing.TransactionHash{}, lastLogBlockHeight}, paymentNetwork}
		self.HandleStateChange(stateChange)

		self.InitializeTokenNetwork()

		lastLogBlockHeight = 0
	} else {

		lastLogBlockHeight = transfer.GetBlockHeight(self.StateFromChannel())

	}

	stateChangeQty := self.Wal.Storage.CountStateChanges()
	self.snapshotGroup = stateChangeQty / SnapshotStateChangesCount

	//set filter start block number
	self.lastFilterBlock = lastLogBlockHeight

	self.alarm.RegisterCallback(self.CallbackNewBlock)
	self.alarm.FirstRun()

	// start the transport layer, pass channel service for message hanlding and signing
	self.transport.Start(self)

	chainState := self.StateFromChannel()

	self.InitializeTransactionsQueues(chainState)

	self.alarm.Start()

	self.InitializeMessagesQueues(chainState)

	self.StartNeighboursHealthcheck()

	return
}

func (self *ChannelService) run() {

}

func (self *ChannelService) Stop() {
	self.alarm.Stop()
	self.transport.Stop()
}

func (self *ChannelService) AddPendingRoutine() {

}

func (self *ChannelService) GetBlockHeight() int {
	return 0
}

func (self *ChannelService) HandleStateChange(stateChange transfer.StateChange) *list.List {

	eventList := self.Wal.LogAndDispatch(stateChange)
	for e := eventList.Front(); e != nil; e = e.Next() {
		temp := e.Value

		self.channelEventHandler.OnChannelEvent(self, temp.(transfer.Event))
	}

	return eventList
}

func (self *ChannelService) SetNodeNetworkState(nodeAddress typing.Address,
	networkState string) {

	return
}

func (self *ChannelService) StartNeighboursHealthcheck() {
	neighbours := transfer.GetNeigbours(self.StateFromChannel())

	for _, v := range neighbours {
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

func (self *ChannelService) RegisterPaymentStatus(target typing.Address, identifier typing.PaymentID, paymentType PaymentType, amount typing.TokenAmount, tokenNetworkIdentifier typing.TokenNetworkID) {
	status := &PaymentStatus{
		paymentType:            paymentType,
		paymentIdentifier:      identifier,
		amount:                 amount,
		tokenNetworkIdentifier: tokenNetworkIdentifier,
		paymentDone:            make(chan bool, 1),
	}

	if payments, exist := self.targetsToIndentifierToStatues[typing.Address(target)]; exist {
		payments.Store(identifier, status)
	} else {
		payments := new(sync.Map)

		payments.Store(identifier, status)
		self.targetsToIndentifierToStatues[typing.Address(target)] = payments
	}
}

func (self *ChannelService) GetPaymentStatus(target typing.Address, identifier typing.PaymentID) (status *PaymentStatus, exist bool) {
	payments, exist := self.targetsToIndentifierToStatues[typing.Address(target)]
	if exist {
		paymentStatus, ok := payments.Load(identifier)
		if ok {
			return paymentStatus.(*PaymentStatus), true
		}
	}

	return nil, false
}

func (self *ChannelService) RemovePaymentStatus(target typing.Address, identifier typing.PaymentID) (ok bool) {
	payments, exist := self.targetsToIndentifierToStatues[typing.Address(target)]
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

				self.RegisterPaymentStatus(typing.Address(e.Recipient), e.PaymentIdentifier, PAYMENT_DIRECT, e.BalanceProof.TransferredAmount, e.BalanceProof.TokenNetworkIdentifier)

			}

			message := messages.MessageFromSendEvent(&event)

			self.Sign(message)
			self.transport.SendAsync(&queueIdentifier, message)
		}
	}
}

func (self *ChannelService) CallbackNewBlock(latestBlock typing.BlockHeight, blockHash typing.BlockHash) {
	var events []map[string]interface{}

	fromBlock := self.lastFilterBlock + 1
	toBlock := latestBlock

	events, err := self.chain.Client.Native.Channel.GetFilterArgsForAllEventsFromChannel(0, uint32(fromBlock), uint32(toBlock))
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
		return fmt.Errorf("message need no signature")
	}

	err := messages.Sign(self.Account, message.(messages.SignedMessageInterface))
	if err != nil {
		return err
	}

	return nil
}

func (self *ChannelService) InstallAllBlockchainFilters() {
	return
}

func (self *ChannelService) ConnectionManagerForTokenNetwork(tokenNetworkIdentifier typing.TokenNetworkID) *ConnectionManager {
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
		WaitForSettleAllChannels(self, self.alarm.GetSleepTime())
	}

	return
}

func CreateDefaultIdentifier() typing.PaymentID {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	num := r.Uint64()

	return typing.PaymentID(num)
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
	//Simulate EVENT_TOKEN_NETWORK_CREATED block chain event to
	// initialize the only one TokenNetworkState!!

	tokenNetworkState := transfer.NewTokenNetworkState()
	newTokenNetwork := &transfer.ContractReceiveNewTokenNetwork{TokenNetwork: tokenNetworkState}

	self.HandleStateChange(newTokenNetwork)
}

func (self *ChannelService) GetPaymentChannelArgs(tokenNetworkId typing.TokenNetworkID, channelId typing.ChannelID) map[string]interface{} {
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
func EventFilterForPayments(
	event transfer.Event,
	tokenNetworkIdentifier typing.TokenNetworkID,
	partnerAddress typing.Address) bool {

	var result bool

	result = false
	emptyAddress := typing.Address{}
	switch event.(type) {
	case *transfer.EventPaymentSentSuccess:
		eventPaymentSentSuccess := event.(*transfer.EventPaymentSentSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentSuccess.Target == typing.Address(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentReceivedSuccess:
		eventPaymentReceivedSuccess := event.(*transfer.EventPaymentReceivedSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentReceivedSuccess.Initiator == typing.InitiatorAddress(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentSentFailed:
		eventPaymentSentFailed := event.(*transfer.EventPaymentSentFailed)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentFailed.Target == typing.Address(partnerAddress) {
			result = true
		}
	}

	return result
}

func (self *ChannelService) Address() typing.Address {
	return self.address
}

func (self *ChannelService) GetChannel(
	registryAddress typing.PaymentNetworkID,
	tokenAddress *typing.TokenAddress,
	partnerAddress *typing.Address) *transfer.NettingChannelState {

	var result *transfer.NettingChannelState

	channelList := self.GetChannelList(registryAddress, tokenAddress, partnerAddress)
	if channelList.Len() != 0 {
		result = channelList.Back().Value.(*transfer.NettingChannelState)
	}

	return result
}

func (self *ChannelService) tokenNetworkConnect(
	registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress,
	funds typing.TokenAmount,
	initialChannelTarget int,
	joinableFundsTarget float32) {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	connectionManager.connect(funds, initialChannelTarget, joinableFundsTarget)

	return
}

func (self *ChannelService) tokenNetworkLeave(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	return connectionManager.Leave(registryAddress)
}

func (self *ChannelService) ChannelOpen(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address,
	settleTimeout typing.BlockTimeout, retryTimeout typing.NetworkTimeout) typing.ChannelID {

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, registryAddress,
		tokenAddress, partnerAddress)

	if channelState != nil {
		return channelState.Identifier
	}

	tokenNetwork := self.chain.TokenNetwork(typing.Address{})
	tokenNetwork.NewNettingChannel(partnerAddress, int(settleTimeout))

	WaitForNewChannel(self, registryAddress, tokenAddress, partnerAddress,
		float32(retryTimeout))

	channelState = transfer.GetChannelStateFor(self.StateFromChannel(), registryAddress, tokenAddress, partnerAddress)

	return channelState.Identifier

}

func (self *ChannelService) SetTotalChannelDeposit(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address, totalDeposit typing.TokenAmount,
	retryTimeout typing.NetworkTimeout) {

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, registryAddress, tokenAddress, partnerAddress)
	if channelState == nil {
		return
	}

	args := self.GetPaymentArgs(channelState)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}

	channelProxy := self.chain.PaymentChannel(typing.Address{}, channelState.Identifier, args)

	balance, err := channelProxy.GetGasBalance()
	if err != nil {
		return
	}

	addednum := totalDeposit - channelState.OurState.ContractBalance
	if balance < addednum {
		return
	}

	channelProxy.SetTotalDeposit(totalDeposit)
	targetAddress := self.address
	WaitForParticipantNewBalance(self, registryAddress, tokenAddress, partnerAddress,
		targetAddress, totalDeposit, float32(retryTimeout))

	return
}

func (self *ChannelService) ChannelClose(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address,
	retryTimeout typing.NetworkTimeout) {

	addressList := list.New()
	addressList.PushBack(partnerAddress)

	self.ChannelBatchClose(registryAddress, tokenAddress, addressList, retryTimeout)
	return
}

func (self *ChannelService) ChannelBatchClose(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress *list.List, retryTimeout typing.NetworkTimeout) {

	chainState := self.StateFromChannel()
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
		//channel := self.chain.PaymentChannel(typing.Address{}, identifier, nil)
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
	}

	WaitForClose(self, registryAddress, tokenAddress, channelIds, float32(retryTimeout))

	return
}

func (self *ChannelService) GetChannelList(registryAddress typing.PaymentNetworkID,
	tokenAddress *typing.TokenAddress, partnerAddress *typing.Address) *list.List {

	result := list.New()
	chainState := self.StateFromChannel()

	if tokenAddress != nil && partnerAddress != nil {
		channelState := transfer.GetChannelStateFor(chainState,
			registryAddress, *tokenAddress, *partnerAddress)

		if channelState != nil {
			result.PushBack(channelState)
		}
	} else if tokenAddress != nil {
		result = transfer.ListChannelStateForTokenNetwork(chainState, registryAddress,
			*tokenAddress)
	} else {
		result = transfer.ListAllChannelState(chainState)
	}

	return result
}

func (self *ChannelService) GetNodeNetworkState(nodeAddress typing.Address) string {
	return transfer.GetNodeNetworkStatus(self.StateFromChannel(), nodeAddress)
}

func (self *ChannelService) GetTokensList(registryAddress typing.PaymentNetworkID) *list.List {
	tokensList := transfer.GetTokenNetworkAddressesFor(self.StateFromChannel(),
		registryAddress)

	return tokensList
}

func (self *ChannelService) TransferAndWait(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID, transferTimeout int) {

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

func (self *ChannelService) TransferAsync(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID) *chan int {

	asyncResult := new(chan int)

	//[TODO] Adding Graph class to hold route information, support async transfer
	// by calling channel mediated_transfer_async
	return asyncResult
}

func (self *ChannelService) DirectTransferAsync(amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID) (chan bool, error) {

	//Only one payment network
	paymentNetworkIdentifier := typing.PaymentNetworkID{}
	tokenAddress := typing.TokenAddress{}
	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(),
		paymentNetworkIdentifier,
		tokenAddress)
	self.transport.StartHealthCheck(typing.Address(target))

	if identifier == typing.PaymentID(0) {
		identifier = CreateDefaultIdentifier()
	}

	paymentStatus, exist := self.GetPaymentStatus(typing.Address(target), identifier)
	if exist {
		if !paymentStatus.Match(PAYMENT_DIRECT, tokenNetworkIdentifier, amount) {
			return nil, fmt.Errorf("Another payment with same id is in flight")
		}
		return paymentStatus.paymentDone, nil
	}

	directTransfer := &transfer.ActionTransferDirect{
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		ReceiverAddress:        typing.Address(target),
		PaymentIdentifier:      identifier,
		Amount:                 amount,
	}

	self.RegisterPaymentStatus(target, identifier, PAYMENT_DIRECT, amount, tokenNetworkIdentifier)
	paymentStatus, _ = self.GetPaymentStatus(typing.Address(target), identifier)

	self.HandleStateChange(directTransfer)

	return paymentStatus.paymentDone, nil

}

func (self *ChannelService) GetEventsPaymentHistoryWithTimestamps(tokenAddress typing.TokenAddress,
	targetAddress typing.Address, limit int, offset int) *list.List {

	result := list.New()

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(self.StateFromChannel(),
		typing.PaymentNetworkID{}, tokenAddress)

	events := self.Wal.Storage.GetEventsWithTimestamps(limit, offset)
	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		if EventFilterForPayments(event.WrappedEvent, tokenNetworkIdentifier, targetAddress) == true {
			result.PushBack(event)
		}

	}

	return result

}

func (self *ChannelService) GetEventsPaymentHistory(tokenAddress typing.TokenAddress,
	targetAddress typing.Address, limit int, offset int) *list.List {
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

func (self *ChannelService) GetBlockchainEventsNetwork(registryAddress typing.PaymentNetworkID,
	fromBlock typing.BlockHeight, toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}

func (self *ChannelService) GetBlockchainEventsTokenNetwork(tokenAddress typing.TokenAddress,
	fromBlock typing.BlockHeight, toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}

func (self *ChannelService) GetBlockchainEventsChannel(tokenAddress typing.TokenAddress,
	partnerAddress typing.Address, fromBlock typing.BlockHeight,
	toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}

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
