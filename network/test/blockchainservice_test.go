/**
 * Description:
 * Author: Yihen.Liu
 * Create: 2018-11-05
 */
package test

import (
	"fmt"
	"testing"

	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/contract"
	"github.com/oniio/oniChannel/network/rpc"
)

var (
	WALLET_PATH = "./wallet.dat"
	WALLET_PWD  = []byte("123456")
)

func TestBlockNumber(t *testing.T) {
	blockchainService := network.NewBlockchainService("", WALLET_PATH, WALLET_PWD)
	height, _ := blockchainService.BlockNumber()
	fmt.Println("Get current block number: ", height)
}

func TestGetBlock(t *testing.T) {
	blockchainService := network.NewBlockchainService("", WALLET_PATH, WALLET_PWD)
	block, _ := blockchainService.GetBlock(uint32(2))
	fmt.Println(block.ToArray())
}

func TestNextBlock(t *testing.T) {
	blockchainService := network.NewBlockchainService("", WALLET_PATH, WALLET_PWD)
	currentHeight := blockchainService.NextBlock()
	fmt.Println("Current height:", currentHeight)
}

func TestRPCServerRun(t *testing.T) {

	rpcClient := rpc.NewRpcClient("http://localhost:20336")
	height, _ := rpcClient.GetCurrentBlockHeight()
	/*
		_, err := ontSdk.OpenWallet("./wallet.dat")
		if err != nil {
			return
		}
	*/

	fmt.Printf("block height:%s\n", height)
	url := "http://localhost:20336"

	blockchainService := network.NewBlockchainService(url, "./wallet.dat", []byte("123456")).UsingContract(contract.MPAY_CONTRACT_ADDRESS)
	tx := blockchainService.ContractManager.Contract.BuildRegisterPaymentEndPointTx(0, 20000, []byte("127.0.0.1"), []byte("15566"), contract.MPAY_CONTRACT_ADDRESS)
	res, err := blockchainService.Client.SendRawTransaction(tx, false)
	fmt.Printf("res:%s, err:%s\n", res, err)
	//ontSdk.Native.MPay.RegisterPaymentEndPoint()
	//fmt.Println("Hello world")
}
