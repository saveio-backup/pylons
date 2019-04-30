package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	chaincomm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3002",
	Protocol:      "udp",
	// set the small timeout for test purpose
	SettleTimeout: "12",
	RevealTimeout: "5",
}

const FACTOR = 1000000000

var isNode1OnLine = false

var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var routeNum = flag.Int("route", 5, "route number")
var nowait = flag.Bool("nowait", false, "enable sending without waiting for pay result")

func main() {
	flag.Parse()

	//log.Init(log.PATH, log.Stdout)
	log.InitLog(2, log.Stdout)
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

	channel.Service.SetHostAddr(common.Address(addr1), "udp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(addr2), "udp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(addr3), "udp://127.0.0.1:3002")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[StartService]")

	if *disable == false {
		if *multiEnable {
			log.Info("begin media multi route transfer test...")
			go multiRouteTest(channel, 1*FACTOR, common.Address(addr1), *transferAmount, 0, *routeNum)
		} else {
			log.Info("begin media single route transfer test...")
			go singleRouteTest(channel, 1*FACTOR, common.Address(addr1), *transferAmount, 0, *routeNum)
		}
	}
	go receivePayment(channel)
	go currentBalance(channel)

	waitToExit()
}

var chInt chan int

func loopTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routeId int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	for index := int(0); index < times; index++ {
		status, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, common.TokenAmount(amount),
			target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] direct transfer failed:", err)
			break
		}

		if *nowait == false {
			ret := <-status
			if !ret {
				log.Error("[loopTest] direct transfer failed")
				break
			} else {
				//log.Info("[loopTest] direct transfer successfully")
			}
		}
	}

	//tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	//log.Info("[loopTest] channelClose")
	//channel.Service.ChannelClose(tokenAddress, target, 3)

	chInt <- routeId
}

func singleRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, 1)
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	//time1 := time.Now().Unix()
	loopTest(channel, amount, target, times, interval, routingNum)

	/*
		time2 := time.Now().Unix()
		timeDuration := time2 - time1
		log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n",
			times, timeDuration, times/int(timeDuration))
	*/
}

func multiRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, routingNum)
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	time1 := time.Now().Unix()

	for i := 0; i < routingNum; i++ {
		go loopTest(channel, amount, target, times, interval, i)
	}

	for i := 0; i < routingNum; i++ {
		<-chInt
	}

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	log.Infof("[multiRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n",
		times*routingNum, timeDuration, (times*routingNum)/int(timeDuration))
}

func receivePayment(channel *ch.Channel) {

	if *nowait == false {
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
	} else {
		for {
			registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
			tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

			partnerAddress, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
			partner := common.Address(partnerAddress)

			chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
			if chanState != nil {
				if chanState.PartnerState.BalanceProof != nil {
					parLocked := chanState.PartnerState.BalanceProof.LockedAmount
					if parLocked > 0 {
						isNode1OnLine = true
						log.Infof("receive locked transfer, set Node1 on line")
						return
					}
				}
			}
			time.Sleep(5 * time.Second)
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
