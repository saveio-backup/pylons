package storage

import (
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

func ChannelStateUntilStateChange(storage *SQLiteStorage, paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, channelIdentifier typing.ChannelID,
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
