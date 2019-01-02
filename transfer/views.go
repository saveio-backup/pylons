package transfer

import (
	"container/list"
	"sort"

	"github.com/oniio/oniChannel/typing"
)

func AllNeighbourNodes(chainState *ChainState) map[typing.Address]int {
	addresses := make(map[typing.Address]int)

	for _, p := range chainState.IdentifiersToPaymentnetworks {
		for _, t := range p.TokenIdentifiersToTokenNetworks {
			for _, c := range t.ChannelIdentifiersToChannels {
				addresses[c.PartnerState.Address] = 0
			}
		}
	}

	return addresses
}

func GetBlockNumber(chainState *ChainState) typing.BlockNumber {
	return chainState.BlockNumber
}

func CountTokenNetworkChannels(chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) int {

	var count int

	tokenNetwork := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	if tokenNetwork != nil {
		//[NOTE] use number of channels in ChannelIdentifiersToChannels currently
		count = len(tokenNetwork.ChannelIdentifiersToChannels)
	} else {
		count = 0
	}

	return count
}

func GetPendingTransactions(chainState *ChainState) []Event {
	return chainState.PendingTransactions
}

func GetAllMessageQueues(chainState *ChainState) *QueueIdsToQueuesType {
	return &chainState.QueueIdsToQueues
}

func GetNetworkStatuses(chainState *ChainState) *map[typing.Address]string {
	return &chainState.NodeAddressesToNetworkstates
}

func GetNodeNetworkStatus(chainState *ChainState, nodeAddress typing.Address) string {
	result, exist := chainState.NodeAddressesToNetworkstates[nodeAddress]

	if !exist {
		result = NodeNetworkUnknown
	}

	return result
}

func GetParticipantsAddresses(chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) map[typing.Address]int {

	addresses := make(map[typing.Address]int)

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	if tokenNetworkState != nil {
		//[TODO] use token_network.network_graph.network.nodes() when supporting route
		//[NOTE] this function return network.nodes whose are relative to route, NOT include
		//channels current nimbus node take part in. So just return empty!!
		//[NOTE]  use channels in ChannelIdentifiersToChannels may break behavior of this function
	}

	return addresses
}

func GetOurCapacityForTokenNetwork(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) int {

	openChannels := GetChannelStateOpen(
		chainState,
		paymentNetworkId,
		tokenAddress)

	var totalDeposit typing.TokenAmount

	for e := openChannels.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*NettingChannelState)
		totalDeposit += channelState.OurState.ContractBalance
	}

	return int(totalDeposit)
}

func GetPaymentNetworkIdentifiers(
	chainState ChainState) *list.List {

	result := list.New()
	for k := range chainState.IdentifiersToPaymentnetworks {
		result.PushBack(k)
	}

	return result
}

func GetTokenNetworkRegistryByTokenNetworkIdentifier(
	chainState *ChainState,
	tokenNetworkIdentifier typing.TokenNetworkID) *PaymentNetworkState {

	for _, v := range chainState.IdentifiersToPaymentnetworks {
		_, exist := v.TokenIdentifiersToTokenNetworks[tokenNetworkIdentifier]
		if exist {
			return v
		}
	}

	return nil
}

func GetTokenNetworkIdentifierByTokenAddress(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) typing.TokenNetworkID {

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	return tokenNetworkState.Address
}

func GetTokenNetworkIdentifiers(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentnetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for _, v := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(v.Address)
		}
	}

	return result
}

func GetTokenIdentifiers(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentnetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for k := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(k)
		}
	}

	return result
}

func GetTokenNetworkAddressesFor(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentnetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for _, v := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(v.TokenAddress)
		}
	}

	return result
}

func TotalTokenNetworkChannels(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) int {

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	result := 0
	if tokenNetworkState != nil {
		result = len(tokenNetworkState.ChannelIdentifiersToChannels)
	}

	return result
}

