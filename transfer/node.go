package transfer

import (
	"reflect"

	"encoding/hex"

	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
	scUtils "github.com/saveio/themis/smartcontract/service/native/utils"
)

func getNetworks(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) (*PaymentNetworkState, *TokenNetworkState) {

	var tokenNetworkState *TokenNetworkState

	paymentNetworkState := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]

	if paymentNetworkState != nil {
		if tokenNetworkId, ok := paymentNetworkState.TokenAddressesToTokenIdentifiers[tokenAddress]; ok {
			tokenNetworkState = paymentNetworkState.TokenIdentifiersToTokenNetworks[tokenNetworkId]
		}
	}

	return paymentNetworkState, tokenNetworkState
}

func getTokenNetworkByTokenAddress(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenAddress common.TokenAddress) *TokenNetworkState {

	_, tokenNetworkState := getNetworks(chainState, paymentNetworkId, tokenAddress)

	return tokenNetworkState
}

func subDispatchToAllChannels(chainState *ChainState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {
	var events []Event
	for _, v := range chainState.IdentifiersToPaymentNetworks {
		for _, v2 := range v.TokenIdentifiersToTokenNetworks {
			for _, v3 := range v2.ChannelsMap {
				result := StateTransitionForChannel(v3, stateChange, blockNumber)
				events = append(events, result.Events...)
			}
		}
	}
	return TransitionResult{NewState: chainState, Events: events}
}

func subDispatchToAllLockedTransfers(chainState *ChainState, stateChange StateChange) TransitionResult {
	var events []Event
	for k := range chainState.PaymentMapping.SecretHashesToTask {
		result := subDispatchToPaymentTask(chainState, stateChange, k)
		events = append(events, result.Events...)
	}
	return TransitionResult{chainState, events}
}

func subDispatchToPaymentTask(chainState *ChainState, stateChange StateChange,
	secretHash common.SecretHash) *TransitionResult {

	blockNumber := chainState.BlockHeight
	subTask := chainState.PaymentMapping.SecretHashesToTask[secretHash]
	var err error
	var events []Event
	var subIteration *TransitionResult

	if subTask != nil {
		log.Debug("[subDispatchToPaymentTask] subTask Type: ", reflect.TypeOf(subTask).String())
		log.Debug("[subDispatchToPaymentTask] stateChange Type: ", reflect.TypeOf(stateChange).String())
		log.Debug("[subDispatchToPaymentTask] secretHash: ", secretHash)

		if reflect.TypeOf(subTask).String() == "*transfer.InitiatorTask" {
			log.Debug("-------------------------------")
			iSubTask := subTask.(*InitiatorTask)
			TokenNetworkId := iSubTask.TokenNetworkId
			tokenNetworkState := GetTokenNetworkByIdentifier(chainState, TokenNetworkId)
			if tokenNetworkState != nil {
				ms := iSubTask.ManagerState.(*InitiatorTransferState)
				paymentState := &InitiatorPaymentState{
					Initiator: ms,
				}
				subIteration = ImStateTransition(paymentState, stateChange,
					tokenNetworkState.ChannelsMap, blockNumber)
				events = subIteration.Events
				for _, e := range events {
					log.Debug("[subDispatchToPaymentTask]:", reflect.TypeOf(e).String())
				}
			} else {
				log.Warn("[subDispatchToPaymentTask] tokenNetworkState is nil")
			}
		} else if reflect.TypeOf(subTask).String() == "*transfer.MediatorTask" {
			mSubTask := subTask.(*MediatorTask)
			TokenNetworkId := mSubTask.TokenNetworkId
			tokenNetworkState := GetTokenNetworkByIdentifier(chainState, TokenNetworkId)
			if tokenNetworkState != nil {
				ms := mSubTask.MediatorState.(*MediatorTransferState)
				subIteration, err = MdStateTransition(ms, stateChange, tokenNetworkState.ChannelsMap, blockNumber)
				if err != nil {
					log.Error("MdStateTransition Err: ", err.Error())
				}
				events = subIteration.Events
			}
		} else if reflect.TypeOf(subTask).String() == "*transfer.TargetTask" {
			tSubTask := subTask.(*TargetTask)
			TokenNetworkId := tSubTask.TokenNetworkId
			channelId := tSubTask.ChannelId

			channelState := GetChannelStateByTokenNetworkId(chainState,
				TokenNetworkId, channelId)

			if channelState != nil {
				subIteration = TgStateTransition(tSubTask.TargetState.(*TargetTransferState),
					stateChange, channelState, blockNumber)
				events = subIteration.Events
			}
		} else {
			log.Errorf("[subDispatchToPaymentTask] Unknown subTask type")
		}

		if subIteration != nil && IsStateNil(subIteration.NewState) {
			log.Debug("[subDispatchToPaymentTask] delete SecretHashesToTask %v", secretHash)
			delete(chainState.PaymentMapping.SecretHashesToTask, secretHash)
		}
	} else {
		log.Warn("subTask is nil, HashValue: ", hex.EncodeToString(secretHash[:]))
	}

	return &TransitionResult{NewState: chainState, Events: events}
}

func subDispatchInitiatorTask(chainState *ChainState, stateChange StateChange,
	TokenNetworkId common.TokenNetworkID, secretHash common.SecretHash) *TransitionResult {

	blockNumber := chainState.BlockHeight
	subTask := chainState.PaymentMapping.SecretHashesToTask[secretHash]

	var isValidSubTask bool
	var managerState *InitiatorPaymentState

	if subTask == nil {
		isValidSubTask = true
		managerState = nil
	} else if subTask != nil && reflect.TypeOf(subTask).String() == "*transfer.InitiatorTask" {
		initTask := subTask.(*InitiatorTask)
		isValidSubTask = TokenNetworkId == initTask.TokenNetworkId
		managerState = initTask.ManagerState.(*InitiatorPaymentState)
	} else {
		isValidSubTask = false
	}

	var events []Event
	if isValidSubTask {
		tokenNetworkState := GetTokenNetworkByIdentifier(chainState, TokenNetworkId)
		iteration := ImStateTransition(managerState, stateChange,
			tokenNetworkState.ChannelsMap, blockNumber)

		events = append(events, iteration.Events...)
		if !IsStateNil(iteration.NewState) {
			subTask = &InitiatorTask{
				TokenNetworkId: TokenNetworkId,
				ManagerState:   iteration.NewState,
			}
			chainState.PaymentMapping.SecretHashesToTask[secretHash] = subTask
		} else if _, ok := chainState.PaymentMapping.SecretHashesToTask[secretHash]; ok {
			delete(chainState.PaymentMapping.SecretHashesToTask, secretHash)
		}
	}
	return &TransitionResult{NewState: chainState, Events: events}
}

func subDispatchMediatorTask(chainState *ChainState, stateChange StateChange,
	TokenNetworkId common.TokenNetworkID, secretHash common.SecretHash) *TransitionResult {

	blockNumber := chainState.BlockHeight
	subTask := chainState.PaymentMapping.SecretHashesToTask[secretHash]
	log.Debug("\n[subDispatchMediatorTask] secretHash:", secretHash)

	var isValidSubTask bool
	var mediatorState *MediatorTransferState
	if subTask == nil {
		isValidSubTask = true
		mediatorState = nil
	} else if subTask != nil && reflect.TypeOf(subTask).String() == "*transfer.MediatorTask" {
		mTask := subTask.(*MediatorTask)
		isValidSubTask = TokenNetworkId == mTask.TokenNetworkId
		mediatorState = mTask.MediatorState.(*MediatorTransferState)
		log.Debug("[subDispatchMediatorTask] Secret: ", mediatorState.Secret)
	} else {
		isValidSubTask = false
	}

	if isValidSubTask {
		log.Debug("[subDispatchMediatorTask] isValidSubTask == true")
	}

	var events []Event
	if isValidSubTask {
		tokenNetworkState := GetTokenNetworkByIdentifier(chainState, TokenNetworkId)
		iteration, err := MdStateTransition(mediatorState, stateChange, tokenNetworkState.ChannelsMap, blockNumber)
		if err != nil {
			log.Error("[subDispatchMediatorTask] MdStateTransition: ", err.Error())
		}
		events = iteration.Events

		if !IsStateNil(iteration.NewState) {
			subTask = &MediatorTask{
				TokenNetworkId: TokenNetworkId,
				MediatorState:  iteration.NewState,
			}
			chainState.PaymentMapping.SecretHashesToTask[secretHash] = subTask
			log.Debug("[subDispatchMediatorTask] iteration.NewState")
		} else if _, ok := chainState.PaymentMapping.SecretHashesToTask[secretHash]; ok {
			log.Debug("[subDispatchMediatorTask] delete SecretHashesToTask %v", secretHash)
			delete(chainState.PaymentMapping.SecretHashesToTask, secretHash)
		}
	}

	return &TransitionResult{NewState: chainState, Events: events}
}

func subDispatchTargetTask(chainState *ChainState, stateChange StateChange,
	TokenNetworkId common.TokenNetworkID, channelId common.ChannelID,
	secretHash common.SecretHash) *TransitionResult {

	blockNumber := chainState.BlockHeight
	subTask := chainState.PaymentMapping.SecretHashesToTask[secretHash]

	var isValidSubTask bool
	var targetState *TargetTransferState

	if subTask == nil {
		isValidSubTask = true
		targetState = nil
	} else if subTask != nil && reflect.TypeOf(subTask).String() == "*transfer.TargetTask" {
		tTask := subTask.(*TargetTask)
		isValidSubTask = TokenNetworkId == tTask.TokenNetworkId
		targetState = tTask.TargetState.(*TargetTransferState)
	} else {
		isValidSubTask = false
	}

	var events []Event
	var channelState *NettingChannelState
	if isValidSubTask {
		channelState = GetChannelStateByTokenNetworkId(chainState,
			TokenNetworkId, channelId)
	}

	if channelState != nil {
		iteration := TgStateTransition(targetState, stateChange, channelState, blockNumber)
		if iteration == nil {
			return &TransitionResult{NewState: chainState, Events: nil}
		} else {
			events = iteration.Events
			if !IsStateNil(iteration.NewState) {
				subTask = &TargetTask{
					TokenNetworkId: TokenNetworkId,
					ChannelId:      channelId,
					TargetState:    iteration.NewState,
				}
				chainState.PaymentMapping.SecretHashesToTask[secretHash] = subTask
			} else if _, ok := chainState.PaymentMapping.SecretHashesToTask[secretHash]; ok {
				log.Debug("[subDispatchTargetTask] delete SecretHashesToTask %v", secretHash)
				delete(chainState.PaymentMapping.SecretHashesToTask, secretHash)
			}
		}
	}
	return &TransitionResult{NewState: chainState, Events: events}
}

func maybeAddTokenNetwork(chainState *ChainState, paymentNetworkId common.PaymentNetworkID,
	tokenNetworkState *TokenNetworkState) {

	TokenNetworkId := tokenNetworkState.Address
	tokenAddress := tokenNetworkState.TokenAddress

	paymentNetworkState, tokenNetworkStatePrevious := getNetworks(chainState,
		paymentNetworkId, tokenAddress)

	if paymentNetworkState == nil {
		paymentNetworkState = NewPaymentNetworkState()
		paymentNetworkState.Address = common.PaymentNetworkID(scUtils.MicroPayContractAddress)
		paymentNetworkState.TokenIdentifiersToTokenNetworks[TokenNetworkId] = tokenNetworkState
		paymentNetworkState.TokenAddressesToTokenIdentifiers[tokenAddress] = tokenNetworkState.Address

		chainState.IdentifiersToPaymentNetworks[paymentNetworkId] = paymentNetworkState
	}

	if tokenNetworkStatePrevious == nil {
		paymentNetworkState.TokenIdentifiersToTokenNetworks[TokenNetworkId] = tokenNetworkState
		paymentNetworkState.TokenAddressesToTokenIdentifiers[tokenAddress] = TokenNetworkId
	}
	return
}

func inPlaceDeleteMessageQueue(chainState *ChainState, stateChange StateChange, queueId QueueId) {
	queue, ok := chainState.QueueIdsToQueues[queueId]
	if !ok {
		//log.Infof("[inPlaceDeleteMessageQueue] queueId is not in QueueIdsToQueues.")
		return
	}

	newQueue := inPlaceDeleteMessage(queue, stateChange)
	if len(newQueue) == 0 {
		delete(chainState.QueueIdsToQueues, queueId)
	} else {
		chainState.QueueIdsToQueues[queueId] = newQueue
	}

	return
}

func inPlaceDeleteMessage(messageQueue []Event, stateChange StateChange) []Event {
	//found := false
	log.Debug("[inPlaceDeleteMessageQueue] called for:", reflect.TypeOf(stateChange).String())
	if messageQueue == nil {
		return messageQueue
	}
	sender, messageId := GetSenderAndMessageId(stateChange)
	len := len(messageQueue)
	for i := 0; i < len; {
		message := GetSenderMessageEvent(messageQueue[i])
		//log.Debugf("[inPlaceDeleteMessageQueue] message: %v, sender:%v, messageId:%v", message, sender, messageId)
		if message.MessageId == messageId && common.AddressEqual(common.Address(message.Recipient), sender) {
			//log.Debugf("[inPlaceDeleteMessageQueue] sender: %s, messageId: %d match", common.ToBase58(sender), messageId)
			messageQueue = append(messageQueue[:i], messageQueue[i+1:]...)
			len--
			//found = true
		} else {
			i++
		}
	}
	//if !found {
	//	log.Errorf("[inPlaceDeleteMessageQueue], sender: %s, messageId: %d not found", common.ToBase58(sender), messageId)
	//}
	return messageQueue
}

func handleBlockForNode(chainState *ChainState, stateChange *Block) *TransitionResult {
	var events []Event

	blockNumber := stateChange.BlockHeight
	chainState.BlockHeight = blockNumber

	channelsResult := subDispatchToAllChannels(chainState, stateChange, blockNumber)
	transfersResult := subDispatchToAllLockedTransfers(chainState, stateChange)

	events = append(events, channelsResult.Events...)
	events = append(events, transfersResult.Events...)

	return &TransitionResult{chainState, events}
}

func handleChainInit(chainState *ChainState, stateChange *ActionInitChain) *TransitionResult {
	if chainState == nil {
		result := NewChainState()
		result.BlockHeight = stateChange.BlockHeight
		result.Address = stateChange.OurAddress
		result.ChainId = stateChange.ChainId
		return &TransitionResult{NewState: result, Events: nil}
	}
	return &TransitionResult{NewState: chainState, Events: nil}
}

func handleTokenNetworkAction(chainState *ChainState, stateChange StateChange,
	tokenNetworkId common.TokenNetworkID) *TransitionResult {

	var events []Event
	tokenNetworkState := GetTokenNetworkByIdentifier(chainState, tokenNetworkId)
	paymentNetworkState := GetTokenNetworkRegistryByTokenNetworkId(chainState, tokenNetworkId)
	if paymentNetworkState == nil {
		return &TransitionResult{}
	}
	paymentNetworkId := paymentNetworkState.Address

	if tokenNetworkState != nil {
		iteration := stateTransitionForNetwork(paymentNetworkId, tokenNetworkState,
			stateChange, chainState.BlockHeight)
		if IsStateNil(iteration.NewState) {
			paymentNetworkState = searchPaymentNetworkByTokenNetworkId(chainState, tokenNetworkId)

			delete(paymentNetworkState.TokenAddressesToTokenIdentifiers, tokenNetworkState.TokenAddress)
			delete(paymentNetworkState.TokenIdentifiersToTokenNetworks, tokenNetworkId)
		}
		events = iteration.Events
	}
	return &TransitionResult{NewState: chainState, Events: events}
}

func handleContractReceiveChannelClosed(chainState *ChainState,
	stateChange *ContractReceiveChannelClosed) *TransitionResult {

	channelId := GetChannelId(stateChange)
	channelState := GetChannelStateByTokenNetworkId(
		chainState, common.TokenNetworkID{}, channelId)

	if channelState != nil {
		queueId := QueueId{channelState.PartnerState.Address, channelId}

		if _, ok := chainState.QueueIdsToQueues[queueId]; ok {
			delete(chainState.QueueIdsToQueues, queueId)
		}
	}

	return handleTokenNetworkAction(chainState, stateChange, stateChange.TokenNetworkId)
}

func handleDelivered(chainState *ChainState, stateChange *ReceiveDelivered) *TransitionResult {
	//log.Debugf("[handleDelivered] stateChange MessageId: %v\n", stateChange.MessageId)
	queueId := QueueId{stateChange.Sender, 0}
	inPlaceDeleteMessageQueue(chainState, stateChange, queueId)
	//
	//return &TransitionResult{chainState, nil}
	return &TransitionResult{chainState, nil}
}

func handleNewTokenNetwork(chainState *ChainState, stateChange *ActionNewTokenNetwork) *TransitionResult {

	tokenNetworkState := stateChange.TokenNetwork
	paymentNetworkId := stateChange.PaymentNetworkId

	maybeAddTokenNetwork(chainState, paymentNetworkId, tokenNetworkState)
	return &TransitionResult{chainState, nil}
}

func handleLeaveAllNetworks(chainState *ChainState) *TransitionResult {
	var events []Event

	for _, v := range chainState.IdentifiersToPaymentNetworks {
		for _, v := range v.TokenIdentifiersToTokenNetworks {
			result := getChannelsCloseEvents(chainState, v)
			events = append(events, result...)
		}
	}

	return &TransitionResult{NewState: chainState, Events: events}
}

func handleNewPaymentNetwork(chainState *ChainState,
	stateChange *ContractReceiveNewPaymentNetwork) *TransitionResult {
	var events []Event

	paymentNetwork := stateChange.PaymentNetwork
	paymentNetworkId := paymentNetwork.Address

	_, ok := chainState.IdentifiersToPaymentNetworks[paymentNetworkId]
	if !ok {
		chainState.IdentifiersToPaymentNetworks[paymentNetworkId] = paymentNetwork
	}

	return &TransitionResult{chainState, events}
}

func handleTokenAdded(chainState *ChainState, stateChange *ContractReceiveNewTokenNetwork) *TransitionResult {
	maybeAddTokenNetwork(chainState, stateChange.PaymentNetworkId, stateChange.TokenNetwork)
	return &TransitionResult{chainState, nil}
}

func handleSecretReveal(chainState *ChainState, stateChange *ReceiveSecretReveal) *TransitionResult {
	secretHash := common.GetHash(stateChange.Secret)
	return subDispatchToPaymentTask(chainState, stateChange, secretHash)
}

func handleInitInitiator(chainState *ChainState, stateChange *ActionInitInitiator) *TransitionResult {
	transferDesc := stateChange.TransferDescription
	secretHash := transferDesc.SecretHash
	return subDispatchInitiatorTask(chainState, stateChange, transferDesc.TokenNetworkId, secretHash)
}

func handleInitMediator(chainState *ChainState, stateChange *ActionInitMediator) *TransitionResult {
	transfer := stateChange.FromTransfer
	secretHash := common.SecretHash(transfer.Lock.SecretHash)
	TokenNetworkId := transfer.BalanceProof.TokenNetworkId
	return subDispatchMediatorTask(chainState, stateChange, TokenNetworkId, secretHash)
}

func handleInitTarget(chainState *ChainState, stateChange *ActionInitTarget) *TransitionResult {
	transfer := stateChange.Transfer
	secretHash := common.SecretHash(transfer.Lock.SecretHash)
	channelId := transfer.BalanceProof.ChannelId
	TokenNetworkId := transfer.BalanceProof.TokenNetworkId
	return subDispatchTargetTask(chainState, stateChange, TokenNetworkId, channelId, secretHash)
}

func handleReceiveSecretRequest(chainState *ChainState, stateChange *ReceiveSecretRequest) *TransitionResult {
	secretHash := stateChange.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, secretHash)
}

