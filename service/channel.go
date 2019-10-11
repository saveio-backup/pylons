package service

import (
	"bytes"
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

	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network"
	"github.com/saveio/pylons/network/secretcrypt"
	"github.com/saveio/pylons/network/transport"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/storage"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/account"
	comm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	mpay "github.com/saveio/themis/smartcontract/service/native/micropayment"
	scUtils "github.com/saveio/themis/smartcontract/service/native/utils"
)

type ChannelService struct {
	chain           *network.BlockChainService
	queryStartBlock common.BlockHeight

	Account                      *account.Account
	channelEventHandler          *ChannelEventHandler
	messageHandler               *MessageHandler
	config                       map[string]string
	Transport                    *transport.Transport
	targetsToIdentifierToStatues map[common.Address]*sync.Map
	ReceiveNotificationChannels  map[chan *transfer.EventPaymentReceivedSuccess]struct{}
	channelWithdrawStatus        *sync.Map
	channelNewNotifier           chan *NewChannelNotification
	isRestoreFinish              bool

	address            common.Address
	microAddress       common.Address
	dispatchEventsLock sync.Mutex
	eventPollLock      sync.Mutex
	databasePath       string
	databaseDir        string
	lockFile           string

	alarm         *AlarmTask
	Wal           *storage.WriteAheadLog
	snapshotGroup int
	firstRun      bool

	tokenNetworkIdsToConnectionManagers map[common.TokenNetworkID]*ConnectionManager
	lastFilterBlock                     common.BlockHeight
	statusLock                          sync.RWMutex
}

type PaymentStatus struct {
	paymentType    common.PaymentType
	paymentId      common.PaymentID
	amount         common.TokenAmount
	TokenNetworkId common.TokenNetworkID
	paymentDone    chan bool //used to notify send success/fail
}

func (self *PaymentStatus) Match(paymentType common.PaymentType, tokenNetworkId common.TokenNetworkID,
	amount common.TokenAmount) bool {
	if self.paymentType == paymentType && self.TokenNetworkId == tokenNetworkId &&
		self.amount == amount {
		return true
	} else {
		return false
	}
}

func NewChannelService(chain *network.BlockChainService, queryStartBlock common.BlockHeight,
	microPayAddress common.Address, messageHandler *MessageHandler, config map[string]string) *ChannelService {
	var err error
	if chain == nil {
		log.Error("error in create new channel service: chain service not available")
		return nil
	}
	self := new(ChannelService)

	self.chain = chain
	self.queryStartBlock = queryStartBlock
	self.microAddress = microPayAddress
	self.config = config
	self.Account = chain.GetAccount()
	self.targetsToIdentifierToStatues = make(map[common.Address]*sync.Map)
	self.tokenNetworkIdsToConnectionManagers = make(map[common.TokenNetworkID]*ConnectionManager)
	self.ReceiveNotificationChannels = make(map[chan *transfer.EventPaymentReceivedSuccess]struct{})
	self.channelWithdrawStatus = new(sync.Map)
	self.channelNewNotifier = make(chan *NewChannelNotification, 1)

	// address in the blockChain service is set when import wallet
	self.address = chain.Address
	self.channelEventHandler = new(ChannelEventHandler)
	self.messageHandler = messageHandler
	self.Transport = transport.NewTransport(self)
	self.alarm = NewAlarmTask(chain)
	self.firstRun = true

	networkId, err := self.chain.ChainClient.GetNetworkId()
	if err != nil {
		log.Error("get network id failed:", err)
		return nil
	}
	log.Debug("[NewChannelService], NetworkId: ", networkId)

	customDBPath, _ := config["database_path"]
	self.setDBPath(customDBPath, networkId)
	self.initDB()
	return self
}

func (self *ChannelService) setDBPath(customDBPath string, dbId uint32) {
	var dbPath string

	if filepath.IsAbs(customDBPath) {
		dbPath = customDBPath
		log.Debug("[setDBPath] dbPath: ", dbPath)
	} else {
		currentPath, err := GetFullDatabasePath()
		if err != nil {
			self.databaseDir = ""
			self.databasePath = ":memory:"
			log.Warn("[setDBPath] get full db path failed,use memory database")
			return
		}
		log.Debug("[setDBPath] currentPath: ", currentPath)
		if customDBPath == "" || customDBPath == "." {
			dbPath = currentPath
			log.Debug("[setDBPath] dbPath: ", dbPath)
		} else {
			customPath := strings.Trim(customDBPath, ".")
			customPath = strings.Trim(customPath, "/")
			customPath = strings.Trim(customPath, "\\")

			dbPath = filepath.Join(currentPath, customPath)
			log.Debug("[setDBPath] dbPath: ", dbPath)
			if !common.PathExists(dbPath) {
				err := os.Mkdir(dbPath, os.ModePerm)
				if err != nil {
					log.Error("[setDBPath] Mkdir error: ", err.Error())
				}
			}
		}
	}

	dbDirName := fmt.Sprintf("channelDB-%d", dbId)

	self.databaseDir = filepath.Join(dbPath, dbDirName)
	if !common.PathExists(self.databaseDir) {
		err := os.Mkdir(self.databaseDir, os.ModePerm)
		if err != nil {
			log.Error("[setDBPath] Mkdir error: ", err.Error())
		}
	}

	self.databasePath = filepath.Join(self.databaseDir, "channel.db")
	log.Info("[setDBPath] database set to", self.databasePath)
}

