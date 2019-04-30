package channelservice

import (
	"strconv"

	"github.com/oniio/oniChain-go-sdk/usdt"
	"github.com/oniio/oniChain/common/log"
	scUtils "github.com/oniio/oniChain/smartcontract/service/native/utils"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network/proxies"
	"github.com/saveio/pylons/transfer"
)

//NOTE, Event here come from blockchain filter
//Not the Event from transfer dir!
func (self *ChannelService) HandleChannelNew(event map[string]interface{}) {
	var transactionHash common.TransactionHash
	log.Debug("[HandleChannelNew]")
	var isParticipant bool

	participant1 := event["participant1"].(common.Address)
	participant2 := event["participant2"].(common.Address)
	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)

	if common.AddressEqual(self.address, participant1) || common.AddressEqual(self.address, participant2) {
		isParticipant = true
	}
	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)
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
		tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
		defaultRegister := common.PaymentNetworkID(scUtils.MicroPayContractAddress)
		channelState := SetupChannelState(tokenAddress, defaultRegister,
			common.TokenNetworkAddress(tokenNetworkIdentifier), revealTimeout, channelProxy, blockNumber)

		newChannel := &transfer.ContractReceiveChannelNew{
			transfer.ContractReceiveStateChange{transactionHash, blockNumber},
			tokenNetworkIdentifier, channelState, channelIdentifier}

		self.HandleStateChange(newChannel)

		//register partner address in UDPTransport!
		partnerAddress := channelState.PartnerState.Address
		go self.NotifyNewChannel(channelIdentifier, partnerAddress)
		self.channelActor.Transport.StartHealthCheck(partnerAddress)

	} else {
		newRoute := &transfer.ContractReceiveRouteNew{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: transactionHash,
				BlockHeight:     blockNumber,
			},
			TokenNetworkIdentifier: tokenNetworkIdentifier,
			ChannelIdentifier:      channelIdentifier,
			Participant1:           participant1,
			Participant2:           participant2,
		}
		self.HandleStateChange(newRoute)
	}
	//todo
	//connectionManager := self.ConnectionManagerForTokenNetwork(tokenNetworkIdentifier)
	//retryConnect := gevent.spawn(connectionManager.RetryConnect)
	//self.AddPendingRoutine(retryConnect)
	return
}

func (self *ChannelService) handleChannelNewBalance(event map[string]interface{}) {
	log.Debug("[handleChannelNewBalance]")
	var transactionHash common.TransactionHash
	var isParticipant bool

	participantAddress := event["participant"].(common.Address)
	channelIdentifier := event["channelID"].(common.ChannelID)
	depositBlockHeight := event["blockHeight"].(common.BlockHeight)
	totalDeposit := event["totalDeposit"].(common.TokenAmount)

	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

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

func (self *ChannelService) HandleChannelClose(event map[string]interface{}) {

	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

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
		channelClosed := &transfer.ContractReceiveRouteClosed{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: transactionHash,
				BlockHeight:     blockNumber,
			},
			TokenNetworkIdentifier: tokenNetworkIdentifier,
			ChannelIdentifier:      channelIdentifier,
		}
		self.HandleStateChange(channelClosed)
	}
}

func (self *ChannelService) HandleChannelUpdateTransfer(event map[string]interface{}) {

	var transactionHash common.TransactionHash

	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)
	nonce := event["nonce"].(common.Nonce)

	chainState := self.StateFromChannel()
	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)
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

func (self *ChannelService) HandleChannelSettled(event map[string]interface{}) {
	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

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

func (self *ChannelService) HandleChannelWithdraw(event map[string]interface{}) {
	log.Info("[HandleChannelWithdraw]")
	var transactionHash common.TransactionHash
	var isParticipant bool

	participantAddress := event["participant"].(common.Address)
	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)
	totalWithdraw := event["totalWithdraw"].(common.TokenAmount)

	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(
		self.StateFromChannel(), tokenNetworkIdentifier, channelIdentifier)

	if channelState != nil {
		isParticipant = true
	}

	if isParticipant {
		channelWithdraw := &transfer.ContractReceiveChannelWithdraw{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: transactionHash,
				BlockHeight:     blockNumber,
			},
			TokenNetworkIdentifier: tokenNetworkIdentifier,
			ChannelIdentifier:      channelIdentifier,
			Participant:            participantAddress,
			TotalWithdraw:          totalWithdraw,
		}

		ourState := channelState.GetChannelEndState(0)
		if ourState != nil && common.AddressEqual(ourState.GetAddress(), participantAddress) {
			go self.HandleWithdrawSuccess(channelIdentifier)
		}

		self.HandleStateChange(channelWithdraw)
	}
	return
}

func (self *ChannelService) HandleWithdrawSuccess(channelId common.ChannelID) {
	ok := self.WithdrawResultNotify(channelId, true)
	if !ok {
		// when process saved event after restart, there is no withdraw status,but there
		// should be a withdrawTransaction in the channelState
		log.Warn("error in HandleWithdrawSuccess, no withdraw status found in the map")
	}

	return
}

