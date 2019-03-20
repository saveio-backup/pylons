package transfer

import (
	"container/list"
	"sort"

	scUtils "github.com/oniio/oniChain/smartcontract/service/native/utils"
	"github.com/oniio/oniChannel/common"
)

func GetNeighbours(chainState *ChainState) []common.Address {
	var addr []common.Address
	for _, p := range chainState.IdentifiersToPaymentNetworks {
		for _, t := range p.TokenIdentifiersToTokenNetworks {
			for _, c := range t.ChannelIdentifiersToChannels {
				if !common.AddressEqual(c.PartnerState.Address, common.Address{}) {
					addr = append(addr, c.PartnerState.Address)
				}
			}
		}
	}
	return addr
}

func GetBlockHeight(chainState *ChainState) common.BlockHeight {
	return chainState.BlockHeight
}

func CountTokenNetworkChannels(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) int {

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

func GetNetworkStatuses(chainState *ChainState) *map[common.Address]string {
	return &chainState.NodeAddressesToNetworkStates
}

func GetNodeNetworkStatus(chainState *ChainState, nodeAddress common.Address) string {
	result, exist := chainState.NodeAddressesToNetworkStates[nodeAddress]
	if !exist {
		result = NetworkUnknown
	}

	return result
}

func GetParticipantsAddresses(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) map[common.Address]int {

	addresses := make(map[common.Address]int)

	tokenNetworkState := GetTokenNetworkByTokenAddress(chainState, paymentNetworkId,
		tokenAddress)

	if tokenNetworkState != nil {
		//[TODO] use token_network.network_graph.network.nodes() when supporting route
		//[NOTE] this function return network.nodes whose are relative to route, NOT include
		//channels current channel node take part in. So just return empty!!
		//[NOTE]  use channels in ChannelIdentifiersToChannels may break behavior of this function
	}

	return addresses
}

func GetOurCapacityForTokenNetwork(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) int {

	openChannels := GetChannelStateOpen(
		chainState,
		paymentNetworkId,
		tokenAddress)

	var totalDeposit common.TokenAmount

	for e := openChannels.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*NettingChannelState)
		totalDeposit += channelState.OurState.ContractBalance
	}

	return int(totalDeposit)
}

func GetPaymentNetworkIdentifiers(chainState *ChainState) []common.PaymentNetworkID {

	result := make([]common.PaymentNetworkID, 0)
	for k := range chainState.IdentifiersToPaymentNetworks {
		result = append(result, k)
	}

	return result
}

func GetTokenNetworkRegistryByTokenNetworkIdentifier(chainState *ChainState,
	tokenNetworkIdentifier common.TokenNetworkID) *PaymentNetworkState {
	for _, v := range chainState.IdentifiersToPaymentNetworks {
		_, exist := v.TokenIdentifiersToTokenNetworks[tokenNetworkIdentifier]
		if exist {
			return v
		}
	}
	return nil
}

func GetTokenNetworkIdentifierByTokenAddress(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) common.TokenNetworkID {
	tokenNetworkState := GetTokenNetworkByTokenAddress(chainState, paymentNetworkId, tokenAddress)

	return tokenNetworkState.Address
}

func GetTokenNetworkIdentifiers(chainState *ChainState,
	paymentNetworkId common.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for _, v := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(v.Address)
		}
	}

	return result
}

func GetTokenIdentifiers(chainState *ChainState,
	paymentNetworkId common.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for k := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(k)
		}
	}

	return result
}

func GetTokenNetworkAddressesFor(chainState *ChainState,
	paymentNetworkId common.PaymentNetworkID) *list.List {

	var result *list.List

	paymentNetworkState := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		result := list.New()
		for _, v := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			result.PushBack(v.TokenAddress)
		}
	}

	return result
}

func TotalTokenNetworkChannels(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) int {

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

func GetTokenNetworkByTokenAddress(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *TokenNetworkState {
	//log.Info("paymentNetworkId = ", paymentNetworkId)
	//log.Info("chainState.IdentifiersToPaymentNetworks = ", chainState.IdentifiersToPaymentNetworks)
	//Hack! Since we only have one TokenNetworkState!!
	paymentNetwork := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]
	if paymentNetwork != nil {
		if tokenNetworkId, ok := paymentNetwork.TokenAddressesToTokenIdentifiers[tokenAddress]; ok {
			return paymentNetwork.TokenIdentifiersToTokenNetworks[tokenNetworkId]
		}
	}
	return nil
}

func GetTokenNetworkByIdentifier(chainState *ChainState,
	tokenNetworkId common.TokenNetworkID) *TokenNetworkState {
	//log.Debug("chainState.IdentifiersToPaymentNetworks = %+v\n", chainState.IdentifiersToPaymentNetworks)
	paymentNetworkState := chainState.IdentifiersToPaymentNetworks[common.PaymentNetworkID(scUtils.MicroPayContractAddress)]
	//log.Debug("paymentNetworkState = %+v\n", paymentNetworkState)
	if paymentNetworkState == nil {
		return nil
	}
	return paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkId]
}