func handleContractReceiveSecretReveal(chainState *ChainState, stateChange *ContractReceiveSecretReveal) *TransitionResult {
	secretHash := stateChange.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, secretHash)
}

func handleReceiveTransferRefundCancelRoute(chainState *ChainState, stateChange *ReceiveTransferRefundCancelRoute) *TransitionResult {
	secretHash := stateChange.Transfer.Lock.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, common.SecretHash(secretHash))
}

func handleReceiveTransferRefund(chainState *ChainState, stateChange *ReceiveTransferRefund) *TransitionResult {
	secretHash := stateChange.Transfer.Lock.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, common.SecretHash(secretHash))
}

func handleReceiveLockedExpired(chainState *ChainState, stateChange *ReceiveLockExpired) *TransitionResult {
	secretHash := stateChange.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, common.SecretHash(secretHash))
}

func handleProcessed(chainState *ChainState, stateChange *ReceiveProcessed) *TransitionResult {
	var events []Event
	for _, v := range chainState.QueueIdsToQueues {
		len := len(v)

		for i := 0; i < len; i++ {
			message := GetSenderMessageEvent(v[i])
			sender, messageId := GetSenderAndMessageId(stateChange)
			if message.MessageId == messageId && common.AddressEqual(message.Recipient, sender) {
				if message, ok := v[i].(*SendDirectTransfer); ok {
					channelState := GetChannelStateByTokenNetworkAndPartner(chainState,
						message.BalanceProof.TokenNetworkId, message.Recipient)

					paySuccess := &EventPaymentSentSuccess{
						PaymentNetworkId: channelState.PaymentNetworkId,
						TokenNetworkId:   channelState.TokenNetworkId,
						Identifier:       message.PaymentId,
						Amount:           message.BalanceProof.TransferredAmount,
						Target:           message.Recipient,
					}
					events = append(events, paySuccess)
				}
			}
		}
	}

	//log.Infof("[handleProcessed] MessageId: %d", stateChange.MessageId)
	for k := range chainState.QueueIdsToQueues {
		inPlaceDeleteMessageQueue(chainState, stateChange, k)
	}
	return &TransitionResult{chainState, events}
}

