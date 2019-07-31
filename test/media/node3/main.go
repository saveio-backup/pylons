package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/saveio/carrier/crypto"
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/actor/client"
	ch_actor "github.com/saveio/pylons/actor/server"
	"github.com/saveio/pylons/common"
	tc "github.com/saveio/pylons/test/media/test_config"
	"github.com/saveio/pylons/test/p2p"
	"github.com/saveio/pylons/test/p2p/actor/req"
	p2p_actor "github.com/saveio/pylons/test/p2p/actor/server"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	chaincomm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var isNode1OnLine = false

var disable = flag.Bool("disable", false, "disable transfer test")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var multiEnable = flag.Bool("multi", false, "enable multi routes test")
var routeNum = flag.Int("route", 5, "route number")

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

	//start channel and actor
	ChannelActor, err := ch_actor.NewChannelActor(tc.Target1, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	if err = ch_actor.SetHostAddr(tc.Initiator1Addr, tc.Initiator1.ListenAddress); err != nil {
		log.Fatal(err)
		return
	}
	if err = ch_actor.SetHostAddr(tc.MediaAddr, tc.Media.ListenAddress); err != nil {
		log.Fatal(err)
		return
	}
	if err = ch_actor.SetHostAddr(tc.Target1Addr, tc.Target1.ListenAddress); err != nil {
		log.Fatal(err)
		return
	}
	chnPid := ChannelActor.GetLocalPID()
	//start p2p and actor
	p2pserver := p2p.NewP2P()
	bPrivate := keypair.SerializePrivateKey(account.PrivKey())
	bPub := keypair.SerializePublicKey(account.PubKey())
	p2pserver.Keys = &crypto.KeyPair{
		PrivateKey: bPrivate,
		PublicKey:  bPub,
	}

	err = p2pserver.Start(tc.Target1.Protocol + "://" + tc.Target1.ListenAddress)
	if err != nil {
		log.Fatal(err)
		return
	}
	P2pPid, err := p2p_actor.NewP2PActor(p2pserver)
	if err != nil {
		log.Fatal(err)
		return
	}
	//binding channel and p2p pid
	req.SetChannelPid(chnPid)
	client.SetP2pPid(P2pPid)
	//start channel service
	err = ChannelActor.Start()
	if err != nil {
		log.Fatal(err)
		return
	}
	time.Sleep(time.Second)

	if *disable == false {
		if *multiEnable {
			log.Info("begin media multi route transfer test...")
			go multiRouteTest(1, tc.Initiator1Addr, *transferAmount, 0, *routeNum)
		} else {
			log.Info("begin media single route transfer test...")
			go singleRouteTest(1, tc.Initiator1Addr, *transferAmount, 0, *routeNum)
		}
	}
	go receivePayment(ChannelActor.GetChannelService())
	go currentBalance(ChannelActor.GetChannelService())

	waitToExit()
}

var chInt chan int

func loopTest(amount int, target common.Address, times int, interval int, routeId int) {
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
		ret, err := ch_actor.MediaTransfer(registryAddress, tokenAddress, common.TokenAmount(amount),
			target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] media transfer failed:", err)
			break
		}

		if !ret {
			log.Error("[loopTest] media transfer failed")
			break
		} else {
			//log.Info("[loopTest] media transfer successfully")
		}
	}

	//tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	//log.Info("[loopTest] channelClose")
	//channel.Service.ChannelClose(tokenAddress, target, 3)

	chInt <- routeId
}

func singleRouteTest(amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, 1)
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
	}

	time1 := time.Now().Unix()
	loopTest(amount, target, times, interval, routingNum)

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	if timeDuration > 0 {
		log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n",
			times, timeDuration, times/int(timeDuration))
	}
}

func multiRouteTest(amount int, target common.Address, times int, interval int, routingNum int) {
	chInt = make(chan int, routingNum)
	for {
		if isNode1OnLine {
			break
		}
		time.Sleep(time.Second)
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

func receivePayment(channel *ch.Channel) {
	notificationChannel, err := ch_actor.RegisterReceiveNotification()
	if err != nil {
		log.Error("[ReceivePayment] error in RegisterReceiveNotification")
		return
	}
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
