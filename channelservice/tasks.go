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
	lastBlockNumber typing.BlockNumber
	stopEvent       chan int
	sleepTime       int
}

type AlarmTaskCallback func(blockNumber typing.BlockNumber, blockHash typing.BlockHash)

func NewAlarmTask(chain *network.BlockchainService) *AlarmTask {
	self := new(AlarmTask)

	self.chain = chain
	self.chainId = 0
	self.lastBlockNumber = 0
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
	var lastBlockNumber, latestBlockNumber typing.BlockNumber
	var blockHash typing.BlockHash
	var err error
	sleepTime := self.sleepTime

	for {
		select {
		case <-self.stopEvent:
			break
		case <-time.After(time.Duration(sleepTime) * time.Millisecond):
			lastBlockNumber = self.lastBlockNumber
			//[TODO] use BlockchainService.GetBlock to get latestBlockNumber
			//and blockHash
			latestBlockNumber, blockHash, err = self.GetLatestBlock()
			if err != nil {
				fmt.Println(err)
				continue
			}

			if latestBlockNumber != lastBlockNumber {
				if latestBlockNumber > lastBlockNumber+1 {
					fmt.Printf("Missing block(s), latest Block number %d, last Block number %d\n", latestBlockNumber, lastBlockNumber)
				}
				self.runCallbacks(latestBlockNumber, blockHash)
			}
		}
	}
}

func (self *AlarmTask) GetLatestBlock() (typing.BlockNumber, typing.BlockHash, error) {
	blockNumber, err := self.chain.BlockNumber()
	latestBlockNumber := typing.BlockNumber(blockNumber)
	if err != nil {
		return 0, nil, fmt.Errorf("GetBlockNumber error")
	}

	header, _ := self.chain.GetBlock(blockNumber)
	blockHash := header.Hash()

	return latestBlockNumber, blockHash[:], nil
}

func (self *AlarmTask) FirstRun() {
	var latestBlock typing.BlockNumber
	var blockHash typing.BlockHash

	//[TODO] use BlockchainService.GetBlock to get latestBlockNumber
	//and blockHash
	latestBlock, blockHash, _ = self.GetLatestBlock()

	self.runCallbacks(latestBlock, blockHash)
	return
}

func (self *AlarmTask) runCallbacks(latestBlockNumber typing.BlockNumber, blockHash typing.BlockHash) {

	fmt.Printf("RunCallbacks for Block %d\n", latestBlockNumber)

	for _, f := range self.callbacks {
		f(latestBlockNumber, blockHash)
	}

	self.lastBlockNumber = latestBlockNumber
}

func (self *AlarmTask) Stop() {
	self.stopEvent <- 0
}

func (self *AlarmTask) GetSleepTime() float32 {
	return float32(self.sleepTime)
}