func (self *ChannelService) initDB() error {
	sqliteStorage, err := storage.NewSQLiteStorage(self.databasePath)
	if err != nil {
		log.Error("create db failed:", err)
		return err
	}
	stateChangeQty := sqliteStorage.CountStateChanges()
	self.snapshotGroup = stateChangeQty / common.Config.SnapshotStateChangeCount

	var lastLogBlockHeight common.BlockHeight
	self.Wal = storage.RestoreToStateChange(transfer.StateTransition, sqliteStorage, "latest",
		self.address)
	self.isRestoreFinish = true

	if self.Wal.StateManager.CurrentState == nil {
		var stateChange transfer.StateChange
		networkId, err := self.chain.ChainClient.GetNetworkId()
		if err != nil {
			log.Error("get network id failed:", err)
			return err
		}

		chainNetworkId := common.ChainID(networkId)
		stateChange = &transfer.ActionInitChain{
			BlockHeight: common.BlockHeight(0),
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
		chainState := self.StateFromChannel()
		lastLogBlockHeight = transfer.GetBlockHeight(chainState)
		err = self.checkAddressIntegrity(chainState)
		if err != nil {
			log.Errorf("check address integrity failed: %s", err)
			return err
		}
		log.Infof("Restored state from WAL,last log BlockHeight=%d", lastLogBlockHeight)
	}
	//set filter start block number
	self.lastFilterBlock = lastLogBlockHeight
	log.Info("db setup done")
	return nil
}

func (self *ChannelService) SyncBlockData() error {
	self.alarm.RegisterCallback(self.CallbackNewBlock)
	if err := self.alarm.FirstRun(); err != nil {
		log.Error("[SyncBlockData] run alarm call back failed:", err)
		return err
	}
	return nil
}

func (self *ChannelService) StartService() error {
	self.UpdateRouteMap()
	chainState := self.StateFromChannel()
	self.InitializeTransactionsQueues(chainState)
	self.InitializeMessagesQueues(chainState)

	self.alarm.Start()
	log.Info("channel service started")
	return nil
}

func (self *ChannelService) checkAddressIntegrity(chainState *transfer.ChainState) error {
	if chainState != nil && !common.AddressEqual(self.address, chainState.Address) {
		return fmt.Errorf("[checkAddressIntegrity] failed, self.address : %s, chainState.Address : %s",
			common.ToBase58(self.address), common.ToBase58(chainState.Address))
	}
	return nil
}

func (self *ChannelService) StopService() {
	self.Transport.Stop()
	self.alarm.Stop()
	self.Wal.Storage.Close()
	log.Info("channel service stopped")
}

func (self *ChannelService) HandleStateChange(stateChange transfer.StateChange) []transfer.Event {
	self.dispatchEventsLock.Lock()
	defer self.dispatchEventsLock.Unlock()

	eventList := self.Wal.LogAndDispatch(stateChange, self.address)
	for _, e := range eventList {
		self.channelEventHandler.OnChannelEvent(self, e.(transfer.Event))
	}
	//take snapshot

	newSnapShotGroup := self.Wal.StateChangeId / common.Config.SnapshotStateChangeCount
	if newSnapShotGroup > self.snapshotGroup {
		log.Infof("storing snapshot, Snapshot Id = %d, LastFilterBlockHeight = %d", newSnapShotGroup, self.GetLastFilterBlock())
		self.Wal.Snapshot()
		self.snapshotGroup = newSnapShotGroup
	}
	return eventList
}

func (self *ChannelService) StartNeighboursHealthCheck() {
	neighbours := transfer.GetNeighbours(self.StateFromChannel())
	for _, v := range neighbours {
		log.Debugf("[StartNeighboursHealthCheck] Neighbour: %s", common.ToBase58(v))
		self.Transport.StartHealthCheck(v)
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
	amount common.TokenAmount, tokenNetworkId common.TokenNetworkID) *PaymentStatus {
	status := PaymentStatus{
		paymentType:    paymentType,
		paymentId:      identifier,
		amount:         amount,
		TokenNetworkId: tokenNetworkId,
		paymentDone:    make(chan bool, 1),
	}

	self.statusLock.Lock()
	defer self.statusLock.Unlock()

	if payments, exist := self.targetsToIdentifierToStatues[common.Address(target)]; exist {
		payments.Store(identifier, status)
	} else {
		payments := new(sync.Map)
		payments.Store(identifier, status)
		self.targetsToIdentifierToStatues[common.Address(target)] = payments
	}

	return &status
}

func (self *ChannelService) GetPaymentStatus(target common.Address, identifier common.PaymentID) (status *PaymentStatus, exist bool) {
	self.statusLock.RLock()
	defer self.statusLock.RUnlock()

	payments, exist := self.targetsToIdentifierToStatues[common.Address(target)]
	if exist {
		paymentStatus, ok := payments.Load(identifier)
		if ok {
			pmStatus := paymentStatus.(PaymentStatus)
			return &pmStatus, true
		}
	}

	return nil, false
}

func (self *ChannelService) RemovePaymentStatus(target common.Address, identifier common.PaymentID) (ok bool) {
	self.statusLock.Lock()
	defer self.statusLock.Unlock()

	payments, exist := self.targetsToIdentifierToStatues[common.Address(target)]
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
	log.Debug("InitializeMessagesQueues len(chainState.PendingTransactions): ", len(chainState.PendingTransactions))

	for queueId, eventQueue := range eventsQueues {
		self.Transport.StartHealthCheck(queueId.Recipient)
		log.Debug("[InitializeMessagesQueues] : ChannelId", queueId.ChannelId)

		for _, event := range eventQueue {
			log.Debugf("eventsQueues event type = %s", reflect.ValueOf(event).Type().String())
			switch event.(type) {
			case *transfer.SendDirectTransfer:
				e := event.(*transfer.SendDirectTransfer)
				self.RegisterPaymentStatus(common.Address(e.Recipient), e.PaymentId, common.PAYMENT_DIRECT,
					e.BalanceProof.TransferredAmount, e.BalanceProof.TokenNetworkId)
			case *transfer.SendLockedTransfer:
				e := event.(*transfer.SendLockedTransfer)
				self.RegisterPaymentStatus(common.Address(e.Recipient), e.Transfer.PaymentId, common.PAYMENT_MEDIATED,
					e.Transfer.BalanceProof.TransferredAmount, e.Transfer.BalanceProof.TokenNetworkId)
			}

			message := messages.MessageFromSendEvent(event)
			if message != nil {
				err := self.Sign(message)
				if err != nil {
					log.Error("[InitializeMessagesQueues] Sign: ", err.Error())
				}
				log.Debugf("SendAsync event type = %s", reflect.ValueOf(event).Type().String())
				err = self.Transport.SendAsync(&queueId, message)
				if err != nil {
					log.Error("[InitializeMessagesQueues] SendAsync: ", err.Error())
				}
			}
		}
	}
}

func (self *ChannelService) GetLastFilterBlock() common.BlockHeight {
	return self.lastFilterBlock
}

func (self *ChannelService) UpdateRouteMap() {
	tokenNetwork := transfer.GetTokenNetworkByIdentifier(self.StateFromChannel(), common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS))
	allOpenChannels, err := self.chain.ChannelClient.GetAllOpenChannels()
	if err != nil {
		log.Errorf("[UpdateRouteMap] GetAllOpenChannels error: %s", err.Error())
		return
	}
	for i := uint64(0); i < allOpenChannels.ParticipantNum; i++ {
		openedChannel := allOpenChannels.Participants[i]
		var partAddr1, partAddr2 common.Address
		copy(partAddr1[:], openedChannel.Part1Addr[:20])
		copy(partAddr2[:], openedChannel.Part2Addr[:20])
		log.Infof("[UpdateRouteMap], AddRoute ChannelId: %d", openedChannel.ChannelID)
		tokenNetwork.AddRoute(partAddr1, partAddr2, common.ChannelID(openedChannel.ChannelID))
		if common.AddressEqual(partAddr1, self.address) {
			if err = self.Transport.StartHealthCheck(partAddr2); err != nil {
				log.Errorf("[UpdateRouteMap] StartHealthCheck PartAddr2: %s error: %s",
					common.ToBase58(partAddr2), err.Error())
			}
		}
	}
}

func (self *ChannelService) CallbackNewBlock() {
	chainBlockHeight, err := self.chain.BlockHeight()
	if err != nil {
		log.Errorf("[CallbackNewBlock] GetBlockHeight error: %s", err.Error())
		return
	}
	if chainBlockHeight <= self.lastFilterBlock {
		return
	}

	if self.firstRun || chainBlockHeight-self.lastFilterBlock > 10 {
		for {
			bgnBlockHeight := self.lastFilterBlock + 1
			endBlockHeight, err := self.chain.BlockHeight()
			if err != nil {
				log.Errorf("[CallbackNewBlock] GetBlockHeight error: %s", err.Error())
				time.Sleep(3 * time.Second)
				continue
			}
			if endBlockHeight-self.lastFilterBlock < 3 {
				log.Infof("[CallbackNewBlock] FastSync done. LastFilterBlock(%d) ChainBlockHeight(%d)",
					self.lastFilterBlock, endBlockHeight)
				break
			}
			log.Infof("[CallbackNewBlock] FastSync begin, From: %d, To: %d", bgnBlockHeight, endBlockHeight)

			events, posMap, err := self.chain.ChannelClient.GetFilterArgsForAllEventsFromChannelByEventId(
				comm.Address(self.microAddress), self.Account.Address, 0, uint32(bgnBlockHeight), uint32(endBlockHeight))
			if err != nil {
				log.Errorf("CallbackNewBlock GetFilterArgsForAllEventsFromChannelByEventId error: %s", err)
				return
			}

			secretRevealEvents, secretRevealPos, err := self.chain.ChannelClient.GetFilterArgsForAllEventsFromChannelByEventId(
				comm.Address(self.microAddress), comm.ADDRESS_EMPTY, mpay.EVENT_SECRET_REVEALED, uint32(bgnBlockHeight), uint32(endBlockHeight))
			if err != nil {
				log.Errorf("CallbackNewBlock GetFilterArgsForAllEventsFromChannelByEventId for secret reveal error: %s", err)
				return
			}

			var start uint32
			var end uint32
			for i := bgnBlockHeight; i <= endBlockHeight; i++ {
				// handle channel events releate with self
				if pos, ok := posMap[uint32(i)]; ok {
					start = pos[0]
					end = pos[1]
					if end == 0 {
						end = start
					}

					eventsSlice := events[start : end+1]
					for _, event := range eventsSlice {
						OnBlockchainEvent(self, event)
					}
				}

				// handle secret reveal events
				if secPos, ok := secretRevealPos[uint32(i)]; ok {
					start = secPos[0]
					end = secPos[1]
					if end == 0 {
						end = start
					}

					eventsSlice := secretRevealEvents[start : end+1]
					for _, event := range eventsSlice {
						OnBlockchainEvent(self, event)
					}
				}

				block := &transfer.Block{}
				block.BlockHeight = common.BlockHeight(i)
				block.BlockHash = common.BlockHash{}
				self.HandleStateChange(block)
				self.lastFilterBlock = common.BlockHeight(i)
			}
		}
		if self.firstRun == false {
			self.UpdateRouteMap()
		}
		self.firstRun = false
	} else {
		bgnBlockHeight := self.lastFilterBlock + 1
		log.Infof("[CallbackNewBlock] From: %d, To: %d, BlockCount: %d", bgnBlockHeight, chainBlockHeight,
			chainBlockHeight-bgnBlockHeight+1)
		for i := bgnBlockHeight; i <= chainBlockHeight; i++ {
			events, err := self.chain.ChannelClient.GetFilterArgsForAllEventsFromChannel(0, uint32(i), uint32(i))
			if err != nil {
				log.Errorf("[CallbackNewBlock] GetFilterArgsForAllEventsFromChannel error: %s", err)
				return
			}

			for _, event := range events {
				OnBlockchainEvent(self, event)
			}
			// blockHash not used
			/*
				hash, err := self.chain.ChannelClient.Client.GetBlockHash(uint32(i))
				if err != nil {
					return
				}
			*/
			block := &transfer.Block{}
			block.BlockHeight = i
			//block.BlockHash = common.BlockHash(hash[:])
			block.BlockHash = common.BlockHash{}
			self.HandleStateChange(block)
			self.lastFilterBlock = i
		}
	}
	log.Infof("[CallbackNewBlock] finish")
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

func (self *ChannelService) ConnectionManagerForTokenNetwork(tokenNetworkId common.TokenNetworkID) *ConnectionManager {
	var manager *ConnectionManager
	var exist bool
	manager, exist = self.tokenNetworkIdsToConnectionManagers[tokenNetworkId]
	if exist == false {
		manager = NewConnectionManager(self, tokenNetworkId)
		self.tokenNetworkIdsToConnectionManagers[tokenNetworkId] = manager
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
	if len(self.tokenNetworkIdsToConnectionManagers) > 0 {
		WaitForSettleAllChannels(self, self.alarm.GetInterval())
	}
	return
}

func CreateDefaultId() common.PaymentID {
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
	tokenNetworkState.Address = common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)
	tokenNetworkState.TokenAddress = common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
	newTokenNetwork := &transfer.ContractReceiveNewTokenNetwork{PaymentNetworkId: common.PaymentNetworkID(scUtils.MicroPayContractAddress),
		TokenNetwork: tokenNetworkState}

	self.HandleStateChange(newTokenNetwork)
}

func (self *ChannelService) GetPaymentChannelArgs(tokenNetworkId common.TokenNetworkID, channelId common.ChannelID) map[string]interface{} {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateByTokenNetworkId(chainState, tokenNetworkId, channelId)
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

func EventFilterForPayments(event transfer.Event, tokenNetworkId common.TokenNetworkID,
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
		} else if eventPaymentReceivedSuccess.Initiator == partnerAddress {
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

func (self *ChannelService) GetChannel(registryAddress common.PaymentNetworkID, tokenAddress common.TokenAddress,
	partnerAddress common.Address) *transfer.NettingChannelState {

	var result *transfer.NettingChannelState
	channelList := self.GetChannelList(registryAddress, tokenAddress, partnerAddress)
	if channelList.Len() != 0 {
		result = channelList.Back().Value.(*transfer.NettingChannelState)
	} else {
		log.Debug("[GetChannelList] Len = 0")
	}

	return result
}

func (self *ChannelService) tokenNetworkConnect(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress, funds common.TokenAmount,
	initialChannelTarget int, joinableFundsTarget float32) {

	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		TokenNetworkId)

	connectionManager.connect(funds, initialChannelTarget, joinableFundsTarget)

	return
}

func (self *ChannelService) tokenNetworkLeave(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)
	connectionManager := self.ConnectionManagerForTokenNetwork(
		TokenNetworkId)

	return connectionManager.Leave(registryAddress)
}

func (self *ChannelService) GetChannelId(
	partnerAddress common.Address) common.ChannelID {
	id, err := self.chain.ChannelClient.GetChannelIdentifier(comm.Address(self.address),
		comm.Address(partnerAddress))
	if err != nil {
		log.Error("get channel identifier failed ", err)
		return 0
	}
	return common.ChannelID(id)
}

func (self *ChannelService) OpenChannel(tokenAddress common.TokenAddress,
	partnerAddress common.Address) (common.ChannelID, error) {
	id, err := self.chain.ChannelClient.GetChannelIdentifier(comm.Address(self.address),
		comm.Address(partnerAddress))
	if err != nil {
		err = fmt.Errorf("[OpenChannel] get channel identifier failed %s", err)
		log.Errorf("[OpenChannel] error: %s", err.Error())
		return 0, err
	}
	regAddr := common.ToBase58(self.address)
	patAddr := common.ToBase58(partnerAddress)
	if id != 0 {
		if err = self.Transport.StartHealthCheck(partnerAddress); err != nil {
			log.Warnf("[OpenChannel] StartHealthCheck warn: %s", err.Error())
		}
		log.Infof("channel between %s and %s already setup, id: %d", regAddr, patAddr, id)
		return common.ChannelID(id), nil
	}

	log.Infof("channel between %s and %s haven`t setup. start to create new one", regAddr, patAddr)
	settleTimeout, err := strconv.Atoi(self.config["settle_timeout"])
	if err != nil {
		err = fmt.Errorf("[OpenChannel] faile to parse settle timeout %s", err)
		log.Errorf("[OpenChannel] error: %s", err.Error())
		return 0, err
	}

	log.Infof("[OpenChannel] try to connect :%s", common.ToBase58(partnerAddress))
	if err = self.Transport.StartHealthCheck(partnerAddress); err != nil {
		log.Warnf("[OpenChannel] StartHealthCheck warn: %s", err.Error())
	}

	tokenNetwork := self.chain.NewTokenNetwork(common.Address(usdt.USDT_CONTRACT_ADDRESS))
	channelId := tokenNetwork.NewNettingChannel(partnerAddress, settleTimeout)
	if channelId == 0 {
		err = fmt.Errorf("NewNettingChannel open channel failed")
		log.Errorf("[OpenChannel] error: %s", err.Error())
		return 0, err
	}
	log.Info("wait for new channel ... ")
	channelState := WaitForNewChannel(self, common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), partnerAddress,
		common.Config.OpenChannelRetryTimeOut, common.Config.OpenChannelRetryTimes)
	if channelState == nil {
		err = fmt.Errorf("WaitForNewChannel setup channel timeout")
		log.Errorf("[OpenChannel] error: %s", err.Error())
		return 0, err
	}
	log.Infof("[OpenChannel] new channel between %s and %s has setup, channel ID = %d", regAddr, patAddr,
		channelState.Identifier)

	chainState := self.StateFromChannel()

	tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, channelState.TokenNetworkId)
	tokenNetworkState.AddRoute(self.Address(), partnerAddress, channelState.Identifier)
	return channelState.Identifier, nil
}

