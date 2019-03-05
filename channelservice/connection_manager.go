package channelservice

import (
	"container/list"
	"sync"

	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

type ConnectionManager struct {
	funds                  common.TokenAmount
	initialChannelTarget   int
	joinableFundsTarget    float32
	channel                *ChannelService
	registryAddress        common.Address
	tokenNetworkIdentifier common.TokenNetworkID
	tokenAddress           common.TokenAddress

	lock sync.Mutex

	wg sync.WaitGroup
}

func NewConnectionManager(channel *ChannelService, tokenNetworkIdentifier common.TokenNetworkID) *ConnectionManager {
	self := new(ConnectionManager)

	self.funds = 0
	self.initialChannelTarget = 0
	self.joinableFundsTarget = 0
	self.channel = channel

	chainState := channel.StateFromChannel()
	tokenNetworkState := transfer.GetTokenNetworkByIdentifier(chainState, tokenNetworkIdentifier)
	tokenNetworkRegistry := transfer.GetTokenNetworkRegistryByTokenNetworkIdentifier(
		chainState, tokenNetworkIdentifier)

	self.registryAddress = tokenNetworkRegistry.GetAddress()
	self.tokenNetworkIdentifier = tokenNetworkIdentifier
	self.tokenAddress = tokenNetworkState.GetTokenAddress()

	return self
}

func getBootstrapAddress() common.Address {
	bootstrapAddr := new(common.Address)

	for i := 0; i < 20; i++ {
		bootstrapAddr[i] = 0x22
	}

	return *bootstrapAddr
}

func (self *ConnectionManager) connect(funds common.TokenAmount,
	initialChannelTarget int,
	joinableFundsTarget float32) {

	//[TODO] check there is enough token for funds
	//var tokenBalance common.TokenAmount

	self.lock.Lock()
	defer self.lock.Unlock()

	self.funds = funds
	self.initialChannelTarget = initialChannelTarget
	self.joinableFundsTarget = joinableFundsTarget

	qtyNetworkChannels := transfer.CountTokenNetworkChannels(
		self.channel.StateFromChannel(),
		common.PaymentNetworkID(self.registryAddress),
		self.tokenAddress)

	if qtyNetworkChannels == 0 {
		bootstrapAddr := getBootstrapAddress()

		self.channel.OpenChannel(self.tokenAddress, bootstrapAddr)
	} else {
		self.openChannels()
	}

	return
}

func (self *ConnectionManager) Leave(registryAddress common.PaymentNetworkID) *list.List {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.initialChannelTarget = 0

	return nil
}

func (self *ConnectionManager) JoinChannel(partnerAddress common.Address,
	partnerDeposit common.TokenAmount) {
	self.lock.Lock()
	defer self.lock.Unlock()

	var joiningFunds common.TokenAmount

	//Currently, only router will join channel! Just use same deposit with partner.
	//Can add policy function later
	joiningFunds = partnerDeposit
	self.channel.SetTotalChannelDeposit(self.tokenAddress, partnerAddress, joiningFunds)

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

	openedChannels := transfer.GetChannelStateOpen(self.channel.StateFromChannel(),
		common.PaymentNetworkID(self.registryAddress), self.tokenAddress)

	known := list.New()
	for e := openedChannels.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(1)
		known.PushBack(channelEndState.GetAddress())
	}

	known.PushBack(getBootstrapAddress())
	known.PushBack(self.channel.address)

	participantsAddresses := transfer.GetParticipantsAddresses(
		self.channel.StateFromChannel(),
		common.PaymentNetworkID{}, common.TokenAddress{})

	available := list.New()
	for k := range participantsAddresses {
		found := false
		for e := known.Front(); e != nil; {
			knownAddress := e.Value.(common.Address)
			if common.AddressEqual(k, knownAddress) {
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

func (self *ConnectionManager) JoinPartner(partner common.Address) {

	self.channel.OpenChannel(self.tokenAddress, partner)

	self.channel.SetTotalChannelDeposit(self.tokenAddress, partner, self.initialFundingPerPartner())

	self.wg.Done()
	return
}

func (self *ConnectionManager) openChannels() bool {

	openChannels := transfer.GetChannelStateOpen(self.channel.StateFromChannel(),
		common.PaymentNetworkID{}, common.TokenAddress{})

	bootstrapAddress := getBootstrapAddress()
	for e := openChannels.Front(); e != nil; {
		channelState := e.Value.(*transfer.NettingChannelState)
		channelEndState := channelState.GetChannelEndState(1)
		if common.AddressEqual(channelEndState.GetAddress(), bootstrapAddress) {
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
		partnerAddress := e.Value.(common.Address)
		joinPartners.PushBack(partnerAddress)
		numJoined++
	}

	for e := possibleNewPartners.Front(); e != nil && numJoined < nToJoin; {
		partnerAddress := e.Value.(common.Address)
		joinPartners.PushBack(partnerAddress)
		numJoined++
	}

	for e := joinPartners.Front(); e != nil; {
		partnerAddress := e.Value.(common.Address)
		self.wg.Add(1)
		go self.JoinPartner(partnerAddress)
	}

	self.wg.Wait()

	return true
}

func (self *ConnectionManager) initialFundingPerPartner() common.TokenAmount {
	return 0
}

func (self *ConnectionManager) fundsRemaining() int {
	return 0
}

func (self *ConnectionManager) leavingState() bool {
	return self.initialChannelTarget < 1
}