func handleReceiveUnlock(chainState *ChainState, stateChange *ReceiveUnlock) *TransitionResult {
	secretHash := stateChange.SecretHash
	return subDispatchToPaymentTask(chainState, stateChange, secretHash)
}

func handleStateChangeForNode(chainStateArg State, stateChange StateChange) *TransitionResult {
	chainState := chainStateArg.(*ChainState)

	var iteration *TransitionResult
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
	case *ActionInitInitiator:
		actionInitInitiator, _ := stateChange.(*ActionInitInitiator)
		iteration = handleInitInitiator(chainState, actionInitInitiator)
	case *ActionInitMediator:
		actionInitMediator, _ := stateChange.(*ActionInitMediator)
		iteration = handleInitMediator(chainState, actionInitMediator)
	case *ActionInitTarget:
		actionInitTarget, _ := stateChange.(*ActionInitTarget)
		iteration = handleInitTarget(chainState, actionInitTarget)
	case *ActionChannelClose:
		actionChannelClose, _ := stateChange.(*ActionChannelClose)
		iteration = handleTokenNetworkAction(chainState, stateChange, actionChannelClose.TokenNetworkId)
	case *ActionTransferDirect:
		actionTransferDirect, _ := stateChange.(*ActionTransferDirect)
		iteration = handleTokenNetworkAction(chainState, stateChange, actionTransferDirect.TokenNetworkId)
	case *ContractReceiveChannelNew:
		contractReceiveChannelNew, _ := stateChange.(*ContractReceiveChannelNew)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelNew.TokenNetworkId)
	case *ContractReceiveChannelNewBalance:
		contractReceiveChannelNewBalance, _ := stateChange.(*ContractReceiveChannelNewBalance)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelNewBalance.TokenNetworkId)
	case *ContractReceiveChannelSettled:
		contractReceiveChannelSettled, _ := stateChange.(*ContractReceiveChannelSettled)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelSettled.TokenNetworkId)
	case *ContractReceiveRouteNew:
		contractReceiveChannelSettled := stateChange.(*ContractReceiveRouteNew)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelSettled.TokenNetworkId)
	case *ContractReceiveRouteClosed:
		contractReceiveChannelSettled := stateChange.(*ContractReceiveRouteClosed)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelSettled.TokenNetworkId)
	case *ContractReceiveUpdateTransfer:
		contractReceiveUpdateTransfer, _ := stateChange.(*ContractReceiveUpdateTransfer)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveUpdateTransfer.TokenNetworkId)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleTokenNetworkAction(chainState, stateChange, receiveTransferDirect.TokenNetworkId)
	case *ActionWithdraw:
		actionWithdraw, _ := stateChange.(*ActionWithdraw)
		iteration = handleTokenNetworkAction(chainState, stateChange, actionWithdraw.TokenNetworkId)
	case *ReceiveWithdrawRequest:
		receiveWithdrawRequest, _ := stateChange.(*ReceiveWithdrawRequest)
		iteration = handleTokenNetworkAction(chainState, stateChange, receiveWithdrawRequest.TokenNetworkId)
	case *ReceiveWithdraw:
		receiveWithdraw, _ := stateChange.(*ReceiveWithdraw)
		iteration = handleTokenNetworkAction(chainState, stateChange, receiveWithdraw.TokenNetworkId)
	case *ContractReceiveChannelWithdraw:
		contractReceiveWithdraw, _ := stateChange.(*ContractReceiveChannelWithdraw)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveWithdraw.TokenNetworkId)
	case *ActionCooperativeSettle:
		actionCooperativeSettle, _ := stateChange.(*ActionCooperativeSettle)
		iteration = handleTokenNetworkAction(chainState, stateChange, actionCooperativeSettle.TokenNetworkId)
	case *ReceiveCooperativeSettleRequest:
		receiveCooperativeSettleRequest, _ := stateChange.(*ReceiveCooperativeSettleRequest)
		iteration = handleTokenNetworkAction(chainState, stateChange, receiveCooperativeSettleRequest.TokenNetworkId)
	case *ReceiveCooperativeSettle:
		receiveCooperativeSettle, _ := stateChange.(*ReceiveCooperativeSettle)
		iteration = handleTokenNetworkAction(chainState, stateChange, receiveCooperativeSettle.TokenNetworkId)
	case *ContractReceiveChannelCooperativeSettled:
		contractReceiveChannelCooperativeSettled, _ := stateChange.(*ContractReceiveChannelCooperativeSettled)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelCooperativeSettled.TokenNetworkId)
	case *ContractReceiveChannelBatchUnlock:
		contractReceiveChannelBatchUnlock, _ := stateChange.(*ContractReceiveChannelBatchUnlock)
		iteration = handleTokenNetworkAction(chainState, stateChange, contractReceiveChannelBatchUnlock.TokenNetworkId)
	case *ActionLeaveAllNetworks:
		iteration = handleLeaveAllNetworks(chainState)
	case *ContractReceiveNewPaymentNetwork:
		contractReceiveNewPaymentNetwork, _ := stateChange.(*ContractReceiveNewPaymentNetwork)
		iteration = handleNewPaymentNetwork(chainState, contractReceiveNewPaymentNetwork)
	case *ContractReceiveNewTokenNetwork:
		contractReceiveNewTokenNetwork, _ := stateChange.(*ContractReceiveNewTokenNetwork)
		iteration = handleTokenAdded(chainState, contractReceiveNewTokenNetwork)
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		iteration = handleContractReceiveChannelClosed(chainState, contractReceiveChannelClosed)
	case *ReceiveDelivered:
		receiveDelivered, _ := stateChange.(*ReceiveDelivered)
		iteration = handleDelivered(chainState, receiveDelivered)
	case *ReceiveProcessed:
		receiveProcessed, _ := stateChange.(*ReceiveProcessed)
		iteration = handleProcessed(chainState, receiveProcessed)
	case *ReceiveSecretReveal:
		receiveSecretReveal, _ := stateChange.(*ReceiveSecretReveal)
		iteration = handleSecretReveal(chainState, receiveSecretReveal)
	case *ReceiveSecretRequest:
		receiveSecretRequest, _ := stateChange.(*ReceiveSecretRequest)
		iteration = handleReceiveSecretRequest(chainState, receiveSecretRequest)
	case *ReceiveTransferRefundCancelRoute:
		receiveTransferRefundCancelRoute, _ := stateChange.(*ReceiveTransferRefundCancelRoute)
		iteration = handleReceiveTransferRefundCancelRoute(chainState, receiveTransferRefundCancelRoute)
	case *ReceiveTransferRefund:
		receiveTransferRefund, _ := stateChange.(*ReceiveTransferRefund)
		iteration = handleReceiveTransferRefund(chainState, receiveTransferRefund)
	case *ReceiveUnlock:
		receiveUnlock, _ := stateChange.(*ReceiveUnlock)
		iteration = handleReceiveUnlock(chainState, receiveUnlock)
	case *ContractReceiveSecretReveal:
		contractReceiveSecretReveal, _ := stateChange.(*ContractReceiveSecretReveal)
		iteration = handleContractReceiveSecretReveal(chainState, contractReceiveSecretReveal)
	case *ReceiveLockExpired:
		receiveLockExpired, _ := stateChange.(*ReceiveLockExpired)
		iteration = handleReceiveLockedExpired(chainState, receiveLockExpired)
	default:
		log.Warn("[node.handleStateChangeForNode] unknown stateChange Type: ",
			reflect.TypeOf(stateChange).String())
	}

	return iteration
}