func (self *ChannelService) HandleChannelCooperativeSettled(event map[string]interface{}) {
	log.Info("[HandleChannelCooperativeSettled]")
	var transactionHash common.TransactionHash

	channelIdentifier := event["channelID"].(common.ChannelID)
	blockNumber := event["blockHeight"].(common.BlockHeight)
	participant1Amount := event["participant1_amount"].(common.TokenAmount)
	participant2Amount := event["participant2_amount"].(common.TokenAmount)

	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateByTokenNetworkIdentifier(chainState,
		tokenNetworkIdentifier, channelIdentifier)

	if channelState != nil {
		channelCooperativeSettled := &transfer.ContractReceiveChannelCooperativeSettled{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: transactionHash,
				BlockHeight:     blockNumber,
			},
			TokenNetworkIdentifier: tokenNetworkIdentifier,
			ChannelIdentifier:      channelIdentifier,
			Participant1Amount:     participant1Amount,
			Participant2Amount:     participant2Amount,
		}

		self.HandleStateChange(channelCooperativeSettled)
	} else {
		//[TODO] generate ContractReceiveRouteClosed when supporting route
		channelClosed := &transfer.ContractReceiveRouteClosed{
			ContractReceiveStateChange: transfer.ContractReceiveStateChange{
				TransactionHash: transactionHash,
				BlockHeight:     blockNumber,
			},
			TokenNetworkIdentifier: tokenNetworkIdentifier,
			ChannelIdentifier:      channelIdentifier,
		}
		self.HandleStateChange(channelClosed)
	}

	return
}

func (self *ChannelService) HandleChannelBatchUnlock(event map[string]interface{}) {
	tokenNetworkIdentifier := common.TokenNetworkID(usdt.USDT_CONTRACT_ADDRESS)

	var transactionHash common.TransactionHash

	blockNumber := event["blockHeight"].(common.BlockHeight)
	//channelIdentifier := event["channelID"].(common.ChannelID)
	participant := event["participant"].(common.Address)
	partner := event["partner"].(common.Address)
	locksRoot := event["computedLocksroot"].(common.Locksroot)
	unlockedAmount := event["unlockedAmount"].(common.TokenAmount)
	returnedTokens := event["returnedTokens"].(common.TokenAmount)

	unlockStateChange := &transfer.ContractReceiveChannelBatchUnlock{
		ContractReceiveStateChange: transfer.ContractReceiveStateChange{
			TransactionHash: transactionHash,
			BlockHeight:     blockNumber,
		},
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		Participant:            participant,
		Partner:                partner,
		Locksroot:              locksRoot,
		UnlockedAmount:         unlockedAmount,
		ReturnedTokens:         returnedTokens,
	}
	self.HandleStateChange(unlockStateChange)
}

func (self *ChannelService) HandleSecretRevealed(event map[string]interface{}) {
	var transactionHash common.TransactionHash
	var secretRegistryAddress common.SecretRegistryAddress

	blockNumber := event["blockHeight"].(common.BlockHeight)
	secretHash := event["secretHash"].(common.SecretHash)
	secret := event["secret"].(common.Secret)
	log.Infof("[HandleSecretRevealed] receive event with blockHeight: %d, secretHash : %v, secret : %v",
		blockNumber, secretHash, secret)
	registeredSecretStateChange := &transfer.ContractReceiveSecretReveal{
		ContractReceiveStateChange: transfer.ContractReceiveStateChange{
			TransactionHash: transactionHash,
			BlockHeight:     blockNumber,
		},
		SecretRegistryAddress: secretRegistryAddress,
		SecretHash:            secretHash,
		Secret:                secret,
	}
	self.HandleStateChange(registeredSecretStateChange)
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
		channel.handleChannelNewBalance(events)
	} else if eventName == "chanSettled" {
		channel.HandleChannelSettled(events)
	} else if eventName == "NonClosingBPFUpdate" {
		channel.HandleChannelUpdateTransfer(events)
	} else if eventName == "SetTotalWithdraw" {
		channel.HandleChannelWithdraw(events)
	} else if eventName == "chanCooperativeSettled" {
		channel.HandleChannelCooperativeSettled(events)
	} else if eventName == "SecretRevealed" {
		channel.HandleSecretRevealed(events)
	} else if eventName == "ChannelUnlocked" {
		channel.HandleChannelBatchUnlock(events)
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
		RevealTimeout:            common.BlockTimeout(revealTimeout),
		SettleTimeout:            common.BlockTimeout(settleTimeout),
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
		case "partner":
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
			fallthrough
		case "totalWithdraw":
			fallthrough
		case "participant1_amount":
			fallthrough
		case "participant2_amount":
			fallthrough
		case "unlockedAmount":
			fallthrough
		case "returnedTokens":
			events[item] = common.TokenAmount(value.(float64))
		case "nonce":
			events[item] = common.Nonce(value.(float64))
		case "secret":
			var secret [constants.SECRET_LEN]byte

			for index, data := range value.([]interface{}) {
				value := data.(float64)
				secret[index] = byte(value)
			}
			events[item] = common.Secret(secret[:])
		case "secretHash":
			var secretHash common.SecretHash

			for index, data := range value.([]interface{}) {
				value := data.(float64)
				secretHash[index] = byte(value)
			}
			events[item] = secretHash
		case "computedLocksroot":
			var locksroot common.Locksroot

			for index, data := range value.([]interface{}) {
				value := data.(float64)
				locksroot[index] = byte(value)
			}
			events[item] = locksroot
		}
	}
	return events
}
