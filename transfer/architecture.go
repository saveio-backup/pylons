package transfer

import (
	"container/list"

	"github.com/oniio/oniChannel/common"
)

type State interface {
	ClassId() int
}

type StateChange interface {
	ClassId() int
}

type Event interface {
	ClassId() int
}

type SendMessageEvent struct {
	Recipient         common.Address
	ChannelIdentifier common.ChannelID
	MessageIdentifier common.MessageID
}

type AuthenticatedSenderStateChange struct {
	Sender common.Address
}

type ContractSendEvent struct {
}

type ContractSendExpirableEvent struct {
	ContractSendEvent
	Expiration common.BlockExpiration
}

type ContractReceiveStateChange struct {
	TransactionHash common.TransactionHash
	BlockHeight     common.BlockHeight
}

type TransitionResult struct {
	NewState State
	Events   *list.List
}

type StateTransitionCallback func(chainState State, stateChange StateChange) TransitionResult

type StateManager struct {
	StateTransition StateTransitionCallback
	CurrentState    State
}

func (self *StateManager) Dispatch(stateChange StateChange) *list.List {
	var nextState State
	var iteration TransitionResult

	nextState = DeepCopy(self.CurrentState)
	iteration = self.StateTransition(nextState, stateChange)
	self.CurrentState = iteration.NewState
	events := iteration.Events

	return events
}

func (self *StateManager) DeepCopy() State {
	return DeepCopy(self.CurrentState)
}
