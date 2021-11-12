package storage

import (
	"reflect"
	"sync"
	"time"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

func RestoreToStateChange(transitionFunction transfer.StateTransitionCallback,
	storage *SQLiteStorage, stateChangeIdentifier interface{}, selfAddr common.Address) *WriteAheadLog {

	fromStateChangeId, snapshot := storage.getSnapshotClosestToStateChange(stateChangeIdentifier)

	if snapshot == nil {
		log.Info("No snapshot found, replaying all state changes")
	}

	unAppliedStateChanges := storage.getStateChangesById(fromStateChangeId, stateChangeIdentifier)

	var ok bool
	var chainState *transfer.ChainState
	var stateManager *transfer.StateManager

	if chainState, ok = snapshot.(*transfer.ChainState); ok {
		chainState.AdjustChainState()
		stateManager = &transfer.StateManager{StateTransition: transitionFunction, CurrentState: chainState}
	} else {
		stateManager = &transfer.StateManager{StateTransition: transitionFunction, CurrentState: nil}
	}

	wal := new(WriteAheadLog)
	wal.StateManager = stateManager
	wal.Storage = storage

	//qLen := 0
	//if chainState != nil {
	//	log.Info("[QueueIdsToQueues] qNum = ", len(chainState.QueueIdsToQueues))
	//	for _, v :=  range chainState.QueueIdsToQueues {
	//		qLen += len(v)
	//	}
	//	log.Info("[QueueIdsToQueues] qLen = ", qLen)
	//}

	i := 1
	for _, change := range unAppliedStateChanges {
		//log.Info("[RestoreToStateChange] stateChangeId: ", fromStateChangeId + i)
		common.SetRandSeed(fromStateChangeId+i, selfAddr)
		if change == nil {
			log.Warn("restore state change is null")
			continue
		}
		wal.StateManager.Dispatch(change)
		i++
	}
	//if wal.StateManager.CurrentState != nil {
	//	log.Info("[QueueIdsToQueues] qNum = ", len(wal.StateManager.CurrentState.(*transfer.ChainState).QueueIdsToQueues))
	//	for _, v :=  range wal.StateManager.CurrentState.(*transfer.ChainState).QueueIdsToQueues {
	//		qLen += len(v)
	//	}
	//	log.Info("[QueueIdsToQueues] qLen = ", qLen)
	//}

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

func (self *WriteAheadLog) LogAndDispatch(stateChange transfer.StateChange, selfAddr common.Address) []transfer.Event {

	self.dbLock.Lock()
	defer self.dbLock.Unlock()

	self.Storage.writeStateChange(stateChange, &self.StateChangeId)
	//log.Info("[LogAndDispatch] stateChangeId: ", self.StateChangeId)
	common.SetRandSeed(self.StateChangeId, selfAddr)
	events := self.StateManager.Dispatch(stateChange)

	t := time.Now()
	timestamp := t.UTC().String()
	self.Storage.writeEvents(self.StateChangeId, events, timestamp)

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
