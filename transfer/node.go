package transfer

import (
	"container/list"
	"reflect"

	"github.com/oniio/oniChannel/typing"
)

func getNetworks(chainState *ChainState,
	paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) (*PaymentNetworkState, *TokenNetworkState) {

	var tokenNetworkState *TokenNetworkState

	paymentNetworkState := chainState.IdentifiersToPaymentnetworks[paymentNetworkIdentifier]

	if paymentNetworkState != nil {
		tokenNetworkState = paymentNetworkState.tokenAddressesToTokenNetworks[tokenAddress]
	}

	return paymentNetworkState, tokenNetworkState
}

func getTokenNetworkByTokenAddress(
	chainState *ChainState,
	paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *TokenNetworkState {

	_, tokenNetworkState := getNetworks(
		chainState,
		paymentNetworkIdentifier,
		tokenAddress,
	)

	return tokenNetworkState
}

func subdispatchToAllChannels(
	chainState *ChainState,
	stateChange StateChange,
	blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	for _, v := range chainState.IdentifiersToPaymentnetworks {
		for _, v2 := range v.tokenAddressesToTokenNetworks {
			for _, v3 := range v2.ChannelIdentifiersToChannels {
				result := StateTransitionForChannel(v3, stateChange, blockNumber)
				events.PushBackList(result.Events)
			}
		}
	}

	return TransitionResult{chainState, events}
}

func subdispatchToAllLockedTransfers(
	chainState *ChainState,
	stateChange StateChange) TransitionResult {
	events := list.New()

	for k := range chainState.PaymentMapping.SecrethashesToTask {
		result := subdispatchToPaymenttask(chainState, stateChange, k)
		events.PushBackList(result.Events)
	}

	return TransitionResult{chainState, events}
}

//[TODO] handle InitiatorTask, MediatorTask, TargetTask when supporting route
func subdispatchToPaymenttask(
	chainState *ChainState,
	stateChange StateChange,
	secrethash typing.SecretHash) TransitionResult {

	events := list.New()

	return TransitionResult{chainState, events}
}

//[TODO] handle InitiatorTask when supporting route
func subdispatchInitiatortask(
	chainState *ChainState,
	stateChange StateChange,
	tokenNetworkIdentifier typing.TokenNetworkID,
	secrethash typing.SecretHash) TransitionResult {

	events := list.New()

	return TransitionResult{chainState, events}
}

////[TODO] handle MediatorTask when supporting route
func subdispatchMediatortask(
	chainState *ChainState,
	stateChange StateChange,
	tokenNetworkIdentifier typing.TokenNetworkID,
	secrethash typing.SecretHash) TransitionResult {

	events := list.New()

	return TransitionResult{chainState, events}
}

//[TODO] handle TargetTask when supporting route
func subdispatchTargettask(
	chainState *ChainState,
	stateChange StateChange,
	tokenNetworkIdentifier typing.TokenNetworkID,
	channelIdentifier typing.ChannelID,
	secrethash typing.SecretHash) TransitionResult {

	events := list.New()

	return TransitionResult{chainState, events}
}

func maybeAddTokennetwork(
	chainState *ChainState,
	paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState) {

	tokenNetworkIdentifier := tokenNetworkState.Address
	tokenAddress := tokenNetworkState.TokenAddress

	paymentNetworkState, tokenNetworkStatePrevious := getNetworks(
		chainState,
		paymentNetworkIdentifier,
		tokenAddress)

	if paymentNetworkState == nil {
		paymentNetworkState = NewPaymentNetworkState()
		paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkState.Address] = tokenNetworkState
		paymentNetworkState.tokenAddressesToTokenNetworks[tokenNetworkState.TokenAddress] = tokenNetworkState

		chainState.IdentifiersToPaymentnetworks[paymentNetworkIdentifier] = paymentNetworkState
	}

	if tokenNetworkStatePrevious == nil {
		paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkIdentifier] = tokenNetworkState
		paymentNetworkState.tokenAddressesToTokenNetworks[tokenAddress] = tokenNetworkState
	}

	return
}

func inplaceDeleteMessageQueue(
	chainState *ChainState,
	stateChange StateChange,
	queueid QueueIdentifier) {

	queue, ok := chainState.QueueIdsToQueues[queueid]
	if ok == false {
		return
	}

	newQueue := inplaceDeleteMessage(queue, stateChange)

	if len(newQueue) == 0 {
		delete(chainState.QueueIdsToQueues, queueid)
	} else {
		chainState.QueueIdsToQueues[queueid] = newQueue
	}

	return
}

