package main

import (
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
	tc "github.com/saveio/pylons/test/testnet/test_config"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis-go-sdk/wallet"
	chaincomm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/crypto/keypair"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

func main() {
	log.Init(log.PATH, log.Stdout)

	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("[GetDefaultAccount] error:%s\n", err)
	}

	var Media = &ch.ChannelConfig{
		ClientType:    tc.Parameters.BaseConfig.Init1ClientType,
		ChainNodeURLs: tc.Parameters.BaseConfig.ChainNodeURLs,
		ListenAddress: tc.Parameters.BaseConfig.DnsListenAddr,
	}

	//start channel and actor
	ChannelActor, err := ch_actor.NewChannelActor(Media, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = ChannelActor.SyncBlockData()
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

	err = channelP2p.Start(tc.Parameters.BaseConfig.DnsListenAddr)
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
	waitToExit()
}

func currentBalance(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58(tc.Parameters.BaseConfig.Init1Addr)
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
