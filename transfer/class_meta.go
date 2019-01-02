package transfer

const (
	InitiatorTaskClassId = 1 + iota
	MediatorTaskClassId
	TargetTaskClassId
	ChainStateClassId
	PaymentNetworkStateClassId
	PaymentMappingStateClassId
	TokenNetworkStateClassId
	NettingChannelStateClassId
	NettingChannelEndStateClassId
	TransactionExecutionStatusClassId
	UnlockPartialProofStateClassId
	HashTimeLockStateClassId
)

const (
	AuthenticatedSenderStateChangeClassId = 100 + iota
	ContractReceiveStateChangeClassId
	BlockClassId
	ActionCancelPaymentClassId
	ActionChannelCloseClassId
	ActionCancelTransferClassId
	ActionTransferDirectClassId
	ContractReceiveChannelNewClassId
	ContractReceiveChannelClosedClassId
	ActionInitChainClassId
	ActionNewTokenNetworkClassId
	ContractReceiveChannelNewBalanceClassId
	ContractReceiveChannelSettledClassId
	ActionLeaveAllNetworksClassId
	ActionChangeNodeNetworkStateClassId
	ContractReceiveNewPaymentNetworkClassId
	ContractReceiveNewTokenNetworkClassId
	ContractReceiveSecretRevealClassId
	ContractReceiveChannelBatchUnlockClassId
	ContractReceiveRouteNewClassId
	ContractReceiveRouteClosedClassId
	ContractReceiveUpdateTransferClassId
	ReceiveTransferDirectClassId
	ReceiveUnlockClassId
	ReceiveDeliveredClassId
	ReceiveProcessedClassId
)

const (
	SendMessageEventClassId = 200 + iota
	ContractSendEventClassId
	ContractSendExpirableEventClassId
	EventPaymentSentSuccessClassId
	EventPaymentSentFailedClassId
	EventPaymentReceivedSuccessClassId
	EventTransferReceivedInvalidDirectTransferClassId
	SendDirectTransferClassId
	SendProcessedClassId
	ContractSendChannelCloseClassId
	ContractSendChannelSettleClassId
	ContractSendChannelUpdateTransferClassId
	ContractSendChannelBatchUnlockClassId
	ContractSendSecretRevealClassId
)

// class meta for State
func (self InitiatorTask) ClassId() int {
	return InitiatorTaskClassId
}

func (self MediatorTask) ClassId() int {
	return MediatorTaskClassId
}

func (self TargetTask) ClassId() int {
	return TargetTaskClassId
}

func (self ChainState) ClassId() int {
	return ChainStateClassId
}

func (self PaymentNetworkState) ClassId() int {
	return PaymentNetworkStateClassId
}

func (self PaymentMappingState) ClassId() int {
	return PaymentMappingStateClassId
}

func (self TokenNetworkState) ClassId() int {
	return TokenNetworkStateClassId
}

func (self NettingChannelState) ClassId() int {
	return NettingChannelStateClassId
}

func (self NettingChannelEndState) ClassId() int {
	return NettingChannelEndStateClassId
}

func (self TransactionExecutionStatus) ClassId() int {
	return TransactionExecutionStatusClassId
}

func (self UnlockPartialProofState) ClassId() int {
	return UnlockPartialProofStateClassId
}

func (self HashTimeLockState) ClassId() int {
	return HashTimeLockStateClassId
}

// class meta for StateChange
func (self ContractReceiveStateChange) ClassId() int {
	return ContractReceiveStateChangeClassId
}

func (self Block) ClassId() int {
	return BlockClassId
}

func (self ActionChannelClose) ClassId() int {
	return ActionChannelCloseClassId
}

func (self ActionTransferDirect) ClassId() int {
	return ActionTransferDirectClassId
}

func (self ContractReceiveChannelNew) ClassId() int {
	return ContractReceiveChannelNewClassId
}

func (self ContractReceiveChannelClosed) ClassId() int {
	return ContractReceiveChannelClosedClassId
}

func (self ActionInitChain) ClassId() int {
	return ActionInitChainClassId
}

func (self ActionNewTokenNetwork) ClassId() int {
	return ActionNewTokenNetworkClassId
}