func isTransactionEffectSatisfied(chainState *ChainState, transaction Event,
	stateChange StateChange) bool {

	if receiveUpdateTransfer, ok := stateChange.(*ContractReceiveUpdateTransfer); ok {
		if sendChannelUpdateTransfer, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
			if receiveUpdateTransfer.ChannelId == sendChannelUpdateTransfer.ChannelId &&
				receiveUpdateTransfer.Nonce == sendChannelUpdateTransfer.BalanceProof.Nonce {
				return true
			}
		}
	}

	if receiveChannelClosed, ok := stateChange.(*ContractReceiveChannelClosed); ok {
		if sendChannelClose, ok := transaction.(*ContractSendChannelClose); ok {
			if receiveChannelClosed.ChannelId == sendChannelClose.ChannelId {
				return true
			}
		}
	}

	if receiveChannelSettled, ok := stateChange.(*ContractReceiveChannelSettled); ok {
		if sendChannelSettle, ok := transaction.(*ContractSendChannelSettle); ok {
			if receiveChannelSettled.ChannelId == sendChannelSettle.ChannelId {
				return true
			}
		}
	}

	if receiveSecretReveal, ok := stateChange.(*ContractReceiveSecretReveal); ok {
		if sendSecretReveal, ok := transaction.(*ContractSendSecretReveal); ok {
			if common.SliceEqual([]byte(receiveSecretReveal.Secret), []byte(sendSecretReveal.Secret)) {
				return true
			}
		}
	}

	if receiveChannelWithdraw, ok := stateChange.(*ContractReceiveChannelWithdraw); ok {
		if sendChannelWithdraw, ok := transaction.(*ContractSendChannelWithdraw); ok {
			if receiveChannelWithdraw.ChannelId == sendChannelWithdraw.ChannelId &&
				receiveChannelWithdraw.Participant == sendChannelWithdraw.Participant {
				return true
			}
		}
	}

	if receiveChannelCooperativeSettled, ok := stateChange.(*ContractReceiveChannelCooperativeSettled); ok {
		if sendChannelCooperativeSettle, ok := transaction.(*ContractSendChannelCooperativeSettle); ok {
			if receiveChannelCooperativeSettled.ChannelId == sendChannelCooperativeSettle.ChannelId {
				return true
			}
		}
	}

	return false
}

