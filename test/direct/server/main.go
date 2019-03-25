package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
	"fmt"

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
)
var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3001",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Uint("amount", 0, "test transfer amount")

func main() {
	log.InitLog(2, log.Stdout)
	flag.Parse()
	var amount uint
	amount = 1000
	if *cpuProfile != "" {

		cupF, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cupF)
		defer pprof.StopCPUProfile()
	}
	if *transferAmount != 0 {
		amount = *transferAmount
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

	target, _ := chaincomm.AddressFromBase58("AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")
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
		log.Info("begin direct transfer test...")
		go multiTest(channel, 1, common.Address(target), amount, 0)
	}

	waitToExit()
}

func loopTest(channel *ch.Channel, amount int, target common.Address, times uint, interval int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("loopTest can transfer!")
			break
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	for index := uint(0); index < times; index++ {
		if interval > 0 {
			<-time.After(time.Duration(interval) * time.Millisecond)
		}
		state := transfer.GetNodeNetworkStatus(channel.Service.StateFromChannel(), common.Address(target))
		if state != transfer.NetworkReachable {
			log.Error("[loopTest] peer AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg is not reachable ")
			continue
		} else {
			//log.Info("[loopTest] peer AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg is reachable ")
		}
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
			log.Info("[loopTest] direct transfer successfully")
		}
	}
	log.Info("[loopTest] direct transfer test done")
}


func multiTest(channel *ch.Channel, amount uint, target common.Address, times uint, interval int) {
	log.Info("wait for multiTest canTransfer...")
	for {
		if channel.Service.CanTransfer(target, common.TokenAmount(amount)) {
			log.Info("multiTest can transfer!")
			break
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}

	var err error
	statuses := make([]chan bool, times)

	f :=  func(statuses []chan bool, index uint) error {
		statuses[index], err = channel.Service.DirectTransferAsync(common.TokenAmount(amount), target, common.PaymentID(index+1))
		if err != nil {
			log.Error("[multiTest] direct transfer failed:", err)
			return fmt.Errorf("[multiTest] direct transfer failed:", err)
		}
		return nil
	}

	time1 := time.Now().Unix()
	for index1 := uint(0); index1 < times; index1++ {
		go f(statuses, index1)
	}

	for index2 := uint(0); index2 < times; index2++ {
		log.Debug("[multiTest] wait for payment status update...")
		if statuses[index2] == nil  {
			time.Sleep(5* time.Millisecond)
			continue
		}
		ret := <-statuses[index2]
		if !ret {
			log.Error("[multiTest] direct transfer failed")
			break
		} else {
			//log.Info("[multiTest] direct transfer successfully")
		}
	}

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	log.Infof("[multiTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/uint(timeDuration))
	log.Info("[multiTest] direct transfer test done")
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
				log.Infof("test channel with %s haven`t been connected", "AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")
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