func (self *ChannelService) SetTotalChannelDeposit(tokenAddress common.TokenAddress, partnerAddress common.Address,
	totalDeposit common.TokenAmount) error {

	chainState := self.StateFromChannel()
	partAddr := common.ToBase58(partnerAddress)
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress),
		tokenAddress, partnerAddress)
	if channelState == nil {
		log.Errorf("deposit failed, can not find specific channel with %s", partAddr)
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

	if totalDeposit < channelState.OurState.ContractBalance {
		return errors.New("totalDeposit must big than contractBalance")
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
	err = WaitForParticipantNewBalance(self, common.PaymentNetworkID(self.microAddress), tokenAddress, partnerAddress,
		targetAddress, totalDeposit, common.Config.DepositRetryTimeout)

	if err != nil {
		log.Errorf("wait for new balance failed, err", err)
	}
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
	registryAddress := common.PaymentNetworkID(self.microAddress)
	channelsToClose := transfer.FilterChannelsByPartnerAddress(chainState, registryAddress,
		tokenAddress, partnerAddress)

	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(
		chainState, registryAddress, tokenAddress)

	//[TODO] use BlockChainService.PaymentChannel to get PaymentChannels and
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
		channelClose.TokenNetworkId = TokenNetworkId
		channelClose.ChannelId = identifier

		self.HandleStateChange(channelClose)

		tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, channelState.TokenNetworkId)
		tokenNetworkState.DelRoute(channelState.Identifier)
	}

	WaitForClose(self, registryAddress, tokenAddress, channelIds, float32(retryTimeout))

	return
}

