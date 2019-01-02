package blockchain

import (
	"github.com/oniio/oniChannel/network/proxies"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

//Only called by new opened channel
func GetChannelState(tokenAddress typing.TokenAddress, paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenNetworkAddress typing.TokenNetworkAddress, revealTimeout typing.BlockNumber,
	paymentChannelProxy *proxies.PaymentChannel, openedBlockNumber typing.BlockNumber) *transfer.NettingChannelState {

	channelDetails := paymentChannelProxy.Detail()
	ourState := transfer.NewNettingChannelEndState()
	ourState.Address = channelDetails.ParticipantsData.OurDetails.Address
	ourState.ContractBalance = channelDetails.ParticipantsData.OurDetails.Deposit

	partnerState := transfer.NewNettingChannelEndState()
	partnerState.Address = channelDetails.ParticipantsData.PartnerDetails.Address
	partnerState.ContractBalance = channelDetails.ParticipantsData.PartnerDetails.Deposit

	identifier := paymentChannelProxy.GetChannelId()
	settleTimeout := paymentChannelProxy.SettleTimeout()

	if openedBlockNumber <= 0 {
		return nil
	}

	openTransaction := &transfer.TransactionExecutionStatus{
		0, openedBlockNumber, transfer.TransactionExecutionStatusSuccess}

	channel := &transfer.NettingChannelState{
		Identifier:               identifier,
		ChainId:                  0,
		TokenAddress:             typing.Address(tokenAddress),
		PaymentNetworkIdentifier: paymentNetworkIdentifier,
		TokenNetworkIdentifier:   typing.TokenNetworkID(tokenNetworkAddress),
		RevealTimeout:            revealTimeout,
		SettleTimeout:            settleTimeout,
		OurState:                 ourState,
		PartnerState:             partnerState,
		OpenTransaction:          openTransaction}

	return channel
}
