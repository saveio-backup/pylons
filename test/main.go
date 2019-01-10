package main

import (
	"fmt"

	"github.com/oniio/dsp-go-sdk/chain/wallet"
	"github.com/oniio/oniChain/common/log"
	ch "github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/typing"
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
	log.InitLog(3, log.PATH, log.Stdout)
	wallet, err := wallet.OpenWallet(WALLET_PATH)
	if err != nil {
		fmt.Printf("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount(WALLET_PWD)
	if err != nil {
		fmt.Printf("GetDefaultAccount error:%s\n", err)
	}
	channel, err := ch.NewChannel(testConfig, account)
	if err != nil {
		fmt.Println(err)
		return
	}

	channel.StartService()

	/*
		var registryAddress typing.PaymentNetworkID
		var tokenAddress typing.TokenAddress
		var partnerAddress typing.Address
		var settleTimeout typing.BlockTimeout
		var retryTimeout typing.NetworkTimeout

		channel.Api.ChannelOpen(registryAddress,
			tokenAddress, partnerAddress,
			settleTimeout, retryTimeout)

	*/
	var target typing.Address
	channel.Service.DirectTransferAsync(1, target, 111)
}
