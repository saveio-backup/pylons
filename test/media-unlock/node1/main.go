package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	chaincomm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	Protocol:      "udp",
	// set the small timeout for test purpose
	SettleTimeout: "12",
	RevealTimeout: "5",
}

const FACTOR = 1000000000

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 500, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var routeNum = flag.Int("route", 5, "route number")
var nowait = flag.Bool("nowait", false, "enable sending without waiting for pay result")
var interval = flag.Int("interval", 0, "send interval in miliseconds")
var noclose = flag.Bool("noclose", false, "if should not call close channel")
var closetimeout = flag.Int("closetimeout", 0, "close channel with target after timeout")

func main() {
	flag.Parse()

	if *cpuProfile != "" {
		log.Infof("CPU Profile: %s", *cpuProfile)
		cupF, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer cupF.Close()

		if err := pprof.StartCPUProfile(cupF); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		} else {
			log.Info("start CPU profile.")
		}
		defer pprof.StopCPUProfile()
	}

	//log.Init(log.PATH, log.Stdout)
	log.InitLog(2, log.Stdout)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

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
	channel.Service.SetHostAddr(common.Address(local), "udp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(partner), "udp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(target), "udp://127.0.0.1:3002")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}

	notify := channel.Service.GetNewChannelNotifier()
	go func() {
		for {
			select {
			case info := <-notify:
				log.Errorf("New channel notify: channel id = %v, partner address = %v", info.ChannelIdentifier, info.PartnerAddress)
			}
		}
	}()

	chanId := channel.Service.OpenChannel(tokenAddress, common.Address(partner))
	log.Info("[OpenChannel] ChanId: ", chanId)

	err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(partner), 20000*FACTOR)
	if err != nil {
		log.Error("[SetTotalChannelDeposit] error: ", err.Error())
	}

	// wait to make sure mediator deposit succesfuly, otherwise, route from mediator to initiator may have not enough balance
	//time.Sleep(6 * time.Second)
	BlockUntilPartnerDeposit(channel)
	if *disable == false {
		if *multiEnable {
			log.Info("begin media multi route transfer test...")
			go multiRouteTest(channel, 1*FACTOR, common.Address(target), *transferAmount, *interval, *routeNum)
		} else {
			// should make before try to receive, otherwise, <- chInt may hang forever
			chInt = make(chan int, 1)
			log.Info("begin media single route transfer test...")
			go singleRouteTest(channel, 1*FACTOR, common.Address(target), *transferAmount, *interval, *routeNum)
		}
	}
	go receivePayment(channel)
	go currentBalance(channel)

	if !*noclose {
		if *closetimeout != 0 {
			time.Sleep(time.Duration(*closetimeout) * time.Second)

		} else {
			time.Sleep(20 * time.Second)
		}
		log.Info("[CloseChannel] locked transfer finish, close channel")
		channel.Service.ChannelClose(tokenAddress, common.Address(partner), 0.5)

		log.Info("channelClose finish")
	}

	waitToExit()
}

func BlockUntilPartnerDeposit(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	partner := common.Address(partnerAddress)

	for {
		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
		if chanState != nil {
			ourCtBal := chanState.OurState.ContractBalance
			parCtBal := chanState.PartnerState.ContractBalance

			if parCtBal == ourCtBal && parCtBal != 0 {
				log.Infof("parter has depost enough to support bidirection transfer")
				return
			}
		}
		time.Sleep(5 * time.Second)
	}
}

var chInt chan int

func loopTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routeId int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	for index := int(0); index < times; index++ {
		if interval > 0 {
			<-time.After(time.Duration(interval) * time.Millisecond)
		}
		status, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, common.TokenAmount(amount),
			target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] media transfer failed:", err)
			break
		}

		if *nowait == false {
			ret := <-status
			if !ret {
				log.Error("[loopTest] media transfer failed")
				break
			} else {
				//log.Info("[loopTest] media transfer successfully")
			}
		}
	}
	chInt <- routeId
	log.Info("[loopTest] signal the channel")
}

func singleRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	//chInt = make(chan int, 1)
	//time1 := time.Now().Unix()

	loopTest(channel, amount, target, times, interval, routingNum)

	//time2 := time.Now().Unix()
	//timeDuration := time2 - time1
	//log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/int(timeDuration))
}

func multiRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, routingNum)
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
	var notificationChannel = make(chan *transfer.EventPaymentReceivedSuccess)
	channel.RegisterReceiveNotification(notificationChannel)
	log.Info("[ReceivePayment] RegisterReceiveNotification")

	var msg *transfer.EventPaymentReceivedSuccess
	for i := 1; ; i++ {
		log.Debug("[ReceivePayment] WaitForReceiveNotification")
		select {
		case msg = <-notificationChannel:
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
			log.Infof("Server received exit signal:%v.", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