func GetTokenNetworkByTokenAddress(chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *TokenNetworkState {

	//Hack! Since we only have one TokenNetworkState!!
	paymentNetwork := chainState.IdentifiersToPaymentnetworks[typing.PaymentNetworkID{}]
	if paymentNetwork != nil {
		return paymentNetwork.tokenAddressesToTokenNetworks[typing.TokenAddress{}]
	}

	return nil
}

func GetTokenNetworkByIdentifier(
	chainState *ChainState,
	tokenNetworkId typing.TokenNetworkID) *TokenNetworkState {

	var tokenNetworkState *TokenNetworkState

	paymentNetworkState := chainState.IdentifiersToPaymentnetworks[typing.PaymentNetworkID{}]
	tokenNetworkState = paymentNetworkState.TokenIdentifiersToTokenNetworks[typing.TokenNetworkID{}]

	return tokenNetworkState
}

func GetChannelStateFor(chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID, tokenAddress typing.TokenAddress,
	partnerAddress typing.Address) *NettingChannelState {

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	var channelState *NettingChannelState

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	if tokenNetworkState != nil {
		states := FilterChannelsByStatus(
			tokenNetworkState.partnerAddressesToChannels[partnerAddress],
			excludeStates)

		if states != nil && states.Len() != 0 {
			channelState = states.Back().Value.(*NettingChannelState)
		}
	}

	return channelState
}

func GetChannelStateByTokenNetworkAndPartner(
	chainState *ChainState,
	tokenNetworkId typing.TokenNetworkID,
	partnerAddress typing.Address) *NettingChannelState {

	tokenNetworkState := GetTokenNetworkByIdentifier(
		chainState,
		tokenNetworkId)

	var channelState *NettingChannelState

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	if tokenNetworkState != nil {
		states := FilterChannelsByStatus(
			tokenNetworkState.partnerAddressesToChannels[partnerAddress],
			excludeStates)

		if states != nil {
			channelState = states.Back().Value.(*NettingChannelState)
		}
	}

	return channelState
}

func GetChannelStateByTokenNetworkIdentifier(
	chainState *ChainState,
	tokenNetworkId typing.TokenNetworkID,
	channelId typing.ChannelID) *NettingChannelState {

	var channelState *NettingChannelState

	tokenNetworkState := GetTokenNetworkByIdentifier(
		chainState,
		tokenNetworkId)

	if tokenNetworkState != nil {
		channelState = tokenNetworkState.ChannelIdentifiersToChannels[channelId]
	}

	return channelState
}

func GetChannelStateById(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress,
	channelId typing.ChannelID) *NettingChannelState {

	var channelState *NettingChannelState

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	if tokenNetworkState != nil {
		channelState = tokenNetworkState.ChannelIdentifiersToChannels[channelId]

	}

	return channelState
}

func getChannelStateFilter(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress,
	filterFn func(x *NettingChannelState) bool) *list.List {

	tokenNetworkState := getTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	result := list.New()

	for _, v := range tokenNetworkState.ChannelIdentifiersToChannels {
		if filterFn(v) {
			result.PushBack(v)
		}
	}

	return result
}

func GetChannelStateForReceiving(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	tokenNetworkState := getTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	result := list.New()

	for _, v := range tokenNetworkState.ChannelIdentifiersToChannels {
		if v.PartnerState.BalanceProof != nil {
			result.PushBack(v)
		}
	}

	return result
}

func GetChannelStateOpen(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateOpened {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(
		chainState,
		paymentNetworkId,
		tokenAddress,
		f)
}

func GetChannelStateClosing(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateClosing {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(
		chainState,
		paymentNetworkId,
		tokenAddress,
		f)
}

func GetChannelStateClosed(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateClosed {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(
		chainState,
		paymentNetworkId,
		tokenAddress,
		f)
}

func GetChannelStateSettling(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateSettling {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(
		chainState,
		paymentNetworkId,
		tokenAddress,
		f)
}

func GetChannelStateSettled(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateSettled {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(
		chainState,
		paymentNetworkId,
		tokenAddress,
		f)
}

func GetTransferRole(
	chainState *ChainState,
	secrethash typing.SecretHash) string {

	var result string

	transferTask, _ := chainState.PaymentMapping.SecrethashesToTask[secrethash]

	switch transferTask.(type) {
	case *InitiatorTask:
		result = "initiator"
	case *MediatorTask:
		result = "mediator"
	case *TargetTask:
		result = "target"

	}

	return result
}

func ListChannelStateForTokenNetwork(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	var result *list.List

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	if tokenNetworkState != nil {
		result = flattenChannelStates(tokenNetworkState.partnerAddressesToChannels)
	}

	return result
}

func ListAllChannelState(chainState *ChainState) *list.List {

	var paymentNetworkState *PaymentNetworkState
	var result *list.List

	for _, v := range chainState.IdentifiersToPaymentnetworks {
		paymentNetworkState = v
		for _, v2 := range paymentNetworkState.tokenAddressesToTokenNetworks {
			tokenNetworkState := v2

			result = flattenChannelStates(tokenNetworkState.partnerAddressesToChannels)
		}
	}

	return result
}

func searchPaymentNetworkByTokenNetworkId(
	chainState *ChainState,
	tokenNetworkId typing.TokenNetworkID) *PaymentNetworkState {

	var paymentNetworkState *PaymentNetworkState

	for _, v := range chainState.IdentifiersToPaymentnetworks {
		paymentNetworkState = v
		_, exist := paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkId]
		if exist {
			return paymentNetworkState
		}
	}

	return paymentNetworkState
}

func FilterChannelsByPartnerAddress(
	chainState *ChainState,
	paymentNetworkId typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress,
	partnerAddresses *list.List) *list.List {

	result := list.New()

	tokenNetworkState := GetTokenNetworkByTokenAddress(
		chainState,
		paymentNetworkId,
		tokenAddress)

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	for e := partnerAddresses.Front(); e != nil; e = e.Next() {
		partner := e.Value.(typing.Address)
		states := FilterChannelsByStatus(
			tokenNetworkState.partnerAddressesToChannels[partner],
			excludeStates)

		if states != nil {
			result.PushBack(states.Back().Value.(*NettingChannelState))
		}
	}

	return result
}

func FilterChannelsByStatus(channelStates map[typing.ChannelID]*NettingChannelState,
	excludeStates map[string]int) *list.List {

	if excludeStates == nil {
		excludeStates = map[string]int{}
	}

	states := list.New()
	if channelStates == nil {
		return states
	}

	for _, v := range channelStates {
		result := GetStatus(v)
		_, exist := excludeStates[result]
		if !exist {
			states.PushBack(v)
		}
	}

	return states
}

type NettingChannelStateSlice []*NettingChannelState

func (h NettingChannelStateSlice) Len() int {
	return len(h)
}

func (h NettingChannelStateSlice) Less(i, j int) bool {
	return h[i].Identifier < h[j].Identifier
}

func (h NettingChannelStateSlice) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func flattenChannelStates(channelStates map[typing.Address]map[typing.ChannelID]*NettingChannelState) *list.List {
	var states NettingChannelStateSlice

	result := list.New()

	for _, v := range channelStates {
		idToChannel := v
		for _, v2 := range idToChannel {
			states = append(states, v2)
		}
	}

	sort.Sort(states)

	for i := 0; i < states.Len(); i++ {
		result.PushBack(states[i])
	}

	return result
}
