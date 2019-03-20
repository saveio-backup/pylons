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
	"fmt"
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
		log.Error("Wallet.Open error:%s\n", err)
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
		log.Error("[SetTotalChannelDeposit] error: ", err.Error())
	}

	go mediaTransfer(channel, 1000)
	go receivePayment(channel)
	go currentBalance(channel)

	waitToExit()
}

func mediaTransfer(channel *ch.Channel, loopTimes int64) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	targetAddress, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
	target := common.Address(targetAddress)

	time1 := time.Now().Unix()
	for i := int64(1); i <= loopTimes; i++ {
		log.Info("[MediaTransfer] Transfer times: ", i)
		ret, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, 1, common.Address(target), common.PaymentID(i))
		if err != nil {
			log.Error("[MediaTransfer] MediaTransfer: ", err.Error())
		} else {
			log.Info("[MediaTransfer] MediaTransfer Calls Return")
		}
		r := <-ret
		if !r {
			log.Error("[MediaTransfer] MediaTransfer Failed")
			time.Sleep(time.Second)
		} else {
			log.Info("[MediaTransfer] MediaTransfer Success")
		}
	}
	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	fmt.Printf("[MediaTransfer] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", loopTimes, timeDuration, loopTimes/timeDuration)
}

func receivePayment(channel *ch.Channel) {
	var notificationChannel = make(chan *transfer.EventPaymentReceivedSuccess)
	channel.RegisterReceiveNotification(notificationChannel)
	log.Info("[ReceivePayment] RegisterReceiveNotification")

	var msg *transfer.EventPaymentReceivedSuccess
	for i := 0; ; i++ {
		log.Info("[ReceivePayment] WaitForReceiveNotification")
		select {
		case msg = <-notificationChannel:
			addr := common.ToBase58(common.Address(msg.Initiator))
			log.Infof("[ReceivePayment] Initiator: %v, Amount: %v Times: %v\n", addr, msg.Amount, i)
		}
	}
}

func currentBalance(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	partner := common.Address(partnerAddress)

	for {
		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
		log.Info()
		if chanState != nil {
			log.Info("[CurrentBalance] Local Balance: ", chanState.OurState.GetGasBalance())
			log.Info("[CurrentBalance] Local ContractBalance: ", chanState.OurState.ContractBalance)
			if chanState.OurState.BalanceProof != nil {
				log.Info("[CurrentBalance] Local LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
			}

			log.Info("[CurrentBalance] Partner Balance: ", chanState.PartnerState.GetGasBalance())
			log.Info("[CurrentBalance] Partner ContractBalance: ", chanState.PartnerState.ContractBalance)
			if chanState.PartnerState.BalanceProof != nil {
				log.Info("[CurrentBalance] Partner LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sc {
			log.Infof("Server received exit signal:%v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