//fwtodo: may need to add channel to notify the result ? following send will be disallowed
func (self *ChannelService) ChannelCooperativeSettle(tokenAddress common.TokenAddress, partnerAddress common.Address) error {

	chainState := self.StateFromChannel()
	registryAddress := common.PaymentNetworkID(self.microAddress)
	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(
		chainState, registryAddress, tokenAddress)

	channelState := transfer.GetChannelStateByTokenNetworkAndPartner(chainState, TokenNetworkId, partnerAddress)
	if channelState == nil {
		return fmt.Errorf("ChannelCooperativeSettle error, no channel found with the partner")
	}

	identifier := channelState.GetIdentifier()

	ChannelCooperativeSettle := new(transfer.ActionCooperativeSettle)
	ChannelCooperativeSettle.TokenNetworkId = TokenNetworkId
	ChannelCooperativeSettle.ChannelId = identifier

	self.HandleStateChange(ChannelCooperativeSettle)

	return nil
}

func (self *ChannelService) GetChannelList(registryAddress common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddress common.Address) *list.List {

	result := list.New()
	chainState := self.StateFromChannel()

	if tokenAddress != common.EmptyTokenAddress && partnerAddress != common.EmptyAddress {
		channelState := transfer.GetChannelStateFor(chainState, registryAddress,
			tokenAddress, partnerAddress)

		if channelState != nil {
			result.PushBack(channelState)
		} else {
			log.Debug("[GetChannelList] channelState == nil")
		}
	} else if tokenAddress != common.EmptyTokenAddress {
		result = transfer.ListChannelStateForTokenNetwork(chainState, registryAddress,
			tokenAddress)
	} else {
		result = transfer.ListAllChannelState(chainState)
	}

	return result
}

