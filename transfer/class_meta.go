package transfer

const ChannelIdentifierGlobalQueue = 0
const DefaultNumberOfBlockConfirmations = 5

//state
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
	InitiatorTransferStateClassId
	InitiatorPaymentStateClassId
	TargetTransferStateClassId
	MediatorTransferStateClassId
)

//action
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
	ActionInitInitiatorClassId
	ReceiveSecretRequestClassId
	ReceiveSecretRevealClassId
	ReceiveLockExpiredClassId
	ReceiveTransferRefundCancelRouteClassId
	ReceiveTransferRefundClassId
	ActionInitTargetClassId
	ActionInitMediatorClassId
	ActionWithdrawClassId
	ReceiveWithdrawRequestClassId
	ReceiveWithdrawClassId
	ContractReceiveChannelWithdrawClassId
	ActionCooperativeSettleClassId
	ReceiveCooperativeSettleRequestClassId
	ReceiveCooperativeSettleClassId
	ContractReceiveChannelCooperativeSettledClassId
)

//event
const (
	SendMessageEventClassId = 200 + iota
	ContractSendEventClassId
	ContractSendExpireAbleEventClassId
	EventPaymentSentSuccessClassId
	EventPaymentSentFailedClassId
	EventPaymentReceivedSuccessClassId
	EventTransferReceivedInvalidDirectTransferClassId
	SendDirectTransferClassId
	SendLockedTransferClassId
	SendLockExpiredClassId
	SendBalanceProofClassId
	SendSecretRevealClassId
	SendSecretRequestClassId
	SendRefundTransferClassId
	SendProcessedClassId
	ContractSendChannelCloseClassId
	ContractSendChannelSettleClassId
	ContractSendChannelUpdateTransferClassId
	ContractSendChannelBatchUnlockClassId
	ContractSendSecretRevealClassId
	EventUnlockSuccessClassId
	EventInvalidReceivedTransferRefundClassId
	EventInvalidReceivedLockExpiredClassId
	EventInvalidReceivedLockedTransferClassId
	EventUnlockFailedClassId
	EventUnlockClaimFailedClassId
	EventInvalidReceivedUnlockClassId
	EventUnlockClaimSuccessClassId
	EventUnexpectedSecretRevealClassId
	SendWithdrawRequestClassId
	SendWithdrawClassId
	ContractSendChannelWithdrawClassId
	EventWithdrawRequestSentFailedClassId
	EventInvalidReceivedWithdrawRequestClassId
	EventInvalidReceivedWithdrawClassId
	EventWithdrawRequestTimeoutClassId
	SendCooperativeSettleRequestClassId
	SendCooperativeSettleClassId
	ContractSendChannelCooperativeSettleClassId
	EventCooperativeSettleRequestSentFailedClassId
	EventInvalidReceivedCooperativeSettleRequestClassId
	EventInvalidReceivedCooperativeSettleClassId
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

func (self InitiatorTransferState) ClassId() int {
	return InitiatorTransferStateClassId
}

func (self InitiatorPaymentState) ClassId() int {
	return InitiatorPaymentStateClassId
}

func (self TargetTransferState) ClassId() int {
	return TargetTransferStateClassId
}

func (self MediatorTransferState) ClassId() int {
	return MediatorTransferStateClassId
}

// class meta for StateChange
func (self AuthenticatedSenderStateChange) ClassId() int {
	return AuthenticatedSenderStateChangeClassId
}

func (self ContractReceiveStateChange) ClassId() int {
	return ContractReceiveStateChangeClassId
}

func (self Block) ClassId() int {
	return BlockClassId
}

func (self ActionCancelPayment) ClassId() int {
	return ActionCancelPaymentClassId
}

func (self ActionChannelClose) ClassId() int {
	return ActionChannelCloseClassId
}

func (self ActionCancelTransfer) ClassId() int {
	return ActionCancelTransferClassId
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

func (self ContractReceiveSecretReveal) ClassId() int {
	return ContractReceiveSecretRevealClassId
}

func (self ContractReceiveChannelBatchUnlock) ClassId() int {
	return ContractReceiveChannelBatchUnlockClassId
}

func (self ContractReceiveRouteNew) ClassId() int {
	return ContractReceiveRouteNewClassId
}

func (self ContractReceiveRouteClosed) ClassId() int {
	return ContractReceiveRouteClosedClassId
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

func (self ActionInitInitiator) ClassId() int {
	return ActionInitInitiatorClassId
}

func (self ReceiveSecretRequest) ClassId() int {
	return ReceiveSecretRequestClassId
}

func (self ReceiveSecretReveal) ClassId() int {
	return ReceiveSecretRevealClassId
}

func (self ReceiveLockExpired) ClassId() int {
	return ReceiveLockExpiredClassId
}

func (self ReceiveTransferRefundCancelRoute) ClassId() int {
	return ReceiveTransferRefundCancelRouteClassId
}

func (self ReceiveTransferRefund) ClassId() int {
	return ReceiveTransferRefundClassId
}

func (self ActionInitTarget) ClassId() int {
	return ActionInitTargetClassId
}

func (self ActionInitMediator) ClassId() int {
	return ActionInitMediatorClassId
}

func (self ActionWithdraw) ClassId() int {
	return ActionWithdrawClassId
}

func (self ReceiveWithdrawRequest) ClassId() int {
	return ReceiveWithdrawRequestClassId
}

func (self ReceiveWithdraw) ClassId() int {
	return ReceiveWithdrawClassId
}

func (self ContractReceiveChannelWithdraw) ClassId() int {
	return ContractReceiveChannelWithdrawClassId
}

func (self ActionCooperativeSettle) ClassId() int {
	return ActionCooperativeSettleClassId
}

func (self ReceiveCooperativeSettleRequest) ClassId() int {
	return ReceiveCooperativeSettleRequestClassId
}

func (self ReceiveCooperativeSettle) ClassId() int {
	return ReceiveCooperativeSettleClassId
}

func (self ContractReceiveChannelCooperativeSettled) ClassId() int {
	return ContractReceiveChannelCooperativeSettledClassId
}

// class meta for Event
func (self SendMessageEvent) ClassId() int {
	return SendMessageEventClassId
}

func (self ContractSendEvent) ClassId() int {
	return ContractSendEventClassId
}

func (self ContractSendExpireAbleEvent) ClassId() int {
	return ContractSendExpireAbleEventClassId
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

func (self SendLockedTransfer) ClassId() int {
	return SendLockedTransferClassId
}

func (self SendLockExpired) ClassId() int {
	return SendLockExpiredClassId
}

func (self SendBalanceProof) ClassId() int {
	return SendBalanceProofClassId
}

func (self SendSecretRequest) ClassId() int {
	return SendSecretRequestClassId
}

func (self SendSecretReveal) ClassId() int {
	return SendSecretRevealClassId
}

func (self SendRefundTransfer) ClassId() int {
	return SendRefundTransferClassId
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

func (self EventUnlockSuccess) ClassId() int {
	return EventUnlockSuccessClassId
}

func (self EventInvalidReceivedTransferRefund) ClassId() int {
	return EventInvalidReceivedTransferRefundClassId
}

func (self EventInvalidReceivedLockExpired) ClassId() int {
	return EventInvalidReceivedLockExpiredClassId
}

func (self EventInvalidReceivedLockedTransfer) ClassId() int {
	return EventInvalidReceivedLockedTransferClassId
}

func (self EventUnlockFailed) ClassId() int {
	return EventUnlockFailedClassId
}

func (self EventUnlockClaimFailed) ClassId() int {
	return EventUnlockClaimFailedClassId
}

func (self EventInvalidReceivedUnlock) ClassId() int {
	return EventInvalidReceivedUnlockClassId
}

func (self EventUnlockClaimSuccess) ClassId() int {
	return EventUnlockClaimSuccessClassId
}

func (self EventUnexpectedSecretReveal) ClassId() int {
	return EventUnexpectedSecretRevealClassId
}

func (self SendWithdrawRequest) ClassId() int {
	return SendWithdrawRequestClassId
}

func (self SendWithdraw) ClassId() int {
	return SendWithdrawClassId
}

func (self ContractSendChannelWithdraw) ClassId() int {
	return ContractSendChannelWithdrawClassId
}

func (self EventWithdrawRequestSentFailed) ClassId() int {
	return EventWithdrawRequestSentFailedClassId
}

func (self EventInvalidReceivedWithdrawRequest) ClassId() int {
	return EventInvalidReceivedWithdrawRequestClassId
}

func (self EventInvalidReceivedWithdraw) ClassId() int {
	return EventInvalidReceivedWithdrawClassId
}

func (self EventWithdrawRequestTimeout) ClassId() int {
	return EventWithdrawRequestTimeoutClassId
}

func (self SendCooperativeSettleRequest) ClassId() int {
	return SendCooperativeSettleRequestClassId
}

func (self SendCooperativeSettle) ClassId() int {
	return SendCooperativeSettleClassId
}

func (self ContractSendChannelCooperativeSettle) ClassId() int {
	return ContractSendChannelCooperativeSettleClassId
}

func (self EventCooperativeSettleRequestSentFailed) ClassId() int {
	return EventCooperativeSettleRequestSentFailedClassId
}

func (self EventInvalidReceivedCooperativeSettleRequest) ClassId() int {
	return EventInvalidReceivedCooperativeSettleRequestClassId
}

func (self EventInvalidReceivedCooperativeSettle) ClassId() int {
	return EventInvalidReceivedCooperativeSettleClassId
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
	case InitiatorTransferStateClassId:
		result = new(InitiatorTransferState)
	case InitiatorPaymentStateClassId:
		result = new(InitiatorPaymentState)
	case MediatorTransferStateClassId:
		result = new(MediatorTransferState)
	case TargetTransferStateClassId:
		result = new(TargetTransferState)
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
	case ActionInitInitiatorClassId:
		result = new(ActionInitInitiator)
	case ActionInitMediatorClassId:
		result = new(ActionInitMediator)
	case ActionInitTargetClassId:
		result = new(ActionInitTarget)
	case ReceiveSecretRequestClassId:
		result = new(ReceiveSecretRequest)
	case ReceiveSecretRevealClassId:
		result = new(ReceiveSecretReveal)
	case ReceiveLockExpiredClassId:
		result = new(ReceiveLockExpired)
	case ReceiveTransferRefundCancelRouteClassId:
		result = new(ReceiveTransferRefundCancelRoute)
	case ReceiveTransferRefundClassId:
		result = new(ReceiveTransferRefund)
	case ActionWithdrawClassId:
		result = new(ActionWithdraw)
	case ReceiveWithdrawRequestClassId:
		result = new(ReceiveWithdrawRequest)
	case ReceiveWithdrawClassId:
		result = new(ReceiveWithdraw)
	case ContractReceiveChannelWithdrawClassId:
		result = new(ContractReceiveChannelWithdraw)
	case ActionCooperativeSettleClassId:
		result = new(ActionCooperativeSettle)
	case ReceiveCooperativeSettleRequestClassId:
		result = new(ReceiveCooperativeSettleRequest)
	case ReceiveCooperativeSettleClassId:
		result = new(ReceiveCooperativeSettle)
	case ContractReceiveChannelCooperativeSettledClassId:
		result = new(ContractReceiveChannelCooperativeSettled)
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
	case ContractSendExpireAbleEventClassId:
		result = new(ContractSendExpireAbleEvent)
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
	case SendLockedTransferClassId:
		result = new(SendLockedTransfer)
	case SendLockExpiredClassId:
		result = new(SendLockExpired)
	case SendBalanceProofClassId:
		result = new(SendBalanceProof)
	case SendSecretRevealClassId:
		result = new(SendSecretReveal)
	case SendSecretRequestClassId:
		result = new(SendSecretRequest)
	case SendRefundTransferClassId:
		result = new(SendRefundTransfer)
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
	case EventUnlockSuccessClassId:
		return new(EventUnlockSuccess)
	case EventInvalidReceivedTransferRefundClassId:
		return new(EventInvalidReceivedTransferRefund)
	case EventInvalidReceivedLockExpiredClassId:
		return new(EventInvalidReceivedLockExpired)
	case EventInvalidReceivedLockedTransferClassId:
		return new(EventInvalidReceivedLockedTransfer)
	case EventUnlockFailedClassId:
		return new(EventUnlockFailed)
	case EventUnlockClaimFailedClassId:
		return new(EventUnlockClaimFailed)
	case EventInvalidReceivedUnlockClassId:
		return new(EventInvalidReceivedUnlock)
	case EventUnlockClaimSuccessClassId:
		return new(EventUnlockClaimSuccess)
	case EventUnexpectedSecretRevealClassId:
		return new(EventUnexpectedSecretReveal)
	case SendWithdrawRequestClassId:
		return new(SendWithdrawRequest)
	case SendWithdrawClassId:
		return new(SendWithdraw)
	case ContractSendChannelWithdrawClassId:
		return new(ContractSendChannelWithdraw)
	case EventWithdrawRequestSentFailedClassId:
		return new(EventWithdrawRequestSentFailed)
	case EventInvalidReceivedWithdrawRequestClassId:
		return new(EventInvalidReceivedWithdrawRequest)
	case EventInvalidReceivedWithdrawClassId:
		return new(EventInvalidReceivedWithdraw)
	case EventWithdrawRequestTimeoutClassId:
		return new(EventWithdrawRequestTimeout)
	case SendCooperativeSettleRequestClassId:
		return new(SendCooperativeSettleRequest)
	case SendCooperativeSettleClassId:
		return new(SendCooperativeSettle)
	case ContractSendChannelCooperativeSettleClassId:
		return new(ContractSendChannelCooperativeSettle)
	case EventCooperativeSettleRequestSentFailedClassId:
		return new(EventCooperativeSettleRequestSentFailed)
	case EventInvalidReceivedCooperativeSettleRequestClassId:
		return new(EventInvalidReceivedCooperativeSettleRequest)
	case EventInvalidReceivedCooperativeSettleClassId:
		return new(EventInvalidReceivedCooperativeSettle)
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
