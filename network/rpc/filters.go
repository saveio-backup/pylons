package rpc

import (
	"encoding/json"
	"errors"

	"fmt"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/network/utils"
	"github.com/oniio/oniChannel/typing"
)

func (this *RpcClient) GetFilterArgsForAllEventsFromChannel(chanID typing.ChannelID, fromBlock, toBlock typing.BlockHeight) ([]map[string]interface{}, error) {
	toBlockUint := uint32(toBlock)

	currentHeight, _ := this.GetCurrentBlockHeight()
	currentH, _ := utils.GetUint32(currentHeight)
	if toBlockUint > currentH {
		return nil, fmt.Errorf("toBlock bigger than currentBlockHeight:%d", currentH)
	} else if toBlockUint == 0 {
		toBlockUint = currentH
	} else if uint32(fromBlock) > toBlockUint {
		return nil, errors.New("fromBlock bigger than toBlock")
	}

	var eventRe = make([]map[string]interface{}, 0)
	for bc := uint32(fromBlock); bc <= toBlockUint; bc++ {
		rawResult, err := this.GetSmartContractEventByBlock(bc)
		if err != nil {
			log.Errorf("get smart contract result by block err, msg:%s", err)
			return nil, err
		}
		if rawResult == nil {
			log.Errorf("rawResult is nil")
			continue
		}
		var raws []interface{}
		err = json.Unmarshal(rawResult, &raws)
		if err != nil {
			log.Errorf("json unmarshal result err:%s", err)
			return nil, err
		}
		if len(raws) == 0 {
			continue
		}
		for _, r := range raws {
			buf, err := json.Marshal(r)
			if err != nil {
				log.Errorf("json marshal result err:%s", err)
				return nil, err
			}
			result, err := utils.GetSmartContractEvent(buf)
			if err != nil {
				log.Errorf("GetSmartContractEvent[rawResult Unmarshal] err: %s", err)
				return nil, err
			}
			if result == nil {
				log.Errorf("rawResult Unmarshal return nil")
				continue
			}
			for _, notify := range result.Notify {
				if _, ok := notify.States.(map[string]interface{}); !ok {
					continue
				}
				eventRe = append(eventRe, notify.States.(map[string]interface{}))
			}
		}
	}
	return eventRe, nil
}

// GetAllFilterArgsForAllEventsFromChannel get all events from fromBlock to current block height
// return a slice of map[string]interface{}
func (this *RpcClient) GetAllFilterArgsForAllEventsFromChannel(chanID typing.ChannelID, fromBlock typing.BlockHeight) ([]map[string]interface{}, error) {
	heightBytes, err := this.GetCurrentBlockHeight()
	if err != nil {
		return nil, err
	}
	//height, err := strconv.ParseUint(string(heightBytes), 10, 64)
	height, err := utils.GetUint32(heightBytes)
	if err != nil {
		return nil, err
	}
	toBlockUint := typing.BlockHeight(height)
	return this.GetFilterArgsForAllEventsFromChannel(chanID, fromBlock, toBlockUint)
}
