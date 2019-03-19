package main

import (
	"github.com/oniio/oniChain-go-sdk/ong"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

func main() {
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("GetDefaultAccount error:%s\n", err)
	}

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}

	local, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	partner, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	target, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
	channel.Service.SetHostAddr(common.Address(local), "tcp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(partner), "tcp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(target), "tcp://127.0.0.1:3002")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}

	chanId := channel.Service.OpenChannel(tokenAddress, common.Address(partner))
	log.Info("[OpenChannel] ChanId: ", chanId)

	err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(partner), 20000)
	if err != nil {
		log.Error("SetTotalChannelDeposit: ", err.Error())
	}

	go mediaTransfer(channel)
	go receivePayment(channel)

	waitToExit()
}

func mediaTransfer(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	targetAddress, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
	target := common.Address(targetAddress)
	partnerAddress, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	partner := common.Address(partnerAddress)

	for i := 0; i < 1000; i++ {
		log.Info("MediaTransfer times: ", i)
		ret, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, 1, common.Address(target), common.PaymentID(i))
		if err != nil {
			log.Error("MediaTransfer: ", err.Error())
		} else {
			log.Info("MediaTransfer Return")
		}
		r := <-ret
		if !r {
			log.Error("MediaTransfer Failed")
			time.Sleep(time.Second)
		} else {
			log.Info("MediaTransfer Success")
			time.Sleep(time.Second)
		}

		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
		log.Info()
		log.Info("Local Balance: ", chanState.OurState.GetGasBalance())
		log.Info("Local ContractBalance: ", chanState.OurState.ContractBalance)
		if chanState.OurState.BalanceProof != nil {
			log.Info("Local LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
		}

		log.Info("Partner Balance: ", chanState.PartnerState.GetGasBalance())
		log.Info("Partner ContractBalance: ", chanState.PartnerState.ContractBalance)
		if chanState.PartnerState.BalanceProof != nil {
			log.Info("Partner LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
		}
	}
}

func receivePayment(channel *ch.Channel) {
	var notificationChannel = make(chan *transfer.EventPaymentReceivedSuccess)
	channel.RegisterReceiveNotification(notificationChannel)
	log.Info("[RegisterReceiveNotification]")

	var msg *transfer.EventPaymentReceivedSuccess
	for i := 0; ; i++ {
		log.Info("[WaitForReceiveNotification]")
		select {
		case msg = <-notificationChannel:
			addr := chaincomm.Address(msg.Initiator)

			log.Infof("Initiator: %v, Amount: %v Times: %v\n", addr.ToBase58(), msg.Amount, i)

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

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			log.Infof("server received exit signal:%v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