func (self *ChannelService) GetNodeNetworkState(nodeAddress common.Address) string {
	return self.Transport.GetNodeNetworkState(nodeAddress)
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
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
	paymentNetworkId := common.PaymentNetworkID(self.microAddress)

	channelState := transfer.GetChannelStateFor(chainState, paymentNetworkId, tokenAddress, target)
	if chainState == nil {
		return false
	}
	if channelState == nil {
		return false
	}
	if channelState.OurState == nil {
		return false
	}
	if channelState.OurState.GetGasBalance() < amount {
		return false
	} else {
		return true
	}
}

func (self *ChannelService) DirectTransferAsync(amount common.TokenAmount, target common.Address,
	identifier common.PaymentID) (chan bool, error) {
	var err error
	if target == common.EmptyAddress {
		log.Error("target address is invalid:", target)
		return nil, fmt.Errorf("target address is invalid")
	}
	chainState := self.StateFromChannel()
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
	paymentNetworkId := common.PaymentNetworkID(self.microAddress)
	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(chainState,
		paymentNetworkId, tokenAddress)

	if err = self.Transport.StartHealthCheck(common.Address(target)); err != nil {
		log.Errorf("[DirectTransferAsync] StartHealthCheck error: %s", err.Error())
		return nil, err
	}

	if identifier == common.PaymentID(0) {
		identifier = CreateDefaultId()
	}

	paymentStatus, exist := self.GetPaymentStatus(common.Address(target), identifier)
	if exist {
		if !paymentStatus.Match(common.PAYMENT_DIRECT, TokenNetworkId, amount) {
			return nil, errors.New("Another payment with same id is in flight. ")
		}
		log.Warn("payment already existed:")
		return paymentStatus.paymentDone, nil
	}

	directTransfer := &transfer.ActionTransferDirect{
		TokenNetworkId:  TokenNetworkId,
		ReceiverAddress: common.Address(target),
		PaymentId:       identifier,
		Amount:          amount,
	}

	paymentStatus = self.RegisterPaymentStatus(target, identifier, common.PAYMENT_DIRECT, amount, TokenNetworkId)

	self.HandleStateChange(directTransfer)
	return paymentStatus.paymentDone, nil
}

