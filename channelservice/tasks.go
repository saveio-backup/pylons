package channelservice

import (
	"fmt"
	"time"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network"
)

type AlarmTask struct {
	callbacks       []AlarmTaskCallback
	chain           *network.BlockchainService
	chainId         int
	lastBlockHeight common.BlockHeight
	stopEvent       chan int
	interval        int
}

type AlarmTaskCallback func(blockNumber common.BlockHeight, blockHash common.BlockHash)

func NewAlarmTask(chain *network.BlockchainService) *AlarmTask {
	self := new(AlarmTask)

	self.chain = chain
	if id, err := chain.ChainClient.GetNetworkId(); err != nil {
		log.Error("get network id failed, set chain id = 0")
		self.chainId = 0
	} else {
		self.chainId = int(id)
	}

	self.lastBlockHeight = 0
	self.interval = constants.ALARM_INTERVAL
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
	var lastBlockHeight, latestBlockHeight common.BlockHeight
	var blockHash common.BlockHash
	var err error
	interval := self.interval

	for {
		select {
		case <-self.stopEvent:
			break
		case <-time.After(time.Duration(interval) * time.Millisecond):
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

func (self *AlarmTask) GetLatestBlock() (common.BlockHeight, common.BlockHash, error) {
	blockNumber, err := self.chain.BlockHeight()
	latestBlockHeight := common.BlockHeight(blockNumber)
	if err != nil {
		return 0, nil, fmt.Errorf("GetBlockHeight error")
	}

	header, _ := self.chain.GetBlock(blockNumber)
	blockHash := header.Hash()

	return latestBlockHeight, blockHash[:], nil
}

func (self *AlarmTask) FirstRun() error {
	var latestBlock common.BlockHeight
	var blockHash common.BlockHash

	//[TODO] use BlockchainService.GetBlock to get latestBlockHeight
	//and blockHash
	latestBlock, blockHash, err := self.GetLatestBlock()
	if err != nil {
		log.Error("get latest block error ", err)
		return err
	}
	self.runCallbacks(latestBlock, blockHash)
	return nil
}

func (self *AlarmTask) runCallbacks(latestBlockHeight common.BlockHeight, blockHash common.BlockHash) {

	log.Infof("RunCallbacks for Block %d\n", latestBlockHeight)

	for _, f := range self.callbacks {
		f(latestBlockHeight, blockHash)
	}

	self.lastBlockHeight = latestBlockHeight
}

func (self *AlarmTask) Stop() {
	self.stopEvent <- 0
}

func (self *AlarmTask) Getinterval() float32 {
	return float32(self.interval)
}
