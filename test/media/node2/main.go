package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oniio/oniChain-go-sdk/usdt"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
)

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "udp",
	RevealTimeout: "1000",
}

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

	channel, err := ch.NewChannelService(testConfig, account)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[NewChannelService]")

	initiator, _ := chaincomm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	local, _ := chaincomm.AddressFromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	target, _ := chaincomm.AddressFromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")

	channel.Service.SetHostAddr(common.Address(initiator), "udp://127.0.0.1:3000")
	channel.Service.SetHostAddr(common.Address(local), "udp://127.0.0.1:3001")
	channel.Service.SetHostAddr(common.Address(target), "udp://127.0.0.1:3002")

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