func inplaceDeleteMessage(
	messageQueue []Event,
	stateChange StateChange) []Event {

	if messageQueue == nil {
		return messageQueue
	}

	len := len(messageQueue)
	for i := 0; i < len; {
		message := GetSenderMessageEvent(messageQueue[i])
		sender, messageId := GetSenderAndMessageIdentifier(stateChange)
		if message.MessageIdentifier == messageId && typing.AddressEqual(typing.Address(message.Recipient), sender) {
			messageQueue = append(messageQueue[:i], messageQueue[i+1:]...)
			len--
		} else {
			i++
		}
	}

	return messageQueue
}

func handleBlockForNode(
	chainState *ChainState,
	stateChange *Block) TransitionResult {

	events := list.New()

	blockNumber := stateChange.BlockHeight
	chainState.BlockHeight = blockNumber

	channelsResult := subdispatchToAllChannels(
		chainState,
		stateChange,
		blockNumber)

	transfersResult := subdispatchToAllLockedTransfers(
		chainState,
		stateChange)

	events.PushBackList(channelsResult.Events)
	events.PushBackList(transfersResult.Events)

	return TransitionResult{chainState, events}
}

func handleChainInit(
	chainState *ChainState,
	stateChange *ActionInitChain) TransitionResult {

	result := NewChainState()
	result.BlockHeight = stateChange.BlockHeight
	result.Address = stateChange.OurAddress
	result.ChainId = stateChange.ChainId

	events := list.New()
	return TransitionResult{result, events}
}

func handleTokenNetworkAction(
	chainState *ChainState,
	stateChange StateChange) TransitionResult {

	events := list.New()

	tokenNetworkState := GetTokenNetworkByIdentifier(chainState, typing.TokenNetworkID{})
	paymentNetworkState := GetTokenNetworkRegistryByTokenNetworkIdentifier(
		chainState, typing.TokenNetworkID{})

	paymentNetworkId := paymentNetworkState.Address

	if tokenNetworkState != nil {
		iteration := stateTransitionForNetwork(paymentNetworkId, tokenNetworkState,
			stateChange, chainState.BlockHeight)

		if reflect.ValueOf(iteration.NewState).IsNil() {

			paymentNetworkState = searchPaymentNetworkByTokenNetworkId(
				chainState, typing.TokenNetworkID{})

			delete(paymentNetworkState.tokenAddressesToTokenNetworks, typing.TokenAddress{})
			delete(paymentNetworkState.TokenIdentifiersToTokenNetworks, typing.TokenNetworkID{})
		}

		events = iteration.Events
	}

	return TransitionResult{chainState, events}
}

func handleContractReceiveChannelClosed(
	chainState *ChainState,
	stateChange *ContractReceiveChannelClosed) TransitionResult {

	channelId := GetChannelIdentifier(stateChange)
	channelState := GetChannelStateByTokenNetworkIdentifier(
		chainState, typing.TokenNetworkID{}, channelId)

	if channelState != nil {
		queueId := QueueIdentifier{channelState.PartnerState.Address, channelId}

		if _, ok := chainState.QueueIdsToQueues[queueId]; ok {
			delete(chainState.QueueIdsToQueues, queueId)
		}
	}

	return handleTokenNetworkAction(chainState, stateChange)
}

func handleDelivered(
	chainState *ChainState,
	stateChange *ReceiveDelivered) TransitionResult {

	queueid := QueueIdentifier{stateChange.Sender, 0}
	inplaceDeleteMessageQueue(chainState, stateChange, queueid)

	return TransitionResult{chainState, list.New()}
}

func handleNewTokenNetwork(
	chainState *ChainState,
	stateChange *ActionNewTokenNetwork) TransitionResult {

	tokenNetworkState := stateChange.TokenNetwork
	paymentNetworkIdentifier := stateChange.PaymentNetworkIdentifier

	maybeAddTokennetwork(
		chainState,
		paymentNetworkIdentifier,
		tokenNetworkState)

	events := list.New()
	return TransitionResult{chainState, events}
}

func handleNodeChangeNetworkState(
	chainState *ChainState,
	stateChange *ActionChangeNodeNetworkState) TransitionResult {
	events := list.New()

	nodeAddress := stateChange.NodeAddress
	networkState := stateChange.NetworkState
	chainState.NodeAddressesToNetworkstates[nodeAddress] = networkState

	return TransitionResult{chainState, events}
}

func handleLeaveAllNetworks(chainState *ChainState) TransitionResult {
	events := list.New()

	for _, v := range chainState.IdentifiersToPaymentnetworks {
		for _, v := range v.tokenAddressesToTokenNetworks {
			result := getChannelsCloseEvents(chainState, v)
			events.PushBackList(result)
		}
	}

	return TransitionResult{chainState, events}
}

