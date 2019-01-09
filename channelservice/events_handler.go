package channelservice

import (
	"github.com/oniio/oniChannel/network/proxies"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

//NOTE, Event here come from blockchain filter
//Not the Event from transfer dir!
func (self ChannelService) HandleChannelNew(event map[string]interface{}) {

	var transactionHash typing.TransactionHash

	var isParticipant bool

	participant1 := event["participant1"].(typing.Address)
	participant2 := event["participant2"].(typing.Address)
	channelIdentifier := event["channelID"].(typing.ChannelID)
	blockNumber := event["blockHeight"].(typing.BlockHeight)

	if typing.AddressEqual(self.address, participant1) || self.address == participant2 {
		isParticipant = true
	}

	tokenNetworkIdentifier := typing.TokenNetworkID{}
	if isParticipant {

		channelProxy := self.chain.PaymentChannel(typing.Address(tokenNetworkIdentifier), channelIdentifier, event)

		//[TODO] get revealTime from channel.config[reveal_timeout]
		var revealTimeout typing.BlockHeight

		tokenAddress := typing.TokenAddress{}
		defaultRegister := typing.PaymentNetworkID{}
		channelState := GetChannelState(tokenAddress, defaultRegister,
			typing.TokenNetworkAddress(tokenNetworkIdentifier), revealTimeout, channelProxy, blockNumber)

		newChannel := &transfer.ContractReceiveChannelNew{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			tokenNetworkIdentifier, channelState, channelIdentifier}

		self.HandleStateChange(newChannel)

		//register partner address in UDPTransport!
		partnerAddress := channelState.PartnerState.Address
		self.StartHealthCheckFor(partnerAddress)

	} else {
		//[TODO] generate ContractReceiveRouteNew when supporting route
	}

	return
}

func (self ChannelService) HandleChannelNewBalance(event map[string]interface{}) {

	var transactionHash typing.TransactionHash
	var isParticipant bool

	participantAddress := event["participant"].(typing.Address)
	channelIdentifier := event["channelID"].(typing.ChannelID)
	depositBlockHeight := event["blockHeight"].(typing.BlockHeight)
	totalDeposit := event["totalDeposit"].(typing.TokenAmount)

	tokenNetworkIdentifier := typing.TokenNetworkID{}

	previousChannelState := transfer.GetChannelStateByTokenNetworkIdentifier(
		self.StateFromChannel(), tokenNetworkIdentifier, channelIdentifier)

	if previousChannelState != nil {
		isParticipant = true
	}

	if isParticipant {
		previousBalance := previousChannelState.OurState.ContractBalance

		depositTransaction := transfer.TransactionChannelNewBalance{
			participantAddress, totalDeposit, depositBlockHeight}

		newBalanceStateChange := &transfer.ContractReceiveChannelNewBalance{
			transfer.ContractReceiveStateChange{transactionHash, depositBlockHeight},
			tokenNetworkIdentifier, channelIdentifier, depositTransaction}

		self.HandleStateChange(newBalanceStateChange)

		if previousBalance == 0 && participantAddress != self.address {
			// if our deposit transaction is not confirmed and participant desopit event received ,
			// we should not deposite again, check the DepositTransactionQueue if we have deposit before
			chainState := self.StateFromChannel()
			channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState,
				tokenNetworkIdentifier, channelIdentifier)

			var found bool
			for _, v := range channelState.DepositTransactionQueue {
				if v.Transaction.ParticipantAddress == self.address && v.Transaction.ContractBalance != 0 {
					found = true
					break
				}
			}

			if !found {
				connectionManager := self.ConnectionManagerForTokenNetwork(tokenNetworkIdentifier)

				go connectionManager.JoinChannel(participantAddress, totalDeposit)
			}
		}

	}

	return
}

func (self ChannelService) HandleChannelClose(event map[string]interface{}) {

	tokenNetworkIdentifier := typing.TokenNetworkID{}

	var channelIdentifier typing.ChannelID
	var transactionHash typing.TransactionHash
	var blockNumber typing.BlockHeight
	var closingParticipant typing.Address

	closingParticipant = event["closingParticipant"].(typing.Address)
	channelIdentifier = event["channelID"].(typing.ChannelID)
	blockNumber = event["blockHeight"].(typing.BlockHeight)

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState,
		tokenNetworkIdentifier, channelIdentifier)

	if channelState != nil {
		channelClosed := &transfer.ContractReceiveChannelClosed{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			closingParticipant, tokenNetworkIdentifier, channelIdentifier}

		self.HandleStateChange(channelClosed)
	} else {
		//[TODO] generate ContractReceiveRouteClosed when supporting route
	}
}

