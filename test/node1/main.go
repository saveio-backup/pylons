package main

import (
	"fmt"
	"github.com/oniio/oniChain-go-sdk/ong"
	"github.com/oniio/oniChain-go-sdk/wallet"
	chaincomm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
	"os"
	"time"
)

//var testConfig = &nimbus.NimbusConfig{
//	ChainNodeURL:  "http://127.0.0.1:20336",
//	ListenAddress: "127.0.0.1:3000",
//	Protocol:      "tcp",
//}

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

func main() {
	if len(os.Args) != 3 {
		log.Error("Param error")
		return
	}

	wallet, err := wallet.OpenWallet("./wallet.dat")
	if err != nil {
		log.Error("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount([]byte("pwd"))
	if err != nil {
		log.Error("GetDefaultAccount error:%s\n", err)
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

	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)

	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	partnerAddress, err := chaincomm.AddressFromBase58(os.Args[1])
	if err != nil {
		log.Error(err)
		return
	}
	partnerAddr := common.Address(partnerAddress)

	targetAddress, err := chaincomm.AddressFromBase58(os.Args[2])
	if err != nil {
		log.Error(err)
		return
	}
	target := common.Address(targetAddress)

	chanId := channel.Service.OpenChannel(tokenAddress, common.Address(partnerAddress))
	log.Info("[OpenChannel] ChanId: ", chanId)

	err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(partnerAddress), 20000)
	if err != nil {
		log.Error("SetTotalChannelDeposit: ", err.Error())
	}

	for i := 0; i < 1000; i++ {
		fmt.Println("MediaTransfer times1: ", i)
		ret, err := channel.Service.MediaTransfer(registryAddress, tokenAddress, 10, target, common.PaymentID(i))
		if err != nil {
			log.Error("MediaTransfer: ", err.Error())
		}
		r := <-ret
		if !r {
			log.Error("MediaTransfer Failed")
			time.Sleep(time.Second)
		} else {
			log.Info("MediaTransfer Success")
			time.Sleep(time.Second)
		}

		log.Info("GetChannel times: ", i)

		chanState := channel.Service.GetChannel(registryAddress, &tokenAddress, &partnerAddr)
		fmt.Println()
		log.Info("State.OurState.GetBalance: ", chanState.OurState.GetGasBalance())
		log.Info("State.OurState.ContractBalance: ", chanState.OurState.ContractBalance)
		if chanState.OurState.BalanceProof != nil {
			log.Info("State.OurState.BalanceProof.LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
		}

		log.Info("State.PartnerState.GetBalance: ", chanState.PartnerState.GetGasBalance())
		log.Info("State.PartnerState.ContractBalance: ", chanState.PartnerState.ContractBalance)
		if chanState.PartnerState.BalanceProof != nil {
			log.Info("State.PartnerState.BalanceProof.LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
		}
	}

}
