/**
 * Description:
 * Author: Yihen.Liu
 * Create: 2018-10-31
 */
package contract

import (
	"github.com/oniio/oniChannel/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/smartcontract/service/native/micropayment"
)

var (
	MPAY_CONTRACT_ADDRESS, _ = common.AddressFromHexString("0900000000000000000000000000000000000000")
)

var (
	MPAY_CONTRACT_VERSION = byte(0)
)

type MPayContract struct {
	version         byte
	ContractAddress common.Address
	Account         *account.Account
}

func (this *MPayContract) GetContractAddress() common.Address {
	return this.ContractAddress
}

func (this *MPayContract) BuildRegisterPaymentEndPointTx(gasPrice, gasLimit uint64, ip, port []byte, regAccount common.Address) *types.Transaction {
	participant := &struct {
		Ip      []byte
		Port    []byte
		Account common.Address
	}{ip, port, regAccount}

	tx, _ := NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.version,
		this.ContractAddress,
		micropayment.MP_ENDPOINT_REGISTRY,
		[]interface{}{participant})
	err := account.SignToTransaction(tx, this.Account)
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
	}

	retTx, err := tx.IntoImmutable()
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
	}
	return retTx
}

func (this *MPayContract) BuildOpenChannelTx(gasPrice, gasLimit uint64, wallet1Addr, wallet2Addr common.Address, blockHeight uint64) *types.Transaction {
	participant := &struct {
		wallet1 common.Address
		wallet2 common.Address
		height  uint64
	}{wallet1Addr, wallet2Addr, blockHeight}
	tx, _ := NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		this.version,
		this.ContractAddress,
		micropayment.MP_ENDPOINT_REGISTRY,
		[]interface{}{participant})
	err := account.SignToTransaction(tx, this.Account)
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
	}
	retTx, err := tx.IntoImmutable()
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
	}
	return retTx
}

func (this *MPayContract) FilterAllLogs(toBlock uint32) {

}

func (this *MPayContract) FilterNewLogs(fromBlock, toBlock uint32) {

}
