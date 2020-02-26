package test

import (
	"fmt"
	"testing"

	"github.com/saveio/pylons/network"
	"github.com/saveio/themis-go-sdk/wallet"
)

var (
	WALLET_PATH       = "./wallet.dat"
	WALLET_PWD        = []byte("123456")
	blockchainService *network.BlockChainService
)

func init() {
	wallet, err := wallet.OpenWallet(WALLET_PATH)
	if err != nil {
		fmt.Println("wallet.Open error:", err)
		return
	}
	account, err := wallet.GetDefaultAccount(WALLET_PWD)
	if err != nil {
		fmt.Println("GetDefaultAccount error:", err)
		return
	}
	url := []string{"http://localhost:20336"}
	blockchainService = network.NewBlockChainService("rpc", url, account)
}

func TestBlockHeight(t *testing.T) {
	height, _ := blockchainService.BlockHeight()
	fmt.Println("Get current block number: ", height)
}

func TestGetBlock(t *testing.T) {
	block, _ := blockchainService.GetBlock(uint32(2))
	fmt.Println(block.ToArray())
}
