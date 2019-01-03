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
	"github.com/oniio/oniChannel/account"
	"github.com/oniio/oniChannel/blockchain"
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

	address          typing.Address
	blockchainEvents *blockchain.BlockchainEvents

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
	account *account.Account,
	channel_event_handler *ChannelEventHandler,
	messageHandler *MessageHandler,
	config map[string]string,
	discovery *network.ContractDiscovery) *ChannelService {

	self := new(ChannelService)

	self.chain = chain
	self.queryStartBlock = queryStartBlock
	//self.defaultSecretRegistry = defaultSecretRegistry
	self.config = config
	self.Account = account
	self.targetsToIndentifierToStatues = make(map[typing.Address]*sync.Map)
	self.tokennetworkidsToConnectionmanagers = make(map[typing.TokenNetworkID]*ConnectionManager)
	self.ReceiveNotificationChannels = make(map[chan *transfer.EventPaymentReceivedSuccess]struct{})

	// address in the blockchain service is set when import wallet
	self.address = chain.Address
	self.discovery = discovery

	self.blockchainEvents = new(blockchain.BlockchainEvents)
	selfEventHandler = new(ChannelEventHandler)
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

		selfEventHandler.OnChannelEvent(self, temp.(transfer.Event))
	}

	return eventList
}

func (self *ChannelService) SetNodeNetworkState(nodeAddress typing.Address,
	networkState string) {

	return
}

func (self *ChannelService) StartHealthCheckFor(nodeAddress typing.Address) {
	self.transport.StartHealthCheck(nodeAddress)
	return
}

func (self *ChannelService) StartNeighboursHealthcheck() {
	neighbours := transfer.AllNeighbourNodes(self.StateFromChannel())

	for k := range neighbours {
		self.StartHealthCheckFor(k)
	}

	return
}

func (self *ChannelService) InitializeTransactionsQueues(chainState *transfer.ChainState) {
	pendingTransactions := transfer.GetPendingTransactions(chainState)

	for _, transaction := range pendingTransactions {
		selfEventHandler.OnChannelEvent(self, transaction)
	}
}

func (self *ChannelService) RegisterPaymentStatus(target typing.TargetAddress, identifier typing.PaymentID, paymentType PaymentType, amount typing.TokenAmount, tokenNetworkIdentifier typing.TokenNetworkID) {
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
		self.StartHealthCheckFor(queueIdentifier.Recipient)

		for _, event := range eventQueue {

			switch event.(type) {
			case transfer.SendDirectTransfer:
				e := event.(transfer.SendDirectTransfer)

				self.RegisterPaymentStatus(typing.TargetAddress(e.Recipient), e.PaymentIdentifier, PAYMENT_DIRECT, e.BalanceProof.TransferredAmount, e.BalanceProof.TokenNetworkIdentifier)

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

	events, err := self.chain.Client.GetFilterArgsForAllEventsFromChannel(0, fromBlock, toBlock)
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

func (self *ChannelService) DirectTransferAsync(
	tokenNetworkIdentifier typing.TokenNetworkID,
	amount typing.TokenAmount,
	target typing.TargetAddress,
	identifier typing.PaymentID) (chan bool, error) {

	self.StartHealthCheckFor(typing.Address(target))

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