func (self *ChannelService) CheckPayRoute(mediaAddr common.Address, targetAddr common.Address) (bool, error) {
	if state := self.Transport.GetNodeNetworkState(mediaAddr); state != transfer.NetworkReachable {
		return false, fmt.Errorf("[CheckPayRoute] MediaNetworkState(%s) status is not reachable", common.ToBase58(mediaAddr))
	}
	if state := self.Transport.GetNodeNetworkState(targetAddr); state != transfer.NetworkReachable {
		log.Warn("[CheckPayRoute] TargetNetworkState(%s) status is not reachable", common.ToBase58(targetAddr))
	}
	return true, nil
}

func (self *ChannelService) MediaTransfer(registryAddress common.PaymentNetworkID, tokenAddress common.TokenAddress,
	media common.Address, target common.Address, amount common.TokenAmount, paymentId common.PaymentID) (chan bool, error) {

	var err error
	if target == common.EmptyAddress {
		log.Error("target address is invalid:", target)
		return nil, fmt.Errorf("target address is invalid")
	}
	if amount <= 0 {
		log.Error("amount negative:", amount)
		return nil, fmt.Errorf("amount negative ")
	}

	//log.Infof("[MediaTransfer] try to connect :%s", common.ToBase58(target))
	if err = self.Transport.StartHealthCheck(target); err != nil {
		log.Errorf("[MediaTransfer] StartHealthCheck error: %s", err.Error())
		return nil, err
	}

	chainState := self.StateFromChannel()

	//TODO: check validTokens
	paymentNetworkId := common.PaymentNetworkID(self.microAddress)
	tokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(chainState, paymentNetworkId, tokenAddress)

	secret := common.SecretRandom(constants.SecretLen)
	log.Debug("[MediaTransfer] Secret: ", secret)

	secretHash := common.GetHash(secret)
	secretRegistry := self.chain.SecretRegistry(common.SecretRegistryAddress(usdt.USDT_CONTRACT_ADDRESS))
	if secretRegistry.IsSecretRegistered(secretHash) {
		log.Errorf("SecretHash %v has been registerd onchain", secretHash)
		return nil, fmt.Errorf("SecretHash %v has been registerd onchain", secretHash)
	}

	if paymentId == common.PaymentID(0) {
		paymentId = CreateDefaultId()
	}

	//with self.payment_identifier_lock:
	paymentStatus, exist := self.GetPaymentStatus(target, paymentId)
	if exist {
		paymentStatusMatches := paymentStatus.Match(common.PAYMENT_MEDIATED, tokenNetworkId, amount)
		if !paymentStatusMatches {
			return nil, fmt.Errorf("Another payment with the same id is in flight. ")
		}
		return paymentStatus.paymentDone, nil
	}

	asyncDone := make(chan bool)
	paymentStatus = &PaymentStatus{
		paymentType:    common.PAYMENT_MEDIATED,
		paymentId:      paymentId,
		amount:         amount,
		TokenNetworkId: tokenNetworkId,
		paymentDone:    asyncDone,
	}

	paymentStatus = self.RegisterPaymentStatus(target, paymentId, common.PAYMENT_MEDIATED,
		amount, tokenNetworkId)

	encSecret, err := secretcrypt.SecretCryptService.EncryptSecret(target, secret)
	if err != nil {
		log.Errorf("Encrypt secret error: %s", err.Error())
		return nil, fmt.Errorf("Encrypt secret error: %s", err.Error())
	}

	var actionInitiator *transfer.ActionInitInitiator
	if media == common.EmptyAddress {
		actionInitiator, err = self.InitiatorInit(paymentId, amount, secret, tokenNetworkId, target, encSecret)
	} else {
		actionInitiator, err = self.InitiatorInitBySpecifiedMedia(paymentId, media, amount, secret, tokenNetworkId,
			target, encSecret)
	}
	if err != nil {
		log.Error("[MediaTransfer]:", err.Error())
		return nil, err
	}

	if ok, err := self.CheckPayRoute(actionInitiator.Routes[0].NodeAddress, target); !ok {
		log.Errorf("[MediaTransfer] CheckPayRoute error: %s", err.Error())
		return nil, err
	}

	//# Dispatch the state change even if there are no routes to create the
	//# wal entry.
	self.HandleStateChange(actionInitiator)
	return paymentStatus.paymentDone, nil
}