func handleNewPaymentNetwork(
	chainState *ChainState,
	stateChange *ContractReceiveNewPaymentNetwork) TransitionResult {
	events := list.New()

	paymentNetwork := stateChange.PaymentNetwork
	paymentNetworkIdentifier := paymentNetwork.Address

	_, ok := chainState.IdentifiersToPaymentnetworks[paymentNetworkIdentifier]
	if ok == false {
		chainState.IdentifiersToPaymentnetworks[paymentNetworkIdentifier] = paymentNetwork
	}

	return TransitionResult{chainState, events}
}

func handleTokenadded(
	chainState *ChainState,
	stateChange *ContractReceiveNewTokenNetwork) TransitionResult {

	events := list.New()
	maybeAddTokennetwork(
		chainState,
		stateChange.PaymentNetworkIdentifier,
		stateChange.TokenNetwork)

	return TransitionResult{chainState, events}
}

func handleProcessed(
	chainState *ChainState,
	stateChange *ReceiveProcessed) TransitionResult {

	events := list.New()

	for _, v := range chainState.QueueIdsToQueues {
		len := len(v)

		for i := 0; i < len; i++ {
			message := GetSenderMessageEvent(v[i])
			sender, messageId := GetSenderAndMessageIdentifier(stateChange)

			if message.MessageIdentifier == messageId && typing.AddressEqual(typing.Address(message.Recipient), sender) {
				if message, ok := v[i].(*SendDirectTransfer); ok {
					channelState := GetChannelStateByTokenNetworkAndPartner(chainState,
						message.BalanceProof.TokenNetworkIdentifier, typing.Address(message.Recipient))

					events.PushBack(&EventPaymentSentSuccess{channelState.PaymentNetworkIdentifier,
						channelState.TokenNetworkIdentifier, message.PaymentIdentifier,
						message.BalanceProof.TransferredAmount, message.Recipient})
				}
			}

		}
	}

	for k := range chainState.QueueIdsToQueues {
		inplaceDeleteMessageQueue(chainState, stateChange, k)
	}
	return TransitionResult{chainState, events}
}

func handleStateChangeForNode(chainStateArg State, stateChange StateChange) TransitionResult {

	chainState := chainStateArg.(*ChainState)
	events := list.New()
	iteration := TransitionResult{chainState, events}

	switch stateChange.(type) {
	case *Block:
		block, _ := stateChange.(*Block)
		iteration = handleBlockForNode(chainState, block)
	case *ActionInitChain:
		actionInitChain, _ := stateChange.(*ActionInitChain)
		iteration = handleChainInit(chainState, actionInitChain)
	case *ActionNewTokenNetwork:
		actionNewTokenNetwork, _ := stateChange.(*ActionNewTokenNetwork)
		iteration = handleNewTokenNetwork(chainState, actionNewTokenNetwork)
	case *ActionChannelClose:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ActionTransferDirect:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ContractReceiveChannelNew:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ContractReceiveChannelNewBalance:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ContractReceiveChannelSettled:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ContractReceiveUpdateTransfer:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ReceiveTransferDirect:
		iteration = handleTokenNetworkAction(chainState, stateChange)
	case *ActionChangeNodeNetworkState:
		actionChangeNodeNetworkState, _ := stateChange.(*ActionChangeNodeNetworkState)
		iteration = handleNodeChangeNetworkState(chainState, actionChangeNodeNetworkState)
	case *ActionLeaveAllNetworks:
		iteration = handleLeaveAllNetworks(chainState)
	case *ContractReceiveNewPaymentNetwork:
		contractReceiveNewPaymentNetwork, _ := stateChange.(*ContractReceiveNewPaymentNetwork)
		iteration = handleNewPaymentNetwork(chainState, contractReceiveNewPaymentNetwork)
	case *ContractReceiveNewTokenNetwork:
		contractReceiveNewTokenNetwork, _ := stateChange.(*ContractReceiveNewTokenNetwork)
		iteration = handleTokenadded(chainState, contractReceiveNewTokenNetwork)
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		iteration = handleContractReceiveChannelClosed(chainState, contractReceiveChannelClosed)
	case *ReceiveDelivered:
		receiveDelivered, _ := stateChange.(*ReceiveDelivered)
		iteration = handleDelivered(chainState, receiveDelivered)
	case *ReceiveProcessed:
		receiveProcessed, _ := stateChange.(*ReceiveProcessed)
		iteration = handleProcessed(chainState, receiveProcessed)
	}

	return iteration
}

