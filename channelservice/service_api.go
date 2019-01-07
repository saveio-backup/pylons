package channelservice

import (
	"container/list"
	"fmt"
	"time"

	"github.com/oniio/oniChannel/storage"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

func EventFilterForPayments(
	event transfer.Event,
	tokenNetworkIdentifier typing.TokenNetworkID,
	partnerAddress typing.Address) bool {

	var result bool

	result = false
	emptyAddress := typing.Address{}
	switch event.(type) {
	case *transfer.EventPaymentSentSuccess:
		eventPaymentSentSuccess := event.(*transfer.EventPaymentSentSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentSuccess.Target == typing.TargetAddress(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentReceivedSuccess:
		eventPaymentReceivedSuccess := event.(*transfer.EventPaymentReceivedSuccess)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentReceivedSuccess.Initiator == typing.InitiatorAddress(partnerAddress) {
			result = true
		}
	case *transfer.EventPaymentSentFailed:
		eventPaymentSentFailed := event.(*transfer.EventPaymentSentFailed)
		if partnerAddress == emptyAddress {
			result = true
		} else if eventPaymentSentFailed.Target == typing.TargetAddress(partnerAddress) {
			result = true
		}
	}

	return result
}

func (self *ChannelService) Address() typing.Address {
	return self.address
}

func (self *ChannelService) GetChannel(
	registryAddress typing.PaymentNetworkID,
	tokenAddress *typing.TokenAddress,
	partnerAddress *typing.Address) *transfer.NettingChannelState {

	var result *transfer.NettingChannelState

	channelList := self.GetChannelList(registryAddress, tokenAddress, partnerAddress)
	if channelList.Len() != 0 {
		result = channelList.Back().Value.(*transfer.NettingChannelState)
	}

	return result
}

func (self *ChannelService) tokenNetworkConnect(
	registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress,
	funds typing.TokenAmount,
	initialChannelTarget int,
	joinableFundsTarget float32) {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	connectionManager.connect(funds, initialChannelTarget, joinableFundsTarget)

	return
}

func (self *ChannelService) tokenNetworkLeave(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress) *list.List {

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(), registryAddress, tokenAddress)

	connectionManager := self.ConnectionManagerForTokenNetwork(
		tokenNetworkIdentifier)

	return connectionManager.Leave(registryAddress)
}

func (self *ChannelService) ChannelOpen(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address,
	settleTimeout typing.BlockTimeout, retryTimeout typing.NetworkTimeout) typing.ChannelID {

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, registryAddress,
		tokenAddress, partnerAddress)

	if channelState != nil {
		return channelState.Identifier
	}

	tokenNetwork := self.chain.TokenNetwork(typing.Address{})
	tokenNetwork.NewNettingChannel(partnerAddress, int(settleTimeout))

	WaitForNewChannel(self, registryAddress, tokenAddress, partnerAddress,
		float32(retryTimeout))

	channelState = transfer.GetChannelStateFor(self.StateFromChannel(), registryAddress, tokenAddress, partnerAddress)

	return channelState.Identifier

}

func (self *ChannelService) SetTotalChannelDeposit(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address, totalDeposit typing.TokenAmount,
	retryTimeout typing.NetworkTimeout) {

	chainState := self.StateFromChannel()
	channelState := transfer.GetChannelStateFor(chainState, registryAddress, tokenAddress, partnerAddress)
	if channelState == nil {
		return
	}

	args := self.GetPaymentArgs(channelState)
	if args == nil {
		panic("error in HandleContractSendChannelClose, cannot get paymentchannel args")
	}

	channelProxy := self.chain.PaymentChannel(typing.Address{}, channelState.Identifier, args)

	balance, err := channelProxy.GetBalance()
	if err != nil {
		return
	}

	addednum := totalDeposit - channelState.OurState.ContractBalance
	if balance < typing.Balance(addednum) {
		return
	}

	channelProxy.SetTotalDeposit(totalDeposit)
	targetAddress := self.address
	WaitForParticipantNewBalance(self, registryAddress, tokenAddress, partnerAddress,
		targetAddress, totalDeposit, float32(retryTimeout))

	return
}

func (self *ChannelService) ChannelClose(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress typing.Address,
	retryTimeout typing.NetworkTimeout) {

	addressList := list.New()
	addressList.PushBack(partnerAddress)

	self.ChannelBatchClose(registryAddress, tokenAddress, addressList, retryTimeout)
	return
}

func (self *ChannelService) ChannelBatchClose(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, partnerAddress *list.List, retryTimeout typing.NetworkTimeout) {

	chainState := self.StateFromChannel()
	channelsToClose := transfer.FilterChannelsByPartnerAddress(chainState, registryAddress,
		tokenAddress, partnerAddress)

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		chainState, registryAddress, tokenAddress)

	//[TODO] use BlockchainService.PaymentChannel to get PaymentChannels and
	// get the lock!!

	channelIds := list.New()
	for e := channelsToClose.Front(); e != nil; e = e.Next() {
		channelState := e.Value.(*transfer.NettingChannelState)

		identifier := channelState.GetIdentifier()
		//channel := self.chain.PaymentChannel(typing.Address{}, identifier, nil)
		/*
			lock := channel.LockOrRaise()
			lock.Lock()
			defer lock.Unlock()
		*/

		channelIds.PushBack(&identifier)

		channelClose := new(transfer.ActionChannelClose)
		channelClose.TokenNetworkIdentifier = tokenNetworkIdentifier
		channelClose.ChannelIdentifier = identifier

		self.HandleStateChange(channelClose)
	}

	WaitForClose(self, registryAddress, tokenAddress, channelIds, float32(retryTimeout))

	return
}

