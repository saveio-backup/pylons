package channelservice

import (
	"container/list"
	"sync"

	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

type ConnectionManager struct {
	funds                  typing.TokenAmount
	initialChannelTarget   int
	joinableFundsTarget    float32
	nimbus                 *ChannelService
	registryAddress        typing.Address
	tokenNetworkIdentifier typing.TokenNetworkID
	tokenAddress           typing.TokenAddress

	lock sync.Mutex
	api  *NimbusAPI

	wg sync.WaitGroup
}

func NewConnectionManager(
	nimbus *ChannelService,
	tokenNetworkIdentifier typing.TokenNetworkID) *ConnectionManager {
	self := new(ConnectionManager)

	self.funds = 0
	self.initialChannelTarget = 0
	self.joinableFundsTarget = 0
	self.nimbus = nimbus

	chainState := nimbus.StateFromNimbus()
	tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, tokenNetworkIdentifier)
	tokenNetworkRegistry := transfer.GetTokenNetworkRegistryByTokenNetworkIdentifier(
		chainState, tokenNetworkIdentifier)

	self.registryAddress = tokenNetworkRegistry.GetAddress()
	self.tokenNetworkIdentifier = tokenNetworkIdentifier
	self.tokenAddress = tokenNetworkState.GetTokenAddress()

	self.api = NewNimbusAPI(nimbus)

	return self
}

func getBootstrapAddress() typing.Address {
	bootstrapAddr := new(typing.Address)

	for i := 0; i < 20; i++ {
		bootstrapAddr[i] = 0x22
	}

	return *bootstrapAddr
}

func (self *ConnectionManager) connect(funds typing.TokenAmount,
	initialChannelTarget int,
	joinableFundsTarget float32) {

	//[TODO] check there is enough token for funds
	//var tokenBalance typing.TokenAmount

	self.lock.Lock()
	defer self.lock.Unlock()

	self.funds = funds
	self.initialChannelTarget = initialChannelTarget
	self.joinableFundsTarget = joinableFundsTarget

	qtyNetworkChannels := transfer.CountTokenNetworkChannels(
		self.nimbus.StateFromNimbus(),
		typing.PaymentNetworkID(self.registryAddress),
		self.tokenAddress)

	if qtyNetworkChannels == 0 {
		bootstrapAddr := getBootstrapAddress()

		self.api.ChannelOpen(typing.PaymentNetworkID(self.registryAddress), self.tokenAddress, bootstrapAddr, 0, 0.5)
	} else {
		self.openChannels()
	}

	return
}

func (self *ConnectionManager) Leave(registryAddress typing.PaymentNetworkID) *list.List {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.initialChannelTarget = 0

	return nil
}

func (self *ConnectionManager) JoinChannel(partnerAddress typing.Address,
	partnerDeposit typing.TokenAmount) {
	self.lock.Lock()
	defer self.lock.Unlock()

	var joiningFunds typing.TokenAmount

	//Currently, only router will join channel! Just use same deposit with partner.
	//Can add policy function later
	joiningFunds = partnerDeposit
	self.api.SetTotalChannelDeposit(typing.PaymentNetworkID(self.registryAddress), self.tokenAddress,
		partnerAddress, joiningFunds, 0.5)

	return
}

func (self *ConnectionManager) RetryConnect() {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.fundsRemaining() > 0 && self.leavingState() {
		self.openChannels()
	}

	return
}

func (self *ConnectionManager) findNewPartners() *list.List {

	openedChannels := transfer.GetChannelStateOpen(self.nimbus.StateFromNimbus(),
		typing.PaymentNetworkID(self.registryAddress), self.tokenAddress)

	known := list.New()
	for e := openedChannels.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(1)
		known.PushBack(channelEndState.GetAddress())
	}

	known.PushBack(getBootstrapAddress())
	known.PushBack(self.nimbus.address)

	participantsAddresses := transfer.GetParticipantsAddresses(
		self.nimbus.StateFromNimbus(),
		typing.PaymentNetworkID{}, typing.TokenAddress{})

	available := list.New()
	for k := range participantsAddresses {
		found := false
		for e := known.Front(); e != nil; {
			knownAddress := e.Value.(typing.Address)
			if typing.AddressEqual(k, knownAddress) {
				found = true
				break
			}
		}

		if found == false {
			available.PushBack(k)
		}
	}

	//[TODO] implement shuffle algorithm
	newPartners := available

	return newPartners
}

func (self *ConnectionManager) JoinPartner(partner typing.Address) {

	self.api.ChannelOpen(typing.PaymentNetworkID(self.registryAddress), self.tokenAddress, partner, 0, 0.5)

	self.api.SetTotalChannelDeposit(typing.PaymentNetworkID(self.registryAddress), self.tokenAddress,
		partner, self.initialFundingPerPartner(), 0.5)

	self.wg.Done()
	return
}

func (self *ConnectionManager) openChannels() bool {

	openChannels := transfer.GetChannelStateOpen(self.nimbus.StateFromNimbus(),
		typing.PaymentNetworkID{}, typing.TokenAddress{})

	bootstrapAddress := getBootstrapAddress()
	for e := openChannels.Front(); e != nil; {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(1)
		if typing.AddressEqual(channelEndState.GetAddress(), bootstrapAddress) {
			e = e.Next()
			openChannels.Remove(e)
		} else {
			e = e.Next()
		}
	}

	fundedChannels := list.New()
	nonfundedChannels := list.New()
	for e := openChannels.Front(); e != nil; {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(0)
		if channelEndState.GetContractBalance() >= self.initialFundingPerPartner() {
			fundedChannels.PushBack(channelState)
		} else {
			nonfundedChannels.PushBack(channelState)
		}
	}

	possibleNewPartners := self.findNewPartners()
	if possibleNewPartners == nil || possibleNewPartners.Len() == 0 {
		return false
	}

	if fundedChannels.Len() > self.initialChannelTarget {
		return false
	}

	if nonfundedChannels == nil && (possibleNewPartners == nil || possibleNewPartners.Len() == 0) {
		return false
	}

	nToJoin := self.initialChannelTarget - fundedChannels.Len()
	nonfundedPartners := list.New()
	for e := nonfundedChannels.Front(); e != nil; {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(1)
		partnerAddress := channelEndState.GetAddress()
		nonfundedPartners.PushBack(partnerAddress)

	}

	joinPartners := list.New()
	numJoined := 0

	for e := nonfundedPartners.Front(); e != nil && numJoined < nToJoin; {
		partnerAddress := e.Value.(typing.Address)
		joinPartners.PushBack(partnerAddress)
		numJoined++
	}

	for e := possibleNewPartners.Front(); e != nil && numJoined < nToJoin; {
		partnerAddress := e.Value.(typing.Address)
		joinPartners.PushBack(partnerAddress)
		numJoined++
	}

	for e := joinPartners.Front(); e != nil; {
		partnerAddress := e.Value.(typing.Address)
		self.wg.Add(1)
		go self.JoinPartner(partnerAddress)
	}

	self.wg.Wait()

	return true
}

func (self *ConnectionManager) initialFundingPerPartner() typing.TokenAmount {
	return 0
}

func (self *ConnectionManager) fundsRemaining() int {
	return 0
}

func (self *ConnectionManager) leavingState() bool {
	return self.initialChannelTarget < 1
}
