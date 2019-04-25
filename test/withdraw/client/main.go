package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	chainsdk "github.com/oniio/oniChain-go-sdk"
	"github.com/oniio/oniChain-go-sdk/usdt"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/config"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

var (
	WALLET_PATH = "./wallet.dat" //address:AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg
	WALLET_PWD  = []byte("123")
	FACOTR      = 1000000000
)
var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

var chainClient *chainsdk.Chain
var our chaincomm.Address
var target chaincomm.Address

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var withdrawAmount = flag.Int("withdrawamount", 1000, "test withdraw amount")
var withdrawTimes = flag.Int("withdrawtimes", 1, "test withdraw times")
var routeNum = flag.Int("route", 5, "route number")
var timeout = flag.Int("timeout", 0, "timeout in second before withdraw")
var chclose = flag.Bool("close", false, "close channel")

func main() {
	log.InitLog(2, log.Stdout)
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

	chainClient = chainsdk.NewChain()
	chainClient.NewRpcClient().SetAddress(testConfig.ChainNodeURL)
	chainClient.SetDefaultAccount(account)

	log.Infof("asset balance before open channel : %d", getAssetBalance(account.Address))

	target, _ = chaincomm.AddressFromBase58("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	our, _ = chaincomm.AddressFromBase58("AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")
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
	depositAmount := common.TokenAmount(1000 * FACOTR)
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
				log.Info("connect peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC successful")
				break
			} else {
				log.Error("connect peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC failed")
			}

			log.Infof("peer state = %s wait for connect ...", state)
			<-time.After(time.Duration(3000) * time.Millisecond)
		}

		chInt = make(chan int, 1)

		if *disable == false {
			log.Info("begin direct single route transfer test...")
			go singleRouteTest(channel, 1*FACOTR, common.Address(target), *transferAmount, 0, *routeNum)
		}

		amount := *withdrawAmount * FACOTR
		withdrawTest(channel, amount, common.Address(target))

		if *chclose {
			log.Infof("start to close channel")
			channel.Service.ChannelClose(common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), common.Address(target), 0.5)
			log.Infof("channel closed")
		}
	} else {
		log.Fatal("setup channel failed, exit")
		return
	}

	waitToExit()
}

func withdrawTest(channel *ch.Channel, withdrawAmount int, target common.Address) {
	if *disable == false {
		<-chInt
	}

	for i := 1; i <= *withdrawTimes; i++ {
		total := withdrawAmount * i
		log.Infof("withdraw amount :%d", withdrawAmount)

		if *timeout != 0 {
			log.Infof("wait %d seconds before withdraw", *timeout)
			time.Sleep(time.Duration(*timeout) * time.Second)
		}

		log.Infof("call withdraw with amount :%d", total)
		tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
		resultChan, err := channel.Service.Withdraw(tokenAddress, target, common.TokenAmount(total))
		if err != nil {
			log.Error("withdraw failed:", err)
			return
		}

		result := <-resultChan
		log.Infof("withdraw result : %v", result)
	}
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
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("loopTest can transfer!")
			break
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	//time1 := time.Now().Unix()

	loopTest(channel, amount, target, times, interval, routingNum)

	//time2 := time.Now().Unix()
	//timeDuration := time2 - time1
	//log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/int(timeDuration))
}

func logCurrentBalance(channel *ch.Channel, targetAddr common.Address) {
	ticker := time.NewTicker(config.MIN_GEN_BLOCK_TIME * time.Second)

	for {
		select {
		case <-ticker.C:
			chainState := channel.Service.StateFromChannel()
			channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(common.Address(utils.MicroPayContractAddress)),
				common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS), targetAddr)
			if channelState == nil {
				log.Infof("test channel with %s haven`t been connected", "Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
				break
			}

			state := channelState.GetChannelEndState(0)
			log.Infof("current balance = %d, transfered = %d, withdraw = %d", state.GetContractBalance(), state.GetContractBalance()-state.GetGasBalance(), state.GetTotalWithdraw())
			state = channelState.GetChannelEndState(1)
			log.Infof("partner balance = %d, transfered = %d, withdraw = %d", state.GetContractBalance(), state.GetContractBalance()-state.GetGasBalance(), state.GetTotalWithdraw())

			log.Infof("current asset balance = %d, partner asset balance = %d", getAssetBalance(our), getAssetBalance(target))
		}
	}
}

func getAssetBalance(address chaincomm.Address) uint64 {
	bal, err := chainClient.Native.Usdt.BalanceOf(address)
	if err != nil {
		log.Fatal(err)
		return 0
	}

	return bal
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
