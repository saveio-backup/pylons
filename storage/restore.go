package storage

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/common/log"
)

func ChannelStateUntilStateChange(
	storage *SQLiteStorage, paymentNetworkIdentifier common.PaymentNetworkID, tokenAddress common.TokenAddress,
	channelIdentifier common.ChannelID, stateChangeIdentifier int, address common.Address) *transfer.NettingChannelState {

	var chainState *transfer.ChainState

	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)
	wal := RestoreToStateChange(transfer.StateTransition, storage, stateChangeIdentifier, address)

	state := wal.StateManager.CurrentState
	if state != nil {
		chainState, _ = state.(*transfer.ChainState)
	}

	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState, tokenNetworkIdentifier, channelIdentifier)
	if channelState == nil {
		log.Errorf("Channel was not found before state_change %d", stateChangeIdentifier)
		return nil
	}

	return channelState
}
