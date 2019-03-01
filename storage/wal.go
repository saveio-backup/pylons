package storage

import (
	"reflect"
	"sync"
	"time"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/transfer"
)

func RestoreToStateChange(transitionFunction transfer.StateTransitionCallback,
	storage *SQLiteStorage, stateChangeIdentifier interface{}) *WriteAheadLog {

	fromStateChangeId, snapshot := storage.getSnapshotClosestToStateChange(stateChangeIdentifier)

	if snapshot == nil {
		log.Info("No snapshot found, replaying all state changes")
	}

	unAppliedStateChanges := storage.getStateChangesByIdentifier(
		fromStateChangeId, stateChangeIdentifier)

	if chainState, ok := snapshot.(*transfer.ChainState); ok {
		chainState.AdjustChainState()
	}
	var stateManager *transfer.StateManager
	if chainState, ok := snapshot.(*transfer.ChainState); ok {
		chainState.AdjustChainState()
		stateManager = &transfer.StateManager{StateTransition:transitionFunction, CurrentState: chainState}
	} else {
		stateManager = &transfer.StateManager{StateTransition:transitionFunction, CurrentState: nil}
	}

	wal := new(WriteAheadLog)
	wal.StateManager = stateManager
	wal.Storage = storage

	for e := unAppliedStateChanges.Front(); e != nil; e = e.Next() {
		wal.StateManager.Dispatch(e.Value.(transfer.StateChange))
	}

	return wal
}

type WriteAheadLog struct {
	StateManager  *transfer.StateManager
	StateChangeId int
	Storage       *SQLiteStorage
	dbLock        sync.Mutex
}

func (self *WriteAheadLog) DeepCopy() *transfer.ChainState {
	self.dbLock.Lock()
	defer self.dbLock.Unlock()

	result := self.StateManager.DeepCopy()
	if reflect.ValueOf(result).IsNil() {
		return nil
	} else {
		return result.(*transfer.ChainState)
	}

	return nil
}

func (self *WriteAheadLog) LogAndDispatch(stateChange transfer.StateChange) []transfer.Event {

	self.dbLock.Lock()
	defer self.dbLock.Unlock()
	self.Storage.StateSync.Wait()
	self.Storage.writeStateChange(stateChange, &self.StateChangeId)
	log.Info("[LogAndDispatch] ", reflect.TypeOf(stateChange).String())
	events := self.StateManager.Dispatch(stateChange)

	self.Storage.EventSync.Wait()
	t := time.Now()
	timestamp := t.UTC().String()
	go self.Storage.writeEvents(self.StateChangeId, events, timestamp)

	return events
}

func (self *WriteAheadLog) Snapshot() {
	self.dbLock.Lock()
	defer self.dbLock.Unlock()

	currentState := self.StateManager.CurrentState
	stateChangeId := self.StateChangeId

	if stateChangeId != 0 {
		self.Storage.writeStateSnapshot(stateChangeId, currentState)
	}

	return
}

func (self *WriteAheadLog) Version() int {
	return self.Storage.getVersion()
}
