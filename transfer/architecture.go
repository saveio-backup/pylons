package transfer

import (
	"github.com/saveio/pylons/common"
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
	Recipient common.Address
	ChannelId common.ChannelID
	MessageId common.MessageID
}

type AuthenticatedSenderStateChange struct {
	Sender common.Address
}

type ContractSendEvent struct {
}

type ContractSendExpireAbleEvent struct {
	ContractSendEvent
	Expiration common.BlockExpiration
}

type ContractReceiveStateChange struct {
	TransactionHash common.TransactionHash
	BlockHeight     common.BlockHeight
}

type TransitionResult struct {
	NewState State
	Events   []Event
}

type StateTransitionCallback func(chainState State, stateChange StateChange) *TransitionResult

type StateManager struct {
	StateTransition StateTransitionCallback
	CurrentState    State
}

func (self *StateManager) Dispatch(stateChange StateChange) []Event {
	nextState := DeepCopy(self.CurrentState)
	iteration := self.StateTransition(nextState, stateChange)
	self.CurrentState = iteration.NewState
	events := iteration.Events

	return events
}

func (self *StateManager) DeepCopy() State {
	return DeepCopy(self.CurrentState)
}