func (self *ChannelService) GetChannelList(registryAddress typing.PaymentNetworkID,
	tokenAddress *typing.TokenAddress, partnerAddress *typing.Address) *list.List {

	result := list.New()
	chainState := self.StateFromChannel()

	if tokenAddress != nil && partnerAddress != nil {
		channelState := transfer.GetChannelStateFor(chainState,
			registryAddress, *tokenAddress, *partnerAddress)

		if channelState != nil {
			result.PushBack(channelState)
		}
	} else if tokenAddress != nil {
		result = transfer.ListChannelStateForTokenNetwork(chainState, registryAddress,
			*tokenAddress)
	} else {
		result = transfer.ListAllChannelState(chainState)
	}

	return result
}

func (self *ChannelService) GetNodeNetworkState(nodeAddress typing.Address) string {
	return transfer.GetNodeNetworkStatus(self.StateFromChannel(), nodeAddress)
}

func (self *ChannelService) StartHealthCheckFor(nodeAddress typing.Address) {

	self.StartHealthCheckFor(nodeAddress)
	return
}

func (self *ChannelService) GetTokensList(registryAddress typing.PaymentNetworkID) *list.List {
	tokensList := transfer.GetTokenNetworkAddressesFor(self.StateFromChannel(),
		registryAddress)

	return tokensList
}

func (self *ChannelService) TransferAndWait(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID, transferTimeout int) {

	asyncResult := self.TransferAsync(registryAddress, tokenAddress, amount,
		target, identifier)

	select {
	case <-*asyncResult:
		break
	case <-time.After(time.Duration(transferTimeout) * time.Second):
		break
	}

	return
}

func (self *ChannelService) TransferAsync(registryAddress typing.PaymentNetworkID,
	tokenAddress typing.TokenAddress, amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID) *chan int {

	asyncResult := new(chan int)

	//[TODO] Adding Graph class to hold route information, support async transfer
	// by calling channel mediated_transfer_async
	return asyncResult
}

func (self *ChannelService) DirectTransferAsync(amount typing.TokenAmount, target typing.Address,
	identifier typing.PaymentID) chan bool {

	//Only one payment network
	paymentNetworkIdentifier := typing.PaymentNetworkID{}
	tokenAddress := typing.TokenAddress{}
	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(
		self.StateFromChannel(),
		paymentNetworkIdentifier,
		tokenAddress)

	asyncResult, err := self.DirectTransferAsync(tokenNetworkIdentifier, amount, typing.TargetAddress(target), identifier)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return asyncResult
}

func (self *ChannelService) GetEventsPaymentHistoryWithTimestamps(tokenAddress typing.TokenAddress,
	targetAddress typing.Address, limit int, offset int) *list.List {

	result := list.New()

	tokenNetworkIdentifier := transfer.GetTokenNetworkIdentifierByTokenAddress(self.StateFromChannel(),
		typing.PaymentNetworkID{}, tokenAddress)

	events := self.Wal.Storage.GetEventsWithTimestamps(limit, offset)
	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		if EventFilterForPayments(event.WrappedEvent, tokenNetworkIdentifier, targetAddress) == true {
			result.PushBack(event)
		}

	}

	return result

}

func (self *ChannelService) GetEventsPaymentHistory(tokenAddress typing.TokenAddress,
	targetAddress typing.Address, limit int, offset int) *list.List {
	result := list.New()

	events := self.GetEventsPaymentHistoryWithTimestamps(tokenAddress, targetAddress,
		limit, offset)

	for e := events.Front(); e != nil; e = e.Next() {
		event := e.Value.(*storage.TimestampedEvent)
		result.PushBack(event.WrappedEvent)

	}

	return result

}

func (self *ChannelService) GetInternalEventsWithTimestamps(limit int, offset int) *list.List {
	return self.Wal.Storage.GetEventsWithTimestamps(limit, offset)

}

func (self *ChannelService) GetBlockchainEventsNetwork(registryAddress typing.PaymentNetworkID,
	fromBlock typing.BlockHeight, toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}

func (self *ChannelService) GetBlockchainEventsTokenNetwork(tokenAddress typing.TokenAddress,
	fromBlock typing.BlockHeight, toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}

func (self *ChannelService) GetBlockchainEventsChannel(tokenAddress typing.TokenAddress,
	partnerAddress typing.Address, fromBlock typing.BlockHeight,
	toBlock typing.BlockHeight) *list.List {

	//[TODO] use blockchain.events to get chain event filter
	//only used by restful, can skip now
	return nil
}
