package main

import (
	"github.com/oniio/oniChain-go-sdk/ong"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
	"time"
	"os"
	"os/signal"
	"syscall"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

func main() {
	log.Init(log.PATH, log.Stdout)
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("[GetDefaultAccount] error:%s\n", err)
	}

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[NewChannelService]")

	initiator, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	local, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	target, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")

	channel.Service.SetHostAddr(common.Address(initiator), "tcp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(local), "tcp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(target), "tcp://127.0.0.1:3002")

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[StartService]")

	chanId := channel.Service.OpenChannel(tokenAddress, common.Address(target))
	log.Info("[OpenChannel] ChanId: ", chanId)

	err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(target), 20000)
	if err != nil {
		log.Error("[SetTotalChannelDeposit]: ", err.Error())
		return
	}

	go currentBalance(channel)

	waitToExit()
}

func currentBalance(channel *ch.Channel) {
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)
	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)

	partnerAddress, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
	partner := common.Address(partnerAddress)

	for {
		chanState := channel.Service.GetChannel(registryAddress, tokenAddress, partner)
		log.Info()
		if chanState != nil {
			log.Info("[CurrentBalance] Local Balance: ", chanState.OurState.GetGasBalance())
			log.Info("[CurrentBalance] Local ContractBalance: ", chanState.OurState.ContractBalance)
			if chanState.OurState.BalanceProof != nil {
				log.Info("[CurrentBalance] Local LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
			} else {
				log.Warn("[CurrentBalance] OurBalance is nil")
			}

			log.Info("[CurrentBalance] Partner Balance: ", chanState.PartnerState.GetGasBalance())
			log.Info("[CurrentBalance] Partner ContractBalance: ", chanState.PartnerState.ContractBalance)
			if chanState.PartnerState.BalanceProof != nil {
				log.Info("[CurrentBalance] Partner LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
			} else {
				log.Warn("[CurrentBalance] PartnerBalance is nil")
			}
		}
		time.Sleep(3 * time.Second)
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
