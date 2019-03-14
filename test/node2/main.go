package main

import (
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

var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3001",
	//MappingAddress: "10.0.1.105:3000",
	Protocol:      "tcp",
	RevealTimeout: "1000",
}

func main() {
	if len(os.Args) != 2 {
		log.Error("Param error")
		return
	}

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

	err = channel.StartService()
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Info("[StartService]")
	registryAddress := common.PaymentNetworkID(utils.MicroPayContractAddress)

	tokenAddress := common.TokenAddress(ong.ONG_CONTRACT_ADDRESS)
	partnerAddress, err := chaincomm.AddressFromBase58(os.Args[1])
	if err != nil {
		log.Error(err)
		return
	}
	partnerAddr := common.Address(partnerAddress)

	chanId := channel.Service.OpenChannel(tokenAddress, common.Address(partnerAddress))
	log.Info("[OpenChannel] ChanId: ", chanId)

	err = channel.Service.SetTotalChannelDeposit(tokenAddress, common.Address(partnerAddress), 20000)
	if err != nil {
		log.Error("[SetTotalChannelDeposit]: ", err.Error())
		return
	}

	for i := 0; ; i++ {
		chanState := channel.Service.GetChannel(registryAddress, &tokenAddress, &partnerAddr)
		log.Info("==============================")
		if chanState != nil {
			if chanState.OurState != nil {
				log.Info("State.OurState.GetBalance: ", chanState.OurState.GetGasBalance())
				log.Info("State.OurState.ContractBalance: ", chanState.OurState.ContractBalance)
				if chanState.OurState.BalanceProof != nil {
					log.Info("State.OurState.BalanceProof.LockedAmount: ", chanState.OurState.BalanceProof.LockedAmount)
				}
			} else {
				log.Error("[GetChannel] chanState.OurState is nil")
			}

			if chanState.PartnerState != nil {
				log.Info("State.PartnerState.GetBalance: ", chanState.PartnerState.GetGasBalance())
				log.Info("State.PartnerState.ContractBalance: ", chanState.PartnerState.ContractBalance)
				if chanState.PartnerState.BalanceProof != nil {
					log.Info("State.PartnerState.BalanceProof.LockedAmount: ", chanState.PartnerState.BalanceProof.LockedAmount)
				}
			} else {
				log.Error("[GetChannel] chanState.OurState is nil")
			}
		} else {
			log.Error("[GetChannel] chanState is nil")
		}
		time.Sleep(3 * time.Second)
	}

}
