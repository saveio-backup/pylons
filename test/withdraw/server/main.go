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
	WALLET_PATH = "./wallet.dat" //address:Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC
	WALLET_PWD  = []byte("123456")
	FACOTR      = 1000000000
)
var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3001",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}
var chainClient *chainsdk.Chain
var our chaincomm.Address
var target chaincomm.Address

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var withdrawAmount = flag.Int("withdrawamount", 0, "test withdraw amount")
var routeNum = flag.Int("route", 5, "route number")
var timeout = flag.Int("timeout", 0, "timeout in second before withdraw")

func main() {
	log.InitLog(2, log.Stdout)
	flag.Parse()

	if *cpuProfile != "" {

		cupF, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cupF)
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

	chainClient = chainsdk.NewChain()
	chainClient.NewRpcClient().SetAddress(testConfig.ChainNodeURL)
	chainClient.SetDefaultAccount(account)

	log.Infof("asset balance before open channel : %d", getAssetBalance(account.Address))

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}

	target, _ = chaincomm.AddressFromBase58("AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")
	our, _ = chaincomm.AddressFromBase58("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	channel.Service.SetHostAddr(common.Address(target), "tcp://127.0.0.1:3000")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	time.Sleep(time.Second)

	go logCurrentBalance(channel, common.Address(target))
	for {
		state := transfer.GetNodeNetworkStatus(channel.Service.StateFromChannel(), common.Address(target))
		if state == transfer.NetworkReachable {
			log.Info("connect peer AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg successful")
			break
		}
		log.Infof("peer state = %s wait for connect ...", state)
		<-time.After(time.Duration(3000) * time.Millisecond)
	}
	if *disable == false {
		log.Info("begin direct single route transfer test...")
		go singleRouteTest(channel, 1*FACOTR, common.Address(target), *transferAmount, 0, *routeNum)
	}

	amount := *withdrawAmount * FACOTR
	if amount != 0 {
		withdrawTest(channel, amount, common.Address(target))
	}

	waitToExit()
}
func withdrawTest(channel *ch.Channel, withdrawAmount int, target common.Address) {
	if *disable == false {
		<-chInt
	}

	log.Infof("withdraw amount :%d", withdrawAmount)

	if *timeout != 0 {
		log.Infof("wait %d seconds before withdraw", *timeout)
		time.Sleep(time.Duration(*timeout) * time.Second)
	}

	log.Infof("call withdraw with amount :%d", withdrawAmount)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
	resultChan, err := channel.Service.Withdraw(tokenAddress, target, common.TokenAmount(withdrawAmount))
	if err != nil {
		log.Error("withdraw failed:", err)
		return
	}

	result := <-resultChan
	log.Infof("withdraw result : %v", result)
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
			log.Error("[loopTest] direct transfer failed:")
			break
		} else {
			//log.Info("[loopTest] direct transfer successfully")
		}
	}

	chInt <- routeId
}

func singleRouteTest(channel *ch.Channel, amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, 1)
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("loopTest can transfer!")
			//wait extra second to make sure client updated the netting channel state for the new balance event
			time.Sleep(2 * time.Second)
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
				log.Infof("test channel with %s haven`t been connected", "AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")
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
