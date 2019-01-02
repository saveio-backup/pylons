package contract

import (
	"fmt"
	"github.com/oniio/oniChannel/account"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/payload"
	"github.com/ontio/ontology/core/types"
	httpcom "github.com/ontio/ontology/http/base/common"
	"math/rand"
)

type ContractManager struct {
	contractAddr common.Address
	MPayContract
}

func NewNativeInvokeTransaction(
	gasPrice,
	gasLimit uint64,
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
) (*types.MutableTransaction, error) {
	if params == nil {
		params = make([]interface{}, 0, 1)
	}
	//Params cannot empty, if params is empty, fulfil with empty string
	if len(params) == 0 {
		params = append(params, "")
	}
	invokeCode, err := httpcom.BuildNativeInvokeCode(contractAddress, version, method, params)
	if err != nil {
		return nil, fmt.Errorf("BuildNativeInvokeCode error:%s", err)
	}
	return NewInvokeTransaction(gasPrice, gasLimit, invokeCode), nil
}

func InvokeNativeContract(
	gasPrice,
	gasLimit uint64,
	singer *account.Account,
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
) *types.MutableTransaction {
	tx, _ := NewNativeInvokeTransaction(gasPrice, gasLimit, version, contractAddress, method, params)
	_ = account.SignToTransaction(tx, singer)
	return tx
}

func PreExecInvokeNativeContract(
	contractAddress common.Address,
	version byte,
	method string,
	params []interface{},
) *types.MutableTransaction {
	tx, _ := NewNativeInvokeTransaction(0, 0, version, contractAddress, method, params)
	return tx
}

func NewNativeContractRawTransaction(
	gasPrice, gasLimit uint64,
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
	signer *account.Account,
) (*types.Transaction, error) {
	tx, err := NewNativeInvokeTransaction(
		gasPrice,
		gasLimit,
		version,
		contractAddress,
		method,
		params)
	if err != nil {
		return nil, err
	}
	err = account.SignToTransaction(tx, signer)
	if err != nil {
		return nil, err
	}
	retTx, err := tx.IntoImmutable()
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
		return nil, err
	}
	return retTx, nil
}

func PreExecNativeContractRawTransaction(
	version byte,
	contractAddress common.Address,
	method string,
	params []interface{},
) (*types.Transaction, error) {
	tx, err := NewNativeInvokeTransaction(
		0,
		0,
		version,
		contractAddress,
		method,
		params)
	if err != nil {
		return nil, err
	}
	retTx, err := tx.IntoImmutable()
	if err != nil {
		log.Errorf("Sign Transaction Error, Msg:", err.Error())
		return nil, err
	}
	return retTx, nil
}

//NewInvokeTransaction return smart contract invoke transaction
func NewInvokeTransaction(gasPrice, gasLimit uint64, invokeCode []byte) *types.MutableTransaction {
	invokePayload := &payload.InvokeCode{
		Code: invokeCode,
	}
	tx := &types.MutableTransaction{
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		TxType:   types.Invoke,
		Nonce:    rand.Uint32(),
		Payload:  invokePayload,
		Sigs:     make([]types.Sig, 0, 0),
	}
	return tx
}
