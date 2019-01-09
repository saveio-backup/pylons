package contract

import (
	"fmt"

	sdk_utils "github.com/oniio/dsp-go-sdk/chain/utils"
	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/core/types"
	"github.com/oniio/oniChain/smartcontract/service/native/micropayment"
)

var (
	MPAY_CONTRACT_VERSION    = byte(0)
	MPAY_CONTRACT_ADDRESS, _ = common.AddressFromHexString("0900000000000000000000000000000000000000")
)

type ContractManager struct {
	version         byte
	ContractAddress common.Address
	Account         *account.Account
}

func (this *ContractManager) GetContractAddress() common.Address {
	return this.ContractAddress
}

func (this *ContractManager) BuildRegisterPaymentEndPointTx(gasPrice, gasLimit uint64, ip, port []byte, regAccount common.Address) *types.Transaction {
	participant := &struct {
		Ip      []byte
		Port    []byte
		Account common.Address
	}{ip, port, regAccount}

	tx, _ := sdk_utils.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.version,
		this.ContractAddress,
		micropayment.MP_ENDPOINT_REGISTRY,
		[]interface{}{participant})
	err := sdk_utils.SignToTransaction(tx, this.Account)
	if err != nil {
		fmt.Println("Sign Transaction Error, Msg:", err.Error())
	}

	retTx, err := tx.IntoImmutable()
	if err != nil {
		fmt.Println("Sign Transaction Error, Msg:", err.Error())
	}
	return retTx
}

func (this *ContractManager) BuildOpenChannelTx(gasPrice, gasLimit uint64, wallet1Addr, wallet2Addr common.Address, blockHeight uint64) *types.Transaction {
	participant := &struct {
		wallet1 common.Address
		wallet2 common.Address
		height  uint64
	}{wallet1Addr, wallet2Addr, blockHeight}
	tx, _ := sdk_utils.NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.version,
		this.ContractAddress,
		micropayment.MP_ENDPOINT_REGISTRY,
		[]interface{}{participant})
	err := sdk_utils.SignToTransaction(tx, this.Account)
	if err != nil {
		fmt.Println("Sign Transaction Error, Msg:", err.Error())
	}
	retTx, err := tx.IntoImmutable()
	if err != nil {
		fmt.Println("Sign Transaction Error, Msg:", err.Error())
	}
	return retTx
}

func (this *ContractManager) FilterAllLogs(toBlock uint32) {

}

func (this *ContractManager) FilterNewLogs(fromBlock, toBlock uint32) {

}
