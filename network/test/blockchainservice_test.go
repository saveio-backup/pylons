package test

import (
	"fmt"
	"testing"

	"github.com/oniio/oniChain-go-sdk/wallet"
	"github.com/oniio/oniChannel/network"
)

var (
	WALLET_PATH       = "./wallet.dat"
	WALLET_PWD        = []byte("123456")
	blockchainService *network.BlockchainService
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
	url := "http://localhost:20336"
	blockchainService = network.NewBlockchainService("rpc", url, account)
}

func TestBlockHeight(t *testing.T) {
	height, _ := blockchainService.BlockHeight()
	fmt.Println("Get current block number: ", height)
}

func TestGetBlock(t *testing.T) {
	block, _ := blockchainService.GetBlock(uint32(2))
	fmt.Println(block.ToArray())
}

func TestNextBlock(t *testing.T) {
	currentHeight := blockchainService.NextBlock()
	fmt.Println("Current height:", currentHeight)
}

func TestRPCServerRun(t *testing.T) {
	tx,err := blockchainService.ChannelClient.RegisterPaymentEndPoint([]byte("127.0.0.1"), []byte("15566"),blockchainService.GetAccount().Address)
	fmt.Printf("res:%s, err:%s\n", tx[:], err)
}