func (self *ChannelService) InitiatorInit(paymentId common.PaymentID, transferAmount common.TokenAmount,
	secret common.Secret, TokenNetworkId common.TokenNetworkID,
	targetAddress common.Address, encSecret common.EncSecret) (*transfer.ActionInitInitiator, error) {
	if 0 == bytes.Compare(secret, common.EmptySecretHash[:]) {
		return nil, fmt.Errorf("Should never end up initiating transfer with Secret 0x0 ")
	}

	secretHash := common.GetHash(secret)
	log.Debug("[InitiatorInit] secretHash: ", secretHash)

	transferState := &transfer.TransferDescriptionWithSecretState{
		PaymentNetworkId: common.PaymentNetworkID{},
		PaymentId:        paymentId,
		Amount:           transferAmount,
		TokenNetworkId:   TokenNetworkId,
		Initiator:        self.address,
		Target:           targetAddress,
		Secret:           secret,
		EncSecret:        encSecret,
		SecretHash:       secretHash,
	}

	var previousAddress common.Address
	routes, err := GetBestRoutes(self, TokenNetworkId, self.address, targetAddress, transferAmount,
		previousAddress)
	if err != nil {
		return nil, err
	}
	initInitiatorStateChange := &transfer.ActionInitInitiator{
		TransferDescription: transferState,
		Routes:              routes,
	}
	return initInitiatorStateChange, nil
}

func (self *ChannelService) InitiatorInitBySpecifiedMedia(paymentId common.PaymentID, media common.Address,
	transferAmount common.TokenAmount, secret common.Secret, TokenNetworkId common.TokenNetworkID,
	targetAddress common.Address, encSecret common.EncSecret) (*transfer.ActionInitInitiator, error) {
	if 0 == bytes.Compare(secret, common.EmptySecretHash[:]) {
		return nil, fmt.Errorf("Should never end up initiating transfer with Secret 0x0 ")
	}

	secretHash := common.GetHash(secret)
	log.Debug("[InitiatorInitBySpecifiedMedia] secretHash: ", secretHash)
	transferState := &transfer.TransferDescriptionWithSecretState{
		PaymentNetworkId: common.PaymentNetworkID{},
		PaymentId:        paymentId,
		Amount:           transferAmount,
		TokenNetworkId:   TokenNetworkId,
		Initiator:        self.address,
		Target:           targetAddress,
		Secret:           secret,
		EncSecret:        encSecret,
		SecretHash:       secretHash,
	}

	routes, err := GetSpecifiedRoute(self, TokenNetworkId, media, self.address, transferAmount)
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
	routes, err := GetBestRoutes(self, fromTransfer.BalanceProof.TokenNetworkId,
		self.address, common.Address(fromTransfer.Target), fromTransfer.Lock.Amount, initiatorAddr)
	if err != nil {
		log.Infof("[GetBestRoutes] error : %v", err)
	}
	fromRoute := &transfer.RouteState{
		NodeAddress: initiatorAddr,
		ChannelId:   fromTransfer.BalanceProof.ChannelId,
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
		NodeAddress: sender,
		ChannelId:   fromTransfer.BalanceProof.ChannelId,
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

	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(self.StateFromChannel(),
		common.PaymentNetworkID{}, tokenAddress)
	events := self.Wal.Storage.GetEventsWithTimestamps(limit, offset)
	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		if EventFilterForPayments(event.WrappedEvent, TokenNetworkId, targetAddress) == true {
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

func (self *ChannelService) GetHostAddr(nodeAddress common.Address) (string, error) {
	return self.Transport.GetHostAddr(nodeAddress)
}

func (self *ChannelService) SetHostAddrCallBack(getHostAddrCallback func(address common.Address) (string, error)) {
	self.Transport.SetGetHostAddrCallback(getHostAddrCallback)
}

func (self *ChannelService) Withdraw(tokenAddress common.TokenAddress, partnerAddress common.Address, totalWithdraw common.TokenAmount) (chan bool, error) {
	chainState := self.StateFromChannel()
	partAddr, _ := comm.AddressParseFromBytes(partnerAddress[:])
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress), tokenAddress, partnerAddress)
	if channelState == nil {
		log.Errorf("withdraw failed, can not find specific channel with %s", partAddr.ToBase58())
		return nil, errors.New("can not find specific channel")
	}

	_, exist := self.GetWithdrawStatus(channelState.Identifier)
	if exist {
		log.Errorf("withdraw failed, there is ongoing withdraw for the channel")
		return nil, errors.New("withdraw ongoing")
	}

	paymentNetworkId := common.PaymentNetworkID(self.microAddress)
	TokenNetworkId := transfer.GetTokenNetworkIdByTokenAddress(self.StateFromChannel(),
		paymentNetworkId, tokenAddress)

	withdraw := &transfer.ActionWithdraw{
		TokenNetworkId: TokenNetworkId,
		ChannelId:      channelState.Identifier,
		Participant:    self.address,
		Partner:        partnerAddress,
		TotalWithdraw:  totalWithdraw,
	}

	withdrawResult := self.RegisterWithdrawStatus(channelState.Identifier)

	self.HandleStateChange(withdraw)

	return withdrawResult, nil
}

func (self *ChannelService) RegisterWithdrawStatus(channelId common.ChannelID) chan bool {
	withdrawResult := make(chan bool, 1)
	self.channelWithdrawStatus.Store(channelId, withdrawResult)
	return withdrawResult
}

func (self *ChannelService) GetWithdrawStatus(channelId common.ChannelID) (chan bool, bool) {
	result, exist := self.channelWithdrawStatus.Load(channelId)
	if exist {
		return result.(chan bool), true
	}

	return nil, false
}

func (self *ChannelService) RemoveWithdrawStatus(channelId common.ChannelID) bool {
	_, exist := self.channelWithdrawStatus.Load(channelId)
	if exist {
		self.channelWithdrawStatus.Delete(channelId)
		return true
	}

	return false
}

func (self *ChannelService) WithdrawResultNotify(channelId common.ChannelID, result bool) bool {
	withdrawResult, exist := self.GetWithdrawStatus(channelId)
	if !exist {
		return false
	}

	self.RemoveWithdrawStatus(channelId)

	log.Infof("WithdrawResultNotify for channel %d, result %v", channelId, result)
	withdrawResult <- result
	return true
}

type NewChannelNotification struct {
	ChannelId      common.ChannelID
	PartnerAddress common.Address
}

func (self *ChannelService) GetNewChannelNotifier() chan *NewChannelNotification {
	return self.channelNewNotifier
}

func (self *ChannelService) NotifyNewChannel(channelId common.ChannelID, partnerAddress common.Address) {
	if self.isRestoreFinish {
		notification := &NewChannelNotification{
			ChannelId:      channelId,
			PartnerAddress: partnerAddress,
		}

		log.Debugf("NotifiNewchannel: channel id : %v, partner address : %v", channelId, partnerAddress)
		self.channelNewNotifier <- notification
	}
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
	return string(path[0 : i+1]), nil
}

func (self *ChannelService) GetTotalDepositBalance(partnerAddress common.Address) (common.TokenAmount, error) {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), partnerAddress)
	if channelState == nil {
		return 0, errors.New("GetTotalDepositBalance error, channel state is nil")
	}
	return channelState.OurState.GetContractBalance(), nil
}

