package main

import (
	"fmt"
	"time"

	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/oniio/oniChain-go-sdk/usdt"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3002",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

var isNode1OnLine = false

var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int64("amount", 100, "test transfer amount")

func main() {
	flag.Parse()

	log.Init(log.PATH, log.Stdout)
	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		fmt.Printf("wallet.Open error:%s\n", err)
	}

	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		fmt.Printf("GetDefaultAccount error:%s\n", err)
	}

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[NewChannelService]")

	addr1, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	addr2, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	addr3, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")

	channel.Service.SetHostAddr(common.Address(addr1), "tcp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(addr2), "tcp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(addr3), "tcp://127.0.0.1:3002")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[StartService]")

	if *disable == false {
		//go mediaTransfer(channel, *transferAmount)
		go multiMediaTest(channel, *transferAmount)
	}
	go receivePayment(channel)
	go currentBalance(channel)

	waitToExit()
}

func mediaTransfer(channel *ch.Channel, loopTimes int64) {
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	targetAddress, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	target := common.Address(targetAddress)

	time1 := time.Now().Unix()
	for i := int64(1); i <= loopTimes; i++ {
		log.Info("[MediaTransfer] MediaTransfer times: ", i)
		ret, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, 1, common.Address(target), common.PaymentID(i))
		if err != nil {
			log.Error("[MediaTransfer] MediaTransfer: ", err.Error())
		} else {
			log.Debug("[MediaTransfer] MediaTransfer Return")
		}
		r := <-ret
		if !r {
			log.Error("[MediaTransfer] MediaTransfer Failed")
			time.Sleep(time.Second)
		} else {
			log.Debug("[MediaTransfer] MediaTransfer Success")
		}
	}
	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	log.Infof("[MediaTransfer] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", loopTimes, timeDuration, loopTimes/timeDuration)
}

func multiMediaTest(channel *ch.Channel, loopTimes int64) {
	for ; ;  {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	targetAddress, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	target := common.Address(targetAddress)
	log.Info("wait for multiMediaTest canTransfer...")

	var err error
	statuses := make([]chan bool, loopTimes)

	f :=  func(statuses []chan bool, index int64) error {
		statuses[index], err = channel.Service.MediaTransfer(registryAddress, tokenAddress, 1, target, common.PaymentID(index + 1))
		if err != nil {
			log.Error("[multiMediaTest] media transfer failed:", err)
			return fmt.Errorf("[multiMediaTest] media transfer failed:", err)
		}
		return nil
	}

	time1 := time.Now().Unix()
	for index1 := int64(0); index1 < loopTimes; index1++ {
		go f(statuses, index1)
	}

	for index2 := int64(0); index2 < loopTimes; index2++ {
		log.Debug("[multiMediaTest] wait for payment status update...")
		if statuses[index2] == nil  {
			time.Sleep(5* time.Millisecond)
			continue
		}
		ret := <-statuses[index2]
		if !ret {
			log.Error("[multiMediaTest] media transfer failed")
			break
		} else {
			//log.Info("[multiMediaTest] media transfer successfully")
		}
	}

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	log.Infof("[multiMediaTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", loopTimes, timeDuration, loopTimes/timeDuration)
	log.Info("[multiMediaTest] media transfer test done")
}

func receivePayment(channel *ch.Channel) {
	var notificationChannel = make(chan *transfer.EventPaymentReceivedSuccess)
	channel.RegisterReceiveNotification(notificationChannel)
	log.Info("[ReceivePayment] RegisterReceiveNotification")

	var msg *transfer.EventPaymentReceivedSuccess
	for i := 1; ; i++ {
		log.Debug("[ReceivePayment] WaitForReceiveNotification")
		select {
		case msg = <-notificationChannel:
			isNode1OnLine = true
			addr := common.ToBase58(common.Address(msg.Initiator))
			log.Infof("[ReceivePayment] Initiator: %v, Amount: %v Times: %v", addr, msg.Amount, i)
		}
	}
}

func currentBalance(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	partner := common.Address(partnerAddress)

	for {
		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
		if chanState != nil {
			var ourLocked, parLocked common.TokenAmount
			ourBalance := chanState.OurState.GetGasBalance()
			outCtBal := chanState.OurState.ContractBalance
			parBalance := chanState.PartnerState.GetGasBalance()
			parCtBal := chanState.PartnerState.ContractBalance

			if chanState.OurState.BalanceProof != nil {
				ourLocked = chanState.OurState.BalanceProof.LockedAmount
			}
			if chanState.PartnerState.BalanceProof != nil {
				parLocked = chanState.PartnerState.BalanceProof.LockedAmount
			}

			log.Infof("[Balance] Our[BL: %d CT: %d LK: %d] Par[BL: %d CT: %d LK: %d]",
				ourBalance, outCtBal, ourLocked, parBalance, parCtBal, parLocked)
		}
		time.Sleep(5 * time.Second)
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

/*
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
*/
