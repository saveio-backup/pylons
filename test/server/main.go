package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oniio/oniChain-go-sdk/ong"
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
	ClientType:     "rpc",
	ChainNodeURL:   "http://127.0.0.1:20336",
	ListenAddress:  "127.0.0.1:3001",
	MappingAddress: "10.0.1.105:3001",
	Protocol:       "tcp",
}

func main() {
	log.InitLog(0, log.Stdout)
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

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	target, _ := chaincomm.AddressFromHexString("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	go logCurrentBalance(channel, common.Address(target))
	waitToExit()
}
func logCurrentBalance(channel *ch.Channel, target common.Address) {
	ticker := time.NewTicker(config.DEFAULT_GEN_BLOCK_TIME * time.Second)
	for {
		select {
		case <-ticker.C:
			chainState := channel.Service.StateFromChannel()
			channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(common.Address(utils.MicroPayContractAddress)),
				common.TokenAddress(ong.ONG_CONTRACT_ADDRESS), target)
			if channelState == nil {
				log.Info("channel haven`t setup")
				break
			}
			state := channelState.GetChannelEndState(0)
			log.Infof("current balance = %d, transfer balance = %d", state.ContractBalance, state.GetGasBalance())
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
