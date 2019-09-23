package service

import (
	"time"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network"
	"github.com/saveio/themis/common/log"
)

type AlarmTask struct {
	callbacks []AlarmTaskCallback
	chain     *network.BlockChainService
	chainId   int
	stopEvent chan int
	interval  int
}

type AlarmTaskCallback func()

func NewAlarmTask(chain *network.BlockChainService) *AlarmTask {
	self := new(AlarmTask)

	self.chain = chain
	if id, err := chain.ChainClient.GetNetworkId(); err != nil {
		log.Error("get network id failed, set chain id = 0")
		self.chainId = 0
	} else {
		self.chainId = int(id)
	}

	self.interval = common.Config.AlarmInterval
	self.stopEvent = make(chan int)

	return self
}

func (self *AlarmTask) Start() {
	go self.LoopUntilStop()
}

func (self *AlarmTask) RegisterCallback(callback AlarmTaskCallback) {
	self.callbacks = append(self.callbacks, callback)
}

func (self *AlarmTask) RemoveCallback(callback AlarmTaskCallback) {
	return
}

func (self *AlarmTask) LoopUntilStop() {
	interval := self.interval
	for {
		select {
		case <-self.stopEvent:
			return
		case <-time.After(time.Duration(interval) * time.Millisecond):
			self.runCallbacks()
		}
	}
}

func (self *AlarmTask) FirstRun() error {
	self.runCallbacks()
	return nil
}

func (self *AlarmTask) runCallbacks() {
	for _, f := range self.callbacks {
		f()
	}
}

func (self *AlarmTask) Stop() {
	self.stopEvent <- 0
}

func (self *AlarmTask) GetInterval() float32 {
	return float32(self.interval)
}