func isTransactionEffectSatisfied(chainState *ChainState, transaction Event,
	stateChange StateChange) bool {

	if receiveUpdateTransfer, ok := stateChange.(*ContractReceiveUpdateTransfer); ok {
		if sendChannelUpdateTransfer, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
			if receiveUpdateTransfer.ChannelIdentifier == sendChannelUpdateTransfer.ChannelIdentifier &&
				receiveUpdateTransfer.Nonce == sendChannelUpdateTransfer.BalanceProof.Nonce {
				return true
			}
		}
	}

	if receiveChannelClosed, ok := stateChange.(*ContractReceiveChannelClosed); ok {
		if sendChannelClose, ok := transaction.(*ContractSendChannelClose); ok {
			if receiveChannelClosed.ChannelIdentifier == sendChannelClose.ChannelIdentifier {
				return true
			}
		}
	}

	if receiveChannelSettled, ok := stateChange.(*ContractReceiveChannelSettled); ok {
		if sendChannelSettle, ok := transaction.(*ContractSendChannelSettle); ok {
			if receiveChannelSettled.ChannelIdentifier == sendChannelSettle.ChannelIdentifier {
				return true
			}
		}
	}

	if receiveSecretReveal, ok := stateChange.(*ContractReceiveSecretReveal); ok {
		if sendSecretReveal, ok := transaction.(*ContractSendSecretReveal); ok {
			if typing.SliceEqual([]byte(receiveSecretReveal.Secret), []byte(sendSecretReveal.Secret)) {
				return true
			}
		}
	}

	return false
}

func isTransactionInvalidated(transaction Event, stateChange StateChange) bool {

	if receiveChannelSettled, ok := stateChange.(*ContractReceiveChannelSettled); ok {
		if sendChannelUpdateTransfer, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
			if receiveChannelSettled.ChannelIdentifier == sendChannelUpdateTransfer.ChannelIdentifier {
				return true
			}
		}
	}

	return false
}

func isTransactionExpired(transaction Event, blockNumber typing.BlockHeight) bool {

	if v, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
		if v.Expiration < typing.BlockExpiration(blockNumber) {
			return true
		}
	}

	if v, ok := transaction.(*ContractSendSecretReveal); ok {
		if v.Expiration < typing.BlockExpiration(blockNumber) {
			return true
		}
	}

	return false
}

func isTransactionPending(chainState *ChainState, transaction Event, stateChange StateChange) bool {
	if isTransactionEffectSatisfied(chainState, transaction, stateChange) == true {
		return false
	} else if isTransactionInvalidated(transaction, stateChange) {
		return false
	} else if isTransactionExpired(transaction, chainState.BlockHeight) {
		return false
	} else {
		return true
	}

	return true
}

func updateQueues(iteration TransitionResult, stateChange StateChange) {

	chainState := iteration.NewState.(*ChainState)

	if GetContractReceiveStateChange(stateChange) != nil {
		var newPendingTransactions []Event

		len := len(chainState.PendingTransactions)
		for i := 0; i < len; i++ {
			event := chainState.PendingTransactions[i]
			if isTransactionPending(chainState, event, stateChange) {
				newPendingTransactions = append(newPendingTransactions, event)
			}
		}
		chainState.PendingTransactions = newPendingTransactions
	}

	for e := iteration.Events.Front(); e != nil; e = e.Next() {
		if v, ok := e.Value.(Event); ok {
			event := GetSenderMessageEvent(v)
			if event != nil {
				queueIdentifier := QueueIdentifier{typing.Address(event.Recipient), event.ChannelIdentifier}
				chainState.QueueIdsToQueues[queueIdentifier] = append(chainState.QueueIdsToQueues[queueIdentifier], v)
			}

			if GetContractSendEvent(v) != nil {
				chainState.PendingTransactions = append(chainState.PendingTransactions, v)
			}
		}
	}

	return
}

func StateTransition(chainState State, stateChange StateChange) TransitionResult {

	iteration := handleStateChangeForNode(chainState, stateChange)

	updateQueues(iteration, stateChange)

	return iteration
}

func getChannelsCloseEvents(
	chainState *ChainState,
	tokenNetworkState *TokenNetworkState) *list.List {
	events := list.New()

	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0
	for _, v := range tokenNetworkState.partnerAddressesToChannels {
		filteredChannelStates := FilterChannelsByStatus(v, excludeStates)

		for e := filteredChannelStates.Front(); e != nil; e = e.Next() {
			channelState := e.Value.(*NettingChannelState)
			eventsForClose := EventsForClose(channelState, chainState.BlockHeight)
			events.PushBackList(eventsForClose)
		}
	}

	return events
}
