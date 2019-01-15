package main

import (
	"github.com/oniio/oniChain-go-sdk/wallet"
	"github.com/oniio/oniChain/common/log"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/common"
)

var (
	WALLET_PATH = "./wallet.dat"
	WALLET_PWD  = []byte("123")
)
var testConfig = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURL:  "http://127.0.0.1:20336",
	ListenAddress: "127.0.0.1:3000",
	Protocol:      "tcp",
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
	/*
		var registryAddress common.PaymentNetworkID
		var tokenAddress common.TokenAddress
		var partnerAddress common.Address
		var settleTimeout common.BlockTimeout
		var retryTimeout common.NetworkTimeout

		channel.Api.ChannelOpen(registryAddress,
			tokenAddress, partnerAddress,
			settleTimeout, retryTimeout)

	*/
	var target common.Address
	channel.Service.DirectTransferAsync(1, target, 111)
}