func (self ContractReceiveChannelNewBalance) ClassId() int {
	return ContractReceiveChannelNewBalanceClassId
}

func (self ContractReceiveChannelSettled) ClassId() int {
	return ContractReceiveChannelSettledClassId
}

func (self ActionLeaveAllNetworks) ClassId() int {
	return ActionLeaveAllNetworksClassId
}

func (self ActionChangeNodeNetworkState) ClassId() int {
	return ActionChangeNodeNetworkStateClassId
}

func (self ContractReceiveNewPaymentNetwork) ClassId() int {
	return ContractReceiveNewPaymentNetworkClassId
}

func (self ContractReceiveNewTokenNetwork) ClassId() int {
	return ContractReceiveNewTokenNetworkClassId
}

func (self ContractReceiveUpdateTransfer) ClassId() int {
	return ContractReceiveUpdateTransferClassId
}

func (self ReceiveTransferDirect) ClassId() int {
	return ReceiveTransferDirectClassId
}

func (self ReceiveUnlock) ClassId() int {
	return ReceiveUnlockClassId
}

func (self ReceiveDelivered) ClassId() int {
	return ReceiveDeliveredClassId
}

func (self ReceiveProcessed) ClassId() int {
	return ReceiveProcessedClassId
}

// class meta for Event
func (self SendMessageEvent) ClassId() int {
	return SendMessageEventClassId
}

func (self EventPaymentSentSuccess) ClassId() int {
	return EventPaymentSentSuccessClassId
}

func (self EventPaymentSentFailed) ClassId() int {
	return EventPaymentSentFailedClassId
}

func (self EventPaymentReceivedSuccess) ClassId() int {
	return EventPaymentReceivedSuccessClassId
}

func (self EventTransferReceivedInvalidDirectTransfer) ClassId() int {
	return EventTransferReceivedInvalidDirectTransferClassId
}

func (self SendDirectTransfer) ClassId() int {
	return SendDirectTransferClassId
}

func (self SendProcessed) ClassId() int {
	return SendProcessedClassId
}

func (self ContractSendChannelClose) ClassId() int {
	return ContractSendChannelCloseClassId
}

func (self ContractSendChannelSettle) ClassId() int {
	return ContractSendChannelSettleClassId
}

func (self ContractSendChannelUpdateTransfer) ClassId() int {
	return ContractSendChannelUpdateTransferClassId
}

func (self ContractSendChannelBatchUnlock) ClassId() int {
	return ContractSendChannelBatchUnlockClassId
}

func (self ContractSendSecretReveal) ClassId() int {
	return ContractSendSecretRevealClassId
}

// create class instance based on ClassId

func CreateObjectByClassId(classId int) interface{} {
	var result interface{}

	result = CreateStateByClassId(classId)
	if result != nil {
		return result
	}

	result = CreateStateChangeByClassId(classId)
	if result != nil {
		return result
	}

	result = CreateEventByClassId(classId)
	if result != nil {
		return result
	}
	return result
}

func CreateStateByClassId(classId int) interface{} {
	var result interface{}

	switch classId {
	case InitiatorTaskClassId:
		result = new(InitiatorTask)
	case MediatorTaskClassId:
		result = new(MediatorTask)
	case TargetTaskClassId:
		result = new(TargetTask)
	case ChainStateClassId:
		result = new(ChainState)
	case PaymentNetworkStateClassId:
		result = new(PaymentNetworkState)
	case PaymentMappingStateClassId:
		result = new(PaymentMappingState)
	case TokenNetworkStateClassId:
		result = new(TokenNetworkState)
	case NettingChannelStateClassId:
		result = new(NettingChannelState)
	case NettingChannelEndStateClassId:
		result = new(NettingChannelEndState)
	case TransactionExecutionStatusClassId:
		result = new(TransactionExecutionStatus)
	case UnlockPartialProofStateClassId:
		result = new(UnlockPartialProofState)
	case HashTimeLockStateClassId:
		result = new(HashTimeLockState)
	}

	return result
}

