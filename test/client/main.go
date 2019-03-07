package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
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
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}
var f *os.File
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var tranferAmount = flag.Uint("amount", 0, "test transfer amount")

func main() {
	log.InitLog(2, log.Stdout)
	flag.Parse()
	var amount uint
	amount = 10000

	if *cpuprofile != "" {
		cupf, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(cupf)
		defer pprof.StopCPUProfile()
	}
	if *tranferAmount != 0 {
		amount = *tranferAmount
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

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}

	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	target, _ := chaincomm.AddressFromBase58("Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	go logCurrentBalance(channel, common.Address(target))
	channelID := channel.Service.OpenChannel(tokenAddress, common.Address(target))
	depositAmount := common.TokenAmount(1000000)
	if channelID != 0 {
		log.Infof("start to deposit %d token to channel", depositAmount)
		err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(target), depositAmount)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Info("deposit successful")
		for {
			state := transfer.GetNodeNetworkStatus(channel.Service.StateFromChannel(), common.Address(target))
			if state == transfer.NetworkReachable {
				log.Info("connect peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC successful")
				break
			} else {
				log.Error("connect peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC failed")
			}

			log.Infof("peer state = %s wait for connect ...", state)
			<-time.After(time.Duration(3000) * time.Millisecond)
		}
		log.Info("begin direct transfer test...")
		go loopTest(channel, 1, common.Address(target), amount, 0)
	} else {
		log.Fatal("setup channel failed, exit")
		return
	}

	waitToExit()
}
func loopTest(channel *ch.Channel, amount uint, target common.Address, times uint, interval int) {

	r := rand.NewSource(time.Now().UnixNano())
	for index := uint(0); index < times; index++ {
		if interval > 0 {
			<-time.After(time.Duration(interval) * time.Millisecond)
		}

		state := transfer.GetNodeNetworkStatus(channel.Service.StateFromChannel(), common.Address(target))
		if state != transfer.NetworkReachable {
			log.Error("[loopTest] peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC is not reachable ")
			break
		} else {
			//log.Info("[loopTest] peer Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC is reachable ")
		}
		status, err := channel.Service.DirectTransferAsync(common.TokenAmount(amount), target, common.PaymentID(r.Int63()))
		if err != nil {
			log.Error("[loopTest] direct transfer failed:", err)
			break
		} else {
			log.Debug("[loopTest] direct transfer successfully")
		}

		log.Debug("[loopTest] wait for payment status update...")
		ret := <-status
		if !ret {
			log.Error("[loopTest] payment failed")
			break
		} else {
			log.Debug("[loopTest] payment successfully")
		}
		log.Debugf("[loopTest] direct transfer %f ong to %s successfully", 100*float32(amount)/1000000000, "Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
	}
	log.Info("[loopTest] direct transfer test done")
}
func logCurrentBalance(channel *ch.Channel, target common.Address) {
	ticker := time.NewTicker(config.MIN_GEN_BLOCK_TIME * time.Second)

	for {
		select {
		case <-ticker.C:
			chainState := channel.Service.StateFromChannel()
			channelState := transfer.GetChannelStateFor(chainState, common.PaymentNetworkID(common.Address(utils.MicroPayContractAddress)),
				common.TokenAddress(ong.ONG_CONTRACT_ADDRESS), target)
			if channelState == nil {
				log.Infof("test channel with %s haven`t been connected", "Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC")
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
