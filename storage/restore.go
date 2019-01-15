package storage

import (
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

func ChannelStateUntilStateChange(storage *SQLiteStorage, paymentNetworkIdentifier common.PaymentNetworkID,
	tokenAddress common.TokenAddress, channelIdentifier common.ChannelID,
	stateChangeIdentifier interface{}) *transfer.NettingChannelState {

	wal := RestoreToStateChange(transfer.StateTransition, storage, stateChangeIdentifier)
	chainState, ok := wal.StateManager.CurrentState.(*transfer.ChainState)
	if ok == false {
		return nil
	}

	channelState := transfer.GetChannelStateById(chainState, paymentNetworkIdentifier,
		tokenAddress, channelIdentifier)

	return channelState
}
