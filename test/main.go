package main

import (
	"fmt"

	"github.com/github.com/ontio/ontology/account"
	"github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/typing"
)

var testConfig = &nimbus.NimbusConfig{
	ChainNodeURL:  "http://127.0.0.1:20336",
	WalletPath:    "wallet.dat",
	Password:      []byte(string("123")),
	ListenAddress: "127.0.0.1:3000",
	Protocol:      "tcp",
}

func main() {
	wallet, err := account.OpenWallet(testConfig.WalletPath)
	if err != nil {
		fmt.Printf("wallet.Open error:%s\n", err)
	}
	account, err := wallet.GetDefaultAccount(testConfig.Password)
	if err != nil {
		fmt.Printf("GetDefaultAccount error:%s\n", err)
	}
	nimbus, err := nimbus.NewNimbus(testConfig, account)
	if err != nil {
		fmt.Println(err)
		return
	}

	nimbus.StartService()

	/*
		var registryAddress typing.PaymentNetworkID
		var tokenAddress typing.TokenAddress
		var partnerAddress typing.Address
		var settleTimeout typing.BlockTimeout
		var retryTimeout typing.NetworkTimeout

		nimbus.Api.ChannelOpen(registryAddress,
			tokenAddress, partnerAddress,
			settleTimeout, retryTimeout)

	*/
	var target typing.Address
	nimbus.Api.DirectTransferAsync(1, target, 111)
}