func (self ChannelService) HandleChannelUpdateTransfer(event map[string]interface{}) {

	var transactionHash typing.TransactionHash

	channelIdentifier := event["channelID"].(typing.ChannelID)
	blockNumber := event["blockHeight"].(typing.BlockHeight)
	nonce := event["nonce"].(typing.Nonce)

	chainState := self.StateFromChannel()
	tokenNetworkIdentifier := typing.TokenNetworkID{}
	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState,
		tokenNetworkIdentifier, channelIdentifier)

	if channelState != nil {
		channelTransferUpdated := &transfer.ContractReceiveUpdateTransfer{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			tokenNetworkIdentifier, channelIdentifier, nonce}

		self.HandleStateChange(channelTransferUpdated)
	}

	return
}

func (self ChannelService) HandleChannelSettled(event map[string]interface{}) {
	tokenNetworkIdentifier := typing.TokenNetworkID{}

	var transactionHash typing.TransactionHash

	channelIdentifier := event["channelID"].(typing.ChannelID)
	blockNumber := event["blockHeight"].(typing.BlockHeight)

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState,
		tokenNetworkIdentifier, channelIdentifier)

	if channelState != nil {
		channelSettled := &transfer.ContractReceiveChannelSettled{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			tokenNetworkIdentifier, channelIdentifier}

		self.HandleStateChange(channelSettled)
	}

	return
}

func (self ChannelService) HandleChannelBatchUnlock(event map[string]interface{}) {
	return
}

func (self ChannelService) HandleSecretRevealed(event map[string]interface{}) {
	return
}

func OnBlockchainEvent(channel *ChannelService, event map[string]interface{}) {
	var eventName string

	if _, ok := event["eventName"].(string); ok == false {
		return
	}

	eventName = event["eventName"].(string)

	events := ParseEvent(event)

	if eventName == "chanOpened" {
		channel.HandleChannelNew(events)
	} else if eventName == "ChannelClose" {
		channel.HandleChannelClose(events)
	} else if eventName == "SetTotalDeposit" {
		channel.HandleChannelNewBalance(events)
	} else if eventName == "chanSettled" {
		channel.HandleChannelSettled(events)
	} else if eventName == "NonClosingBPFUpdate" {
		channel.HandleChannelUpdateTransfer(events)
	}

	return
}
func GetChannelState(tokenAddress typing.TokenAddress, paymentNetworkIdentifier typing.PaymentNetworkID,
	tokenNetworkAddress typing.TokenNetworkAddress, revealTimeout typing.BlockHeight,
	paymentChannelProxy *proxies.PaymentChannel, openedBlockHeight typing.BlockHeight) *transfer.NettingChannelState {

	channelDetails := paymentChannelProxy.Detail()
	ourState := transfer.NewNettingChannelEndState()
	ourState.Address = channelDetails.ParticipantsData.OurDetails.Address
	ourState.ContractBalance = channelDetails.ParticipantsData.OurDetails.Deposit

	partnerState := transfer.NewNettingChannelEndState()
	partnerState.Address = channelDetails.ParticipantsData.PartnerDetails.Address
	partnerState.ContractBalance = channelDetails.ParticipantsData.PartnerDetails.Deposit

	identifier := paymentChannelProxy.GetChannelId()
	settleTimeout := paymentChannelProxy.SettleTimeout()

	if openedBlockHeight <= 0 {
		return nil
	}

	openTransaction := &transfer.TransactionExecutionStatus{
		0, openedBlockHeight, transfer.TxnExecSucc}

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

func ParseEvent(event map[string]interface{}) map[string]interface{} {
	events := make(map[string]interface{})

	for item, value := range event {
		switch item {
		case "participant":
			fallthrough
		case "participant1":
			fallthrough
		case "participant2":
			fallthrough
		case "closingParticipant":
			var address typing.Address

			for index, data := range value.([]interface{}) {
				value := data.(float64)
				address[index] = byte(value)
			}

			events[item] = address
		case "channelID":
			events[item] = typing.ChannelID(value.(float64))
		case "blockHeight":
			fallthrough
		case "settleTimeout":
			events[item] = typing.BlockHeight(value.(float64))
		case "totalDeposit":
			events[item] = typing.TokenAmount(value.(float64))
		case "nonce":
			events[item] = typing.Nonce(value.(float64))
		}
	}
	return events
}
