package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/saveio/carrier/crypto"
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/actor/client"
	ch_actor "github.com/saveio/pylons/actor/server"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/test/p2p/actor/req"
	p2p_actor "github.com/saveio/pylons/test/p2p/actor/server"
	p2p "github.com/saveio/pylons/test/p2p/network"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	chaincomm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/config"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var (
	WALLET_PATH = "./wallet.dat" //address:Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC
	WALLET_PWD  = []byte("123456")
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3001",
	Protocol: "tcp",
}

var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var routeNum = flag.Int("route", 5, "route number")

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
	target, _ := chaincomm.AddressFromBase58("AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg")

	//start channel and actor
	ChannelActor, err := ch_actor.NewChannelActor(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = ch_actor.SetHostAddr(common.Address(target), "127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
		return
	}
	chnPid := ChannelActor.GetLocalPID()

	//start p2p and actor
	channelP2p := p2p.NewP2P()
	bPrivate := keypair.SerializePrivateKey(account.PrivKey())
	bPub := keypair.SerializePublicKey(account.PubKey())
	channelP2p.Keys = &crypto.KeyPair{
		PrivateKey: bPrivate,
		PublicKey:  bPub,
	}
	err = channelP2p.Start(testConfig.Protocol + "://" + testConfig.ListenAddress)
	if err != nil {
		log.Fatal(err)
		return
	}
	p2pActor, err := p2p_actor.NewP2PActor()
	if err != nil {
		log.Fatal(err)
		return
	}
	p2pActor.SetChannelNetwork(channelP2p)

	//binding channel and p2p pid
	req.SetChannelPid(chnPid)
	client.SetP2pPid(p2pActor.GetLocalPID())
	//start channel service
	err = ChannelActor.Start()
	if err != nil {
		log.Fatal(err)
		return
	}

	time.Sleep(time.Second)

	go logCurrentBalance(ChannelActor.GetChannelService(), common.Address(target))
	for {
		ret, _ := ch_actor.ChannelReachable(common.Address(target))
		var state string
		if ret == true {
			state = transfer.NetworkReachable
			log.Info("connect peer AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg successful")
			break
		} else {
			state = transfer.NetworkUnreachable
			log.Warn("connect peer failed")
			ch_actor.HealthyCheckNodeState(common.Address(target))
		}

		log.Infof("peer state = %s wait for connect ...", state)
		<-time.After(time.Duration(3000) * time.Millisecond)
	}
	if *disable == false {
		if *multiEnable {
			log.Info("begin direct multi route transfer test...")
			go multiRouteTest(1, common.Address(target), *transferAmount, 0, *routeNum)
		} else {
			log.Info("begin direct single route transfer test...")
			go singleRouteTest(1, common.Address(target), *transferAmount, 0, *routeNum)
		}
	}

	waitToExit()
}

var chInt chan int

func loopTest(amount int, target common.Address, times int, interval int, routeId int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	for index := int(0); index < times; index++ {
		ret, err := ch_actor.DirectTransferAsync(common.TokenAmount(amount), target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] direct transfer failed:", err)
			break
		}

		if !ret {
			log.Error("[loopTest] direct transfer failed:")
			break
		} else {
			//log.Info("[loopTest] direct transfer successfully")
		}
	}

	chInt <- routeId
}

func singleRouteTest(amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, 1)
	for {
		ret, err := ch_actor.CanTransfer(target, common.TokenAmount(amount))
		if ret {
			log.Info("loopTest can transfer!")
			break
		} else {
			if err != nil {
				log.Error("loopTest cannot transfer!")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	time1 := time.Now().Unix()

	loopTest(amount, target, times, interval, routingNum)

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	if timeDuration > 0 {
		log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/int(timeDuration))
	}
}

func multiRouteTest(amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, routingNum)
	for {
		ret, err := ch_actor.CanTransfer(target, common.TokenAmount(amount))
		if ret {
			log.Info("loopTest can transfer!")
			break
		} else {
			if err != nil {
				log.Error("loopTest cannot transfer!")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	time1 := time.Now().Unix()

	for i := 0; i < routingNum; i++ {
		go loopTest(amount, target, times, interval, i)
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