func (self *ChannelService) GetTotalWithdraw(partnerAddress common.Address) (common.TokenAmount, error) {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), partnerAddress)
	if channelState == nil {
		return 0, errors.New("GetTotalWithdraw error, channel state is nil")
	}
	return channelState.OurState.GetTotalWithdraw(), nil
}

func (self *ChannelService) GetCurrentBalance(partnerAddress common.Address) (common.TokenAmount, error) {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), partnerAddress)
	if channelState == nil {
		return 0, errors.New("GetCurrentBalance error, channel state is nil")
	}
	return channelState.OurState.GetGasBalance(), nil
}

func (self *ChannelService) GetAvailableBalance(partnerAddress common.Address) (common.TokenAmount, error) {
	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), partnerAddress)
	if channelState == nil {
		return 0, errors.New("GetAvailableBalance error, channel state is nil")
	}
	return transfer.GetDistributable(channelState.OurState, channelState.PartnerState), nil
}

func (self *ChannelService) GetAllMessageQueues() transfer.QueueIdsToQueuesType {
	return transfer.GetAllMessageQueues(self.StateFromChannel())
}

type ChannelInfo struct {
	ChannelId common.ChannelID
	Balance   common.TokenAmount
	Address   common.Address
	TokenAddr common.TokenAddress
}

func (self *ChannelService) GetAllChannelInfo() []*ChannelInfo {
	infos := make([]*ChannelInfo, 0)

	channelList := self.GetChannelList(common.PaymentNetworkID(self.microAddress),
		common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), common.EmptyAddress)

	for e := channelList.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*transfer.NettingChannelState)

		partnerAddress := channelState.PartnerState.Address
		balance := transfer.GetDistributable(channelState.OurState, channelState.PartnerState)

		info := &ChannelInfo{
			ChannelId: channelState.Identifier,
			Address:   partnerAddress,
			Balance:   balance,
			TokenAddr: common.TokenAddress(channelState.TokenAddress),
		}
		infos = append(infos, info)
	}

	return infos
}

type PaymentResult struct {
	Identifier common.PaymentID
	Target     common.Address
	Result     bool
	Reason     string
}

func (self *ChannelService) GetPaymentResult(target common.Address, identifier common.PaymentID) *PaymentResult {
	eventRecord := storage.GetEventWithTargetAndPaymentId(self.Wal.Storage, target, identifier)
	if eventRecord != nil && eventRecord.Data != nil {
		switch eventRecord.Data.(type) {
		case *transfer.EventPaymentSentSuccess:
			event := eventRecord.Data.(*transfer.EventPaymentSentSuccess)
			result := &PaymentResult{
				Identifier: event.Identifier,
				Target:     event.Target,
				Result:     true,
			}
			return result
		case *transfer.EventPaymentSentFailed:
			event := eventRecord.Data.(*transfer.EventPaymentSentFailed)
			result := &PaymentResult{
				Identifier: event.Identifier,
				Target:     event.Target,
				Result:     false,
				Reason:     event.Reason,
			}
			return result
		default:
			log.Errorf("invalid type for payment result : %s", reflect.TypeOf(eventRecord.Data).String())
			return nil
		}
	}

	log.Debugf("Payment result not found for paymentId %d, target %s", identifier, common.ToBase58(target))
	return nil
}
