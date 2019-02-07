package storage

import (
	"container/list"
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

	unappliedStateChanges := storage.getStateChangesByIdentifier(
		fromStateChangeId, stateChangeIdentifier)

	if chainState, ok := snapshot.(*transfer.ChainState); ok {
		chainState.AdjustChainState()
	}
	stateManager := &transfer.StateManager{transitionFunction, snapshot}

	wal := new(WriteAheadLog)
	wal.StateManager = stateManager
	wal.Storage = storage

	for e := unappliedStateChanges.Front(); e != nil; e = e.Next() {
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

func (self *WriteAheadLog) LogAndDispatch(stateChange transfer.StateChange) *list.List {

	self.dbLock.Lock()
	defer self.dbLock.Unlock()

	stateChangeId := self.Storage.writeStateChange(stateChange)
	self.StateChangeId = stateChangeId

	events := self.StateManager.Dispatch(stateChange)

	t := time.Now()
	timestamp := t.UTC().String()
	self.Storage.writeEvents(stateChangeId, events, timestamp)

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
