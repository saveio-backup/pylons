package main

import (
	"flag"
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
	p2p_actor "github.com/saveio/pylons/test/p2p/actor/server"
	p2p "github.com/saveio/pylons/test/p2p/network"
	tc "github.com/saveio/pylons/test/test/config"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var mate = flag.String("mate", "", "node which open channel with")
var closeMate = flag.String("close", "", "close channel")
var deposit = flag.Int("deposit", 10000, "deposit count")
var transferTo = flag.String("transferTo", "", "transfer asset to")
var transferAmount = flag.Int("amount", 1000, "test transfer amount")
var transferType = flag.Int("type", 0, "transferType [0: MediaTransfer; 1: DirectTransfer; Other: MediaTransferBySpecificMedia]")

var tokenAddress = common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)
var channelActor *ch_actor.ChannelActorServer

func StartPylons() {
	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("Wallet.Open error:%s\n", err)
		return
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("GetDefaultAccount error:%s\n", err)
		return
	}

	var walletAddr common.Address
	copy(walletAddr[:], account.Address[:])
	log.Info("StartPylons WalletAddress: ", common.ToBase58(walletAddr))
	listenAddr, err := tc.GetHostAddrCallBack(walletAddr)
	if err != nil {
		log.Errorf("GetHostAddrCallBack error:%s\n", err.Error())
		return
	}
	var NodeConfig = &ch.ChannelConfig{
		ClientType:    tc.Parameters.BaseConfig.ChainClientType,
		ChainNodeURLs: tc.Parameters.BaseConfig.ChainNodeURLs,
		SettleTimeout: "50",
		RevealTimeout: "20",
	}

	//start channel and actor
	channelActor, err = ch_actor.NewChannelActor(NodeConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = channelActor.SyncBlockData()
	if err != nil {
		log.Fatal(err)
		return
	}

	//start p2p
	channelP2p := p2p.NewP2P()
	bPrivate := keypair.SerializePrivateKey(account.PrivKey())
	bPub := keypair.SerializePublicKey(account.PubKey())
	channelP2p.Keys = &crypto.KeyPair{
		PrivateKey: bPrivate,
		PublicKey:  bPub,
	}

	err = channelP2p.Start(listenAddr)
	if err != nil {
		log.Fatal(err)
		return
	}

	//bind p2p and p2p actor
	p2pActor, err := p2p_actor.NewP2PActor()
	if err != nil {
		log.Fatal(err)
		return
	}
	p2pActor.SetChannelNetwork(channelP2p)

	//binding channel and p2p actor
	client.SetP2pPid(p2pActor.GetLocalPID())

	if err = ch_actor.SetGetHostAddrCallback(tc.GetHostAddrCallBack); err != nil {
		log.Fatal(err)
		return
	}

	//start channel service
	err = ch_actor.StartPylons()
	if err != nil {
		log.Fatal(err)
		return
	}
	time.Sleep(time.Second)
}

func main() {
	flag.Parse()
	log.Init(log.PATH, log.Stdout)

	log.Info("StartPylons")
	StartPylons()

	allChannels, err := ch_actor.GetAllChannels()
	if err != nil {
		log.Error("[GetAllChannels] error: ", err.Error())
	}

	log.Info("[Show All Channels: ]")
	for _, channel := range allChannels.Channels {
		log.Infof("[Channel]: Address: %s, ChannelId: %d", channel.Address, channel.ChannelId)
	}

	if *mate != "" {
		log.Infof("[OpenChannel] Open Channel with %s", *mate)
		mateAddr, _ := common.FromBase58(*mate)
		channelId, err := ch_actor.OpenChannel(tokenAddress, mateAddr)
		if err != nil {
			log.Fatal(err)
			return
		}
		if channelId != 0 {
			depositAmount := common.TokenAmount(*deposit * 1000)
			log.Infof("[SetTotalChannelDeposit] start to deposit %d token to channel %d", depositAmount, channelId)
			err = ch_actor.SetTotalChannelDeposit(tokenAddress, mateAddr, depositAmount)
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Info("[SetTotalChannelDeposit] deposit successful")
			time.Sleep(6 * time.Second)
		}
		if *transferTo != "" {
			log.Info("begin media single route transfer test...")
			transferTo, _ := common.FromBase58(*transferTo)

			go singleRouteTest(mateAddr, 1, transferTo, *transferAmount)
		}
		go currentBalance(channelActor.GetChannelService(), mateAddr)
	}

	if *closeMate != "" {
		log.Infof("[CloseChannel] Close Channel with %s", *closeMate)
		mateAddr, _ := common.FromBase58(*closeMate)
		ok, err := ch_actor.CloseChannel(mateAddr)
		if err != nil {
			log.Error("[CloseChannel] error: ", err.Error())
		}
		if ok {
			log.Info("[CloseChannel] successful")
		}
	}

	go receivePayment()

	waitToExit()
}

var chInt chan int

func DirectLoopTest(amount int, target common.Address, times int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	for index := int(0); index < times; index++ {
		ret, err := ch_actor.DirectTransferAsync(target, common.TokenAmount(amount), common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] direct transfer failed:", err)
			break
		}

		if !ret {
			log.Error("[loopTest] direct transfer failed")
			break
		} else {
			//log.Info("[loopTest] direct transfer successfully")
		}
	}

	chInt <- 0
}

func MediaLoopTest(amount int, target common.Address, times int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	for index := int(0); index < times; index++ {
		ret, err := ch_actor.MediaTransfer(registryAddress, tokenAddress, common.EmptyAddress, target, common.TokenAmount(amount),
			common.PaymentID(r.Int63()))
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
	chInt <- 0
}

func MediaLoopTestBySpecificMedia(media common.Address, amount int, target common.Address, times int) {
	r := rand.NewSource(time.Now().UnixNano())
	log.Info("wait for loopTest canTransfer...")

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	for index := int(0); index < times; index++ {
		ret, err := ch_actor.MediaTransfer(registryAddress, tokenAddress, media, target, common.TokenAmount(amount),
			common.PaymentID(r.Int63()))
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
	chInt <- 0
}

func singleRouteTest(media common.Address, microAmount int, target common.Address, times int) {
	chInt = make(chan int, 1)
	time1 := time.Now().Unix()

	if *transferType == 0 {
		MediaLoopTest(microAmount, target, times)
	} else if *transferType == 1 {
		DirectLoopTest(microAmount, target, times)
	} else {
		MediaLoopTestBySpecificMedia(media, microAmount, target, times)
	}

	time2 := time.Now().Unix()
	timeDuration := time2 - time1
	if timeDuration > 0 {
		log.Infof("[singleRouteTest] LoopTimes: %v, TimeDuration: %v, Speed: %v\n", times, timeDuration, times/int(timeDuration))
	}
}

func receivePayment() {
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
			addr := common.ToBase58(common.Address(msg.Initiator))
			log.Infof("[ReceivePayment] Initiator: %v, Amount: %v Times: %v", addr, msg.Amount, i)
		}
	}
}

func currentBalance(channel *ch.Channel, mateAddr common.Address) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	for {
		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, mateAddr)
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
