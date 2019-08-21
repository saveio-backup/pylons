package channelservice

import (
	"errors"
	"time"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network"
	"github.com/saveio/themis/common/log"
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
	var channelBlockHeight, chainBlockHeight common.BlockHeight
	var blockHash common.BlockHash
	var err error
	interval := self.interval

	for {
		select {
		case <-self.stopEvent:
			return
		case <-time.After(time.Duration(interval) * time.Millisecond):
			channelBlockHeight = self.lastBlockHeight
			chainBlockHeight, blockHash, err = self.GetLatestBlock()
			if err != nil {
				log.Error(err)
				continue
			}

			if chainBlockHeight > channelBlockHeight {
				if chainBlockHeight > channelBlockHeight+1 {
					log.Infof("Missing block(s), Chain Block Height %d, channel Block Height %d", chainBlockHeight, channelBlockHeight)
				}
				self.runCallbacks(chainBlockHeight, blockHash)
			}
		}
	}
}

func (self *AlarmTask) GetLatestBlock() (common.BlockHeight, common.BlockHash, error) {
	blockNumber, err := self.chain.BlockHeight()
	if err != nil {
		return 0, nil, errors.New("get chain block height error")
	}

	blockHash, err := self.chain.GetBlockHash(blockNumber)
	if err != nil {
		return 0, nil, errors.New("get block hash error")
	}
	return common.BlockHeight(blockNumber), blockHash, nil
}

func (self *AlarmTask) FirstRun() error {
	var latestBlock common.BlockHeight
	var blockHash common.BlockHash

	//[TODO] use BlockChainService.GetBlock to get latestBlockHeight
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

	log.Infof("process block %d", latestBlockHeight)

	for _, f := range self.callbacks {
		f(latestBlockHeight, blockHash)
	}

	self.lastBlockHeight = latestBlockHeight
}

func (self *AlarmTask) Stop() {
	self.stopEvent <- 0
}

func (self *AlarmTask) GetInterval() float32 {
	return float32(self.interval)
}
