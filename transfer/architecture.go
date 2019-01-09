package transfer

import (
	"container/list"

	"github.com/oniio/oniChannel/typing"
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
	Recipient         typing.Address
	ChannelIdentifier typing.ChannelID
	MessageIdentifier typing.MessageID
}

type AuthenticatedSenderStateChange struct {
	Sender typing.Address
}

type ContractSendEvent struct {
}

type ContractSendExpirableEvent struct {
	ContractSendEvent
	Expiration typing.BlockExpiration
}

type ContractReceiveStateChange struct {
	TransactionHash typing.TransactionHash
	BlockHeight     typing.BlockHeight
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