func CreateStateChangeByClassId(classId int) interface{} {
	var result interface{}

	switch classId {
	case AuthenticatedSenderStateChangeClassId:
		result = new(AuthenticatedSenderStateChange)
	case ContractReceiveStateChangeClassId:
		result = new(ContractReceiveStateChange)
	case BlockClassId:
		result = new(Block)
	case ActionCancelPaymentClassId:
		result = new(ActionCancelPayment)
	case ActionChannelCloseClassId:
		result = new(ActionChannelClose)
	case ActionCancelTransferClassId:
		result = new(ActionCancelTransfer)
	case ActionTransferDirectClassId:
		result = new(ActionTransferDirect)
	case ContractReceiveChannelNewClassId:
		result = new(ContractReceiveChannelNew)
	case ContractReceiveChannelClosedClassId:
		result = new(ContractReceiveChannelClosed)
	case ActionInitChainClassId:
		result = new(ActionInitChain)
	case ActionNewTokenNetworkClassId:
		result = new(ActionNewTokenNetwork)
	case ContractReceiveChannelNewBalanceClassId:
		result = new(ContractReceiveChannelNewBalance)
	case ContractReceiveChannelSettledClassId:
		result = new(ContractReceiveChannelSettled)
	case ActionLeaveAllNetworksClassId:
		result = new(ActionLeaveAllNetworks)
	case ActionChangeNodeNetworkStateClassId:
		result = new(ActionChangeNodeNetworkState)
	case ContractReceiveNewPaymentNetworkClassId:
		result = new(ContractReceiveNewPaymentNetwork)
	case ContractReceiveNewTokenNetworkClassId:
		result = new(ContractReceiveNewTokenNetwork)
	case ContractReceiveSecretRevealClassId:
		result = new(ContractReceiveSecretReveal)
	case ContractReceiveChannelBatchUnlockClassId:
		result = new(ContractReceiveChannelBatchUnlock)
	case ContractReceiveRouteNewClassId:
		result = new(ContractReceiveRouteNew)
	case ContractReceiveRouteClosedClassId:
		result = new(ContractReceiveRouteClosed)
	case ContractReceiveUpdateTransferClassId:
		result = new(ContractReceiveUpdateTransfer)
	case ReceiveTransferDirectClassId:
		result = new(ReceiveTransferDirect)
	case ReceiveUnlockClassId:
		result = new(ReceiveUnlock)
	case ReceiveDeliveredClassId:
		result = new(ReceiveDelivered)
	case ReceiveProcessedClassId:
		result = new(ReceiveProcessed)
	}

	return result
}

func CreateEventByClassId(classId int) interface{} {
	var result interface{}

	switch classId {
	case SendMessageEventClassId:
		result = new(SendMessageEvent)
	case ContractSendEventClassId:
		result = new(ContractSendEvent)
	case ContractSendExpirableEventClassId:
		result = new(ContractSendExpirableEvent)
	case EventPaymentSentSuccessClassId:
		result = new(EventPaymentSentSuccess)
	case EventPaymentSentFailedClassId:
		result = new(EventPaymentSentFailed)
	case EventPaymentReceivedSuccessClassId:
		result = new(EventPaymentReceivedSuccess)
	case EventTransferReceivedInvalidDirectTransferClassId:
		result = new(EventTransferReceivedInvalidDirectTransfer)
	case SendDirectTransferClassId:
		result = new(SendDirectTransfer)
	case SendProcessedClassId:
		result = new(SendProcessed)
	case ContractSendChannelCloseClassId:
		return new(ContractSendChannelClose)
	case ContractSendChannelSettleClassId:
		return new(ContractSendChannelSettle)
	case ContractSendChannelUpdateTransferClassId:
		return new(ContractSendChannelUpdateTransfer)
	case ContractSendChannelBatchUnlockClassId:
		return new(ContractSendChannelBatchUnlock)
	case ContractSendSecretRevealClassId:
		return new(ContractSendSecretReveal)
	}

	return result
}

func AdjustObject(v interface{}) {
	switch v.(type) {
	case *ChainState:
		rv, _ := v.(*ChainState)
		if rv.PendingTransactions == nil {

		}
	case *TokenNetworkState:
		//rv, _ := v.(*TokenNetworkState)

	}
}