func GetChannelStateFor(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddress common.Address) *NettingChannelState {

	tokenNetworkState := GetTokenNetworkByTokenAddress(chainState, paymentNetworkId, tokenAddress)

	var channelState *NettingChannelState

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	if tokenNetworkState != nil {
		states := FilterChannelsByStatus(tokenNetworkState.PartnerAddressesToChannels[partnerAddress],
			excludeStates)

		if states != nil && states.Len() != 0 {
			channelState = states.Back().Value.(*NettingChannelState)
		}
	}

	return channelState
}

func GetChannelStateByTokenNetworkAndPartner(chainState *ChainState, tokenNetworkId common.TokenNetworkID,
	partnerAddress common.Address) *NettingChannelState {

	tokenNetworkState := GetTokenNetworkByIdentifier(chainState, tokenNetworkId)

	var channelState *NettingChannelState

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	if tokenNetworkState != nil {
		states := FilterChannelsByStatus(tokenNetworkState.PartnerAddressesToChannels[partnerAddress], excludeStates)
		if states != nil {
			channelState = states.Back().Value.(*NettingChannelState)
		}
	}

	return channelState
}

func GetChannelStateByTokenNetworkIdentifier(chainState *ChainState,
	tokenNetworkId common.TokenNetworkID, channelId common.ChannelID) *NettingChannelState {

	var channelState *NettingChannelState

	tokenNetworkState := GetTokenNetworkByIdentifier(
		chainState,
		tokenNetworkId)

	if tokenNetworkState != nil {
		channelState = tokenNetworkState.ChannelIdentifiersToChannels[channelId]
	}

	return channelState
}

func GetChannelStateById(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, channelId common.ChannelID) *NettingChannelState {

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

func getChannelStateFilter(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, filterFn func(x *NettingChannelState) bool) *list.List {

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

func GetChannelStateForReceiving(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	tokenNetworkState := getTokenNetworkByTokenAddress(chainState, paymentNetworkId,
		tokenAddress)

	result := list.New()

	for _, v := range tokenNetworkState.ChannelIdentifiersToChannels {
		if v.PartnerState.BalanceProof != nil {
			result.PushBack(v)
		}
	}

	return result
}

func GetChannelStateOpen(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateOpened {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(chainState, paymentNetworkId, tokenAddress, f)
}

func GetChannelStateClosing(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

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

func GetChannelStateClosed(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

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

func GetChannelStateSettling(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

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

func GetChannelStateSettled(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	f := func(channelState *NettingChannelState) bool {
		state := GetStatus(channelState)
		if state == ChannelStateSettled {
			return true
		} else {
			return false
		}
	}

	return getChannelStateFilter(chainState, paymentNetworkId, tokenAddress, f)
}

func GetTransferRole(chainState *ChainState, secrethash common.SecretHash) string {

	var result string

	transferTask, _ := chainState.PaymentMapping.SecretHashesToTask[secrethash]

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

func ListChannelStateForTokenNetwork(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *list.List {

	var result *list.List

	tokenNetworkState := GetTokenNetworkByTokenAddress(chainState, paymentNetworkId, tokenAddress)

	if tokenNetworkState != nil {
		result = flattenChannelStates(tokenNetworkState.PartnerAddressesToChannels)
	}

	return result
}

func ListAllChannelState(chainState *ChainState) *list.List {
	var paymentNetworkState *PaymentNetworkState
	var result *list.List

	for _, v := range chainState.IdentifiersToPaymentNetworks {
		paymentNetworkState = v
		for _, v2 := range paymentNetworkState.TokenIdentifiersToTokenNetworks {
			tokenNetworkState := v2

			result = flattenChannelStates(tokenNetworkState.PartnerAddressesToChannels)
		}
	}
	return result
}

func searchPaymentNetworkByTokenNetworkId(chainState *ChainState,
	tokenNetworkId common.TokenNetworkID) *PaymentNetworkState {

	var paymentNetworkState *PaymentNetworkState

	for _, v := range chainState.IdentifiersToPaymentNetworks {
		paymentNetworkState = v
		_, exist := paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkId]
		if exist {
			return paymentNetworkState
		}
	}

	return paymentNetworkState
}

func FilterChannelsByPartnerAddress(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress, partnerAddresses *list.List) *list.List {

	result := list.New()

	tokenNetworkState := GetTokenNetworkByTokenAddress(chainState, paymentNetworkId,
		tokenAddress)

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0

	for e := partnerAddresses.Front(); e != nil; e = e.Next() {
		partner := e.Value.(common.Address)
		states := FilterChannelsByStatus(
			tokenNetworkState.PartnerAddressesToChannels[partner],
			excludeStates)

		if states != nil {
			result.PushBack(states.Back().Value.(*NettingChannelState))
		}
	}

	return result
}

func FilterChannelsByStatus(channelStates map[common.ChannelID]*NettingChannelState,
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

func flattenChannelStates(channelStates map[common.Address]map[common.ChannelID]*NettingChannelState) *list.List {
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