func isTransactionInvalidated(transaction Event, stateChange StateChange) bool {

	if receiveChannelSettled, ok := stateChange.(*ContractReceiveChannelSettled); ok {
		if sendChannelUpdateTransfer, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
			if receiveChannelSettled.ChannelId == sendChannelUpdateTransfer.ChannelId {
				return true
			}
		}
	}

	return false
}

func isTransactionExpired(transaction Event, blockNumber common.BlockHeight) bool {

	if v, ok := transaction.(*ContractSendChannelUpdateTransfer); ok {
		if v.Expiration < common.BlockExpiration(blockNumber) {
			return true
		}
	}

	if v, ok := transaction.(*ContractSendSecretReveal); ok {
		if v.Expiration < common.BlockExpiration(blockNumber) {
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
}

func updateQueues(iteration *TransitionResult, stateChange StateChange) {
	var events []Event
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

	for _, e := range iteration.Events {
		event := GetSenderMessageEvent(e)
		if event != nil {
			queueId := QueueId{common.Address(event.Recipient), event.ChannelId}
			events = chainState.QueueIdsToQueues[queueId]
			events = append(events, e)
			chainState.QueueIdsToQueues[queueId] = events
			log.Debug("[updateQueses] add: ", reflect.TypeOf(e).String(), "queueId:", queueId)
		}

		if GetContractSendEvent(e) != nil {
			chainState.PendingTransactions = append(chainState.PendingTransactions, e)
		}
	}

	//if chainState != nil {
	//	if chainState.QueueIdsToQueues != nil {
	//		for _, v := range chainState.QueueIdsToQueues {
	//			l := len(v)
	//			for i := 0; i < l; i++ {
	//				e := v[i]
	//				log.Debug("[updateQueues] QueueIdsToQueues:", reflect.TypeOf(e).String())
	//			}
	//		}
	//	}
	//}
	return
}

func StateTransition(chainState State, stateChange StateChange) *TransitionResult {
	iteration := handleStateChangeForNode(chainState, stateChange)
	if IsStateNil(iteration.NewState) {
		log.Warn("[node.StateTransition] iteration.NewState is nil")
	}
	for _, e := range iteration.Events {
		log.Debug("[node.StateTransition]:", reflect.TypeOf(e).String())
	}
	updateQueues(iteration, stateChange)
	return iteration
}

func getChannelsCloseEvents(chainState *ChainState, tokenNetworkState *TokenNetworkState) []Event {
	var events []Event
	excludeStates := make(map[string]int)
	excludeStates[ChannelStateUnusable] = 0
	for _, v := range tokenNetworkState.PartnerAddressesToChannels {
		filteredChannelStates := FilterChannelsByStatus(v, excludeStates)

		for e := filteredChannelStates.Front(); e != nil; e = e.Next() {
			channelState := e.Value.(*NettingChannelState)
			eventsForClose := EventsForClose(channelState, chainState.BlockHeight)
			events = append(events, eventsForClose...)
		}
	}
	return events
}
