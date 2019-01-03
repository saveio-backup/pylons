package channelservice

import (
	"fmt"
	"time"

	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/typing"
)

type AlarmTask struct {
	callbacks       []AlarmTaskCallback
	chain           *network.BlockchainService
	chainId         int
	lastBlockHeight typing.BlockHeight
	stopEvent       chan int
	sleepTime       int
}

type AlarmTaskCallback func(blockNumber typing.BlockHeight, blockHash typing.BlockHash)

func NewAlarmTask(chain *network.BlockchainService) *AlarmTask {
	self := new(AlarmTask)

	self.chain = chain
	self.chainId = 0
	self.lastBlockHeight = 0
	self.sleepTime = 500
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
	var lastBlockHeight, latestBlockHeight typing.BlockHeight
	var blockHash typing.BlockHash
	var err error
	sleepTime := self.sleepTime

	for {
		select {
		case <-self.stopEvent:
			break
		case <-time.After(time.Duration(sleepTime) * time.Millisecond):
			lastBlockHeight = self.lastBlockHeight
			//[TODO] use BlockchainService.GetBlock to get latestBlockHeight
			//and blockHash
			latestBlockHeight, blockHash, err = self.GetLatestBlock()
			if err != nil {
				fmt.Println(err)
				continue
			}

			if latestBlockHeight != lastBlockHeight {
				if latestBlockHeight > lastBlockHeight+1 {
					fmt.Printf("Missing block(s), latest Block number %d, last Block number %d\n", latestBlockHeight, lastBlockHeight)
				}
				self.runCallbacks(latestBlockHeight, blockHash)
			}
		}
	}
}

func (self *AlarmTask) GetLatestBlock() (typing.BlockHeight, typing.BlockHash, error) {
	blockNumber, err := self.chain.BlockHeight()
	latestBlockHeight := typing.BlockHeight(blockNumber)
	if err != nil {
		return 0, nil, fmt.Errorf("GetBlockHeight error")
	}

	header, _ := self.chain.GetBlock(blockNumber)
	blockHash := header.Hash()

	return latestBlockHeight, blockHash[:], nil
}

func (self *AlarmTask) FirstRun() {
	var latestBlock typing.BlockHeight
	var blockHash typing.BlockHash

	//[TODO] use BlockchainService.GetBlock to get latestBlockHeight
	//and blockHash
	latestBlock, blockHash, _ = self.GetLatestBlock()

	self.runCallbacks(latestBlock, blockHash)
	return
}

func (self *AlarmTask) runCallbacks(latestBlockHeight typing.BlockHeight, blockHash typing.BlockHash) {

	fmt.Printf("RunCallbacks for Block %d\n", latestBlockHeight)

	for _, f := range self.callbacks {
		f(latestBlockHeight, blockHash)
	}

	self.lastBlockHeight = latestBlockHeight
}

func (self *AlarmTask) Stop() {
	self.stopEvent <- 0
}

func (self *AlarmTask) GetSleepTime() float32 {
	return float32(self.sleepTime)
}
