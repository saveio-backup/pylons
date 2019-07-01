package main

import (
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
	"github.com/saveio/pylons/test/p2p/actor/req"
	"github.com/saveio/pylons/actor/client"
	"github.com/saveio/carrier/crypto"
	"github.com/saveio/pylons/test/p2p"
	"github.com/saveio/themis/crypto/keypair"
	ch_actor "github.com/saveio/pylons/actor/server"
	p2p_actor "github.com/saveio/pylons/test/p2p/actor/server"
	tc "github.com/saveio/pylons/test/media/test_config"
)

func main() {
	log.Init(log.PATH, log.Stdout)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("[GetDefaultAccount] error:%s\n", err)
	}

	//start channel and actor
	ChannelActor, err := ch_actor.NewChannelActor(tc.Media, account)
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

	err = p2pserver.Start(tc.Media.Protocol + "://" + tc.Media.ListenAddress)
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

	_, err = ch_actor.OpenChannel(tokenAddress, tc.Target1Addr)
	if err != nil {
		log.Fatal(err)
		return
	}

	depositAmount := common.TokenAmount(1000 * 1000000000)
	log.Infof("start to deposit %d token to channel", depositAmount)
	err = ch_actor.SetTotalChannelDeposit(tokenAddress, tc.Target1Addr, depositAmount)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("deposit successful")

	go currentBalance(ChannelActor.GetChannelService())
	waitToExit()
}

func currentBalance(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
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
