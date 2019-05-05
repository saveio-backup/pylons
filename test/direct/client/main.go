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
	"github.com/saveio/themis/common/config"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var (
	WALLET_PATH = "./wallet.dat" //address:AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg
	WALLET_PWD  = []byte("123")
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	//MappingAddress: "10.0.1.105:3000",
	Protocol: "tcp",
}

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var routeNum = flag.Int("route", 5, "route number")

func main() {
	log.Init(log.PATH, log.Stdout)
	//log.InitLog(2, log.Stdout)
	flag.Parse()
	if *cpuProfile != "" {
		cupF, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer cupF.Close()
		if err := pprof.StartCPUProfile(cupF); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}

		defer pprof.StopCPUProfile()
	}

	wallet, err := wallet.OpenWallet(WALLET_PATH)
	if err != nil {
		log.Fatal("wallet.Open error:%s", err)
		return
	}
	account, err := wallet.GetDefaultAccount(WALLET_PWD)
	if err != nil {
		log.Fatal("GetDefaultAccount error:%s", err)
		return
	}

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}

	target, _ := chaincomm.AddressFromBase58("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	channel.Service.SetHostAddr(common.Address(target), "tcp://127.0.0.1:3001")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	time.Sleep(time.Second)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	go logCurrentBalance(channel, common.Address(target))
	channelID := channel.Service.OpenChannel(tokenAddress, common.Address(target))
	depositAmount := common.TokenAmount(1000 * 1000000000)
	if channelID != 0 {
		log.Infof("start to deposit %d token to channel", depositAmount)
		err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(target), depositAmount)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Info("deposit successful")

		for {
			state := transfer.GetNodeNetworkStatus(channel.Service.StateFromChannel(), common.Address(target))
			if state == transfer.NetworkReachable {
				log.Info("connect peer successful")
				break
			} else {
				log.Error("connect peer failed")
			}

			log.Infof("peer state = %s wait for connect ...", state)
			<-time.After(time.Duration(3000) * time.Millisecond)
		}
		if *disable == false {
			if *multiEnable {
				log.Info("begin direct multi route transfer test...")
				go multiRouteTest(channel, 1, common.Address(target), *transferAmount, 0, *routeNum)
			} else {
				log.Info("begin direct single route transfer test...")
				go singleRouteTest(channel, 1, common.Address(target), *transferAmount, 0, *routeNum)
			}
		}
	} else {
		log.Fatal("setup channel failed, exit")
		return
	}

	waitToExit()
}

var chInt chan int

func loopTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routeId int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	for index := int(0); index < times; index++ {
		status, err := channel.Service.DirectTransferAsync(common.TokenAmount(amount), target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] direct transfer failed:", err)
			break
		}

		ret := <-status
		if !ret {
			log.Error("[loopTest] direct transfer failed")
			break
		} else {
			//log.Info("[loopTest] direct transfer successfully")
		}
	}

	//tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
	//log.Info("[loopTest] channelClose")
	//channel.Service.ChannelClose(tokenAddress, target, 3)

	chInt <- routeId
}

func singleRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, 1)
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("loopTest can transfer!")
			break
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	time1 := time.Now().Unix()

	loopTest(channel, amount, target, times, interval, routingNum)

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/int(timeDuration))
}

func multiRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, routingNum)
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("loopTest can transfer!")
			break
		} else {
			time.Sleep(100 * time.Millisecond)
		}
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

func logCurrentBalance(channel *ch.Channel, target common.Address) {
	ticker := time.NewTicker(config.MIN_GEN_BLOCK_TIME * time.Second)

	for {
		select {
		case <-ticker.C:
			chainState := channel.Service.StateFromChannel()
			channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(common.Address(utils.MicroPayContractAddress)),
				common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), target)
			if channelState == nil {
				log.Infof("test channel with %s haven`t been connected", "Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
				break
			}
			state := channelState.GetChannelEndState(0)
			log.Infof("current balance = %d, transfered = %d", state.GetContractBalance(), state.GetContractBalance()-state.GetGasBalance())
			state = channelState.GetChannelEndState(1)
			log.Infof("partner balance = %d, transfered = %d", state.GetContractBalance(), state.GetContractBalance()-state.GetGasBalance())
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
