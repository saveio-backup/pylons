package main

import (
	"time"
	"fmt"

	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/transfer"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3002",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

func main() {
	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		fmt.Printf("wallet.Open error:%s\n", err)
	}

	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		fmt.Printf("GetDefaultAccount error:%s\n", err)
	}
log.Debug()
	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[NewChannelService]")
	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[StartService]")

	var notificationChannel = make(chan *transfer.EventPaymentReceivedSuccess)
	channel.RegisterReceiveNotification(notificationChannel)
	log.Info("[RegisterReceiveNotification]")

	var msg *transfer.EventPaymentReceivedSuccess
	for i := 0; ;i++ {
		log.Info("[WaitForReceiveNotification]")
		select {
		case msg = <-notificationChannel:
			addr := chaincomm.Address(msg.Initiator)

			log.Info("Initiator1: %v, Amount: %v Times: %v\n", addr.ToBase58(), msg.Amount, i)

			//var tokenAddr common.TokenAddress
			//var partnerAddr common.Address
			//
			//chanStates := channel.Service.GetChannelList(msg.PaymentNetworkIdentifier, &tokenAddr, partnerAddr)
			//if chanStates == nil {
			//	fmt.Println("ChainStates == nil")
			//	os.Exit(0)
			//} else {
			//	fmt.Println("[GetChannelList] Len(chanStates) = ", len(chanStates))
			//}
			//
			//if chanStates[0].OurState == nil {
			//	fmt.Println("ChainState[0].OurState == nil")
			//}
			//fmt.Println("State.OurState.GetBalance: ", chanStates[0].OurState.GetBalance())
			//fmt.Println("State.OurState.ContractBalance: ", chanStates[0].OurState.ContractBalance)
			//if chanStates[0].OurState.BalanceProof != nil {
			//	fmt.Println("State.OurState.BalanceProof.LockedAmount: ", chanStates[0].OurState.BalanceProof.LockedAmount)
			//}
			//
			//fmt.Println("State.PartnerState.GetBalance: ", chanStates[0].PartnerState.GetBalance())
			//fmt.Println("State.PartnerState.ContractBalance: ", chanStates[0].PartnerState.ContractBalance)
			//if chanStates[0].PartnerState.BalanceProof != nil {
			//	fmt.Println("State.PartnerState.BalanceProof.LockedAmount: ", chanStates[0].PartnerState.BalanceProof.LockedAmount)
			//}
			//
			//
			//chanState := channel.Service.GetChannel(registryAddress, &tokenAddress, &partnerAddr)
			//fmt.Println()
			//fmt.Println("State.OurState.GetBalance: ", chanState.OurState.GetGasBalance())
			//fmt.Println("State.OurState.ContractBalance: ", chanState.OurState.ContractBalance)
			//if chanState.OurState.BalanceProof != nil {
			//	fmt.Println("State.OurState.BalanceProof.LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
			//}
			//
			//fmt.Println("State.PartnerState.GetBalance: ", chanState.PartnerState.GetGasBalance())
			//fmt.Println("State.PartnerState.ContractBalance: ", chanState.PartnerState.ContractBalance)
			//if chanState.PartnerState.BalanceProof != nil {
			//	fmt.Println("State.PartnerState.BalanceProof.LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
			//}
			time.Sleep(time.Second)
		}
	}
}
