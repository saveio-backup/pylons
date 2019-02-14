package channelservice

import (
	"strconv"

	"github.com/oniio/oniChain-go-sdk/ong"
	"github.com/oniio/oniChain/common/log"
	sc_utils "github.com/oniio/oniChain/smartcontract/service/native/utils"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network/proxies"
	"github.com/oniio/oniChannel/transfer"
)

//NOTE, Event here come from blockchain filter
//Not the Event from transfer dir!
func (self ChannelService) HandleChannelNew(event map[string]interface{}) {

	var transactionHash common.TransactionHash

	var isParticipant bool

	participant1 := event["participant1"].(common.Address)
	participant2 := event["participant2"].(common.Address)
	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)

	if common.AddressEqual(self.address, participant1) || self.address == participant2 {
		isParticipant = true
	}

	tokenNetworkIdentifier := common.TokenNetworkID(ong.ONG_CONTRACT_ADDRESS)
	if isParticipant {

		channelProxy := self.chain.PaymentChannel(common.Address(tokenNetworkIdentifier), channelIdentifier, event)
		var revealTimeout common.BlockHeight
		if _, exist := self.config["reveal_timeout"]; exist == false {
			revealTimeout = common.BlockHeight(constants.DEFAULT_REVEAL_TIMEOUT)
		} else {
			rt := self.config["reveal_timeout"]
			if ret, err := strconv.Atoi(rt); err != nil {
				log.Warn("reveal timeout invalid in channel config %s, use default value %d", rt, constants.DEFAULT_REVEAL_TIMEOUT)
				revealTimeout = common.BlockHeight(constants.DEFAULT_REVEAL_TIMEOUT)
			} else {
				revealTimeout = common.BlockHeight(ret)
			}

		}
		tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
		defaultRegister := common.PaymentNetworkID(sc_utils.MicroPayContractAddress)
		channelState := SetupChannelState(tokenAddress, defaultRegister,
			common.TokenNetworkAddress(tokenNetworkIdentifier), revealTimeout, channelProxy, blockNumber)

		newChannel := &transfer.ContractReceiveChannelNew{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			tokenNetworkIdentifier, channelState, channelIdentifier}

		self.HandleStateChange(newChannel)

		//register partner address in UDPTransport!
		partnerAddress := channelState.PartnerState.Address
		self.transport.StartHealthCheck(partnerAddress)

	} else {
		//[TODO] generate ContractReceiveRouteNew when supporting route
	}

	return
}

func (self ChannelService) HandleChannelNewBalance(event map[string]interface{}) {

	var transactionHash common.TransactionHash
	var isParticipant bool

	participantAddress := event["participant"].(common.Address)
	channelIdentifier := event["channelID"].(common.ChannelID)
	depositBlockHeight := event["blockHeight"].(common.BlockHeight)
	totalDeposit := event["totalDeposit"].(common.TokenAmount)

	tokenNetworkIdentifier := common.TokenNetworkID(ong.ONG_CONTRACT_ADDRESS)

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

	tokenNetworkIdentifier := common.TokenNetworkID{}

	var channelIdentifier common.ChannelID
	var transactionHash common.TransactionHash
	var blockNumber common.BlockHeight
	var closingParticipant common.Address

	closingParticipant = event["closingParticipant"].(common.Address)
	channelIdentifier = event["channelID"].(common.ChannelID)
	blockNumber = event["blockHeight"].(common.BlockHeight)

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

	var transactionHash common.TransactionHash

	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)
	nonce := event["nonce"].(common.Nonce)

	chainState := self.StateFromChannel()
	tokenNetworkIdentifier := common.TokenNetworkID{}
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
	tokenNetworkIdentifier := common.TokenNetworkID{}

	var transactionHash common.TransactionHash

	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)

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
	log.Info(events)
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
func SetupChannelState(tokenAddress common.TokenAddress, paymentNetworkIdentifier common.PaymentNetworkID,
	tokenNetworkAddress common.TokenNetworkAddress, revealTimeout common.BlockHeight,
	paymentChannelProxy *proxies.PaymentChannel, openedBlockHeight common.BlockHeight) *transfer.NettingChannelState {

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
		ChainId:                  channelDetails.ChainId,
		TokenAddress:             common.Address(tokenAddress),
		PaymentNetworkIdentifier: paymentNetworkIdentifier,
		TokenNetworkIdentifier:   common.TokenNetworkID(tokenNetworkAddress),
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
			var address common.Address

			for index, data := range value.([]interface{}) {
				value := data.(float64)
				address[index] = byte(value)
			}

			events[item] = address
		case "channelID":
			events[item] = common.ChannelID(value.(float64))
		case "blockHeight":
			fallthrough
		case "settleTimeout":
			events[item] = common.BlockHeight(value.(float64))
		case "totalDeposit":
			events[item] = common.TokenAmount(value.(float64))
		case "nonce":
			events[item] = common.Nonce(value.(float64))
		}
	}
	return events
}
