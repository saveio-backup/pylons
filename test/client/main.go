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
	WALLET_PATH = "./wallet.dat" //address:AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg
	WALLET_PWD  = []byte("123")
)
var testConfig = &ch.ChannelConfig{
	ClientType:     "rpc",
	ChainNodeURL:   "http://127.0.0.1:20336",
	ListenAddress:  "127.0.0.1:3000",
	MappingAddress: "10.0.1.105:3000",
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

	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	target, _ := chaincomm.AddressFromHexString("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")

	channelID := channel.Service.OpenChannel(tokenAddress, common.Address(target))

	if channelID != 0 {
		go loopTest(channel, 1000, common.Address(target), 10, 1000)
	} else {
		log.Fatal("setup channel failed, exit")
		return
	}
	go logCurrentBalance(channel, common.Address(target))
	waitToExit()
}
func loopTest(channel *ch.Channel, amount int, target common.Address, times, interval int) {
	for index := 0; index < times; index++ {
		<-time.After(time.Duration(interval) * time.Millisecond)
		channel.Service.DirectTransferAsync(common.TokenAmount(amount), target, common.PaymentID(index))
	}
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
