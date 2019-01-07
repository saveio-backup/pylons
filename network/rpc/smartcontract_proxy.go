package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	httpcom "github.com/oniio/oniChain/http/base/common"
	"github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	"github.com/oniio/oniChain/vm/neovm/types"
	"github.com/oniio/oniChannel/network/contract"
	daseinNetUtils "github.com/oniio/oniChannel/network/utils"
)

// Micropaymenet contract const
const (
	MPAY_CONTRACT_VERSION = byte(0)
	DEFALUT_GAS_PRICE     = 0
	DEFAULT_GAS_LIMIT     = 30000
)

type ContractProxy struct {
	JsonrpcClient *RpcClient
	gasPrice      uint64
	gasLimit      uint64
	version       byte
}

func NewContractProxy(rpcClient *RpcClient) *ContractProxy {
	rand.Seed(time.Now().UnixNano())
	return &ContractProxy{
		JsonrpcClient: rpcClient,
		gasPrice:      DEFALUT_GAS_PRICE,
		gasLimit:      DEFAULT_GAS_LIMIT,
		version:       MPAY_CONTRACT_VERSION,
	}
}

func (self *ContractProxy) SetGasPrice(gasPrice uint64) {
	self.gasPrice = gasPrice
}

func (self *ContractProxy) SetGasLimit(gasLimit uint64) {
	self.gasLimit = gasLimit
}

func (self *ContractProxy) SetVersion(version byte) {
	self.version = version
}

func (self *ContractProxy) GetContractAddress() common.Address {
	return utils.MicroPayContractAddress
}

func (self *ContractProxy) GetVersion() byte {
	return self.version
}

func (self *ContractProxy) RegisterPaymentEndPoint(ip, port []byte, regAccount common.Address) ([]byte, error) {
	params := &micropayment.NodeInfo{
		WalletAddr: regAccount,
		IP:         ip,
		Port:       port,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_ENDPOINT_REGISTRY,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)

	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (self *ContractProxy) OpenChannel(wallet1Addr, wallet2Addr common.Address, blockHeight uint64) ([]byte, error) {
	params := &micropayment.OpenChannelInfo{
		Participant1WalletAddr: wallet1Addr,
		Participant2WalletAddr: wallet2Addr,
		SettleBlockHeight:      blockHeight,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_OPEN_CHANNEL,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (self *ContractProxy) SetTotalDeposit(channelId uint64, participantWalletAddr common.Address, partnerWalletAddr common.Address, deposit uint64) ([]byte, error) {
	params := &micropayment.SetTotalDepositInfo{
		ChannelID:             channelId,
		ParticipantWalletAddr: participantWalletAddr,
		PartnerWalletAddr:     partnerWalletAddr,
		SetTotalDeposit:       deposit,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_SET_TOTALDEPOSIT,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}
func (self *ContractProxy) SetTotalWithdraw(channelID uint64, participant, partner common.Address, totalWithdraw uint64, participantSig, participantPubKey, partnerSig, partnerPubKey []byte) ([]byte, error) {
	params := &micropayment.WithDraw{
		ChannelID:         channelID,
		Participant:       participant,
		Partner:           partner,
		TotalWithdraw:     totalWithdraw,
		ParticipantSig:    participantSig,
		ParticipantPubKey: participantPubKey,
		PartnerSig:        partnerSig,
		PartnerPubKey:     partnerPubKey,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_SET_TOTALWITHDRAW,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}
func (self *ContractProxy) CooperativeSettle(channelID uint64, participant1Address common.Address, participant1Balance uint64, participant2Address common.Address, participant2Balance uint64, participant1Signature, participant1PubKey, participant2Signature, participant2PubKey []byte) ([]byte, error) {
	params := &micropayment.CooperativeSettleInfo{
		ChannelID:             channelID,
		Participant1Address:   participant1Address,
		Participant1Balance:   participant1Balance,
		Participant2Address:   participant2Address,
		Participant2Balance:   participant2Balance,
		Participant1Signature: participant1Signature,
		Participant1PubKey:    participant1PubKey,
		Participant2Signature: participant2Signature,
		Participant2PubKey:    participant2PubKey,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_COOPERATIVESETTLE,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}
func (self *ContractProxy) CloseChannel(channelID uint64, participantAddress, partnerAddress common.Address, balanceHash []byte, nonce uint64, additionalHash, partnerSignature, partnerPubKey []byte) ([]byte, error) {
	params := &micropayment.CloseChannelInfo{
		ChannelID:          channelID,
		ParticipantAddress: participantAddress,
		PartnerAddress:     partnerAddress,
		BalanceHash:        balanceHash,
		Nonce:              nonce,
		AdditionalHash:     additionalHash,
		PartnerSignature:   partnerSignature,
		PartnerPubKey:      partnerPubKey,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_CLOSE_CHANNEL,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}
func (self *ContractProxy) RegisterSecret(secret []byte) ([]byte, error) {
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_SECRET_REG,
		[]interface{}{secret},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}
func (self *ContractProxy) RegisterSecretBatch(secrets []byte) ([]byte, error) {

	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_SECRET_REG_BATCH,
		[]interface{}{secrets},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (self *ContractProxy) GetSecretRevealBlockHeight(secretHash []byte) (uint64, error) {
	tx, err := contract.PreExecNativeContractRawTransaction(
		self.version,
		self.GetContractAddress(),
		micropayment.MP_GET_SECRET_REVEAL_BLOCKHEIGHT,
		[]interface{}{secretHash},
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return 0, err
	}

	data, err := self.JsonrpcClient.SendRawTransaction(tx, true)
	if err != nil {
		log.Errorf("Preexcute contract error, Msg:", err.Error())
		return 0, err
	}
	preResult := &daseinNetUtils.PreExecResult{}
	err = json.Unmarshal(data, &preResult)
	if err != nil {
		return 0, err
	}
	buf, err := preResult.Result.ToByteArray()
	if err != nil {
		return 0, err
	}
	height, err := strconv.ParseUint(hex.EncodeToString(common.ToArrayReverse(buf)), 16, 64)
	if err != nil {
		return 0, err
	}
	return height, nil
}
func (self *ContractProxy) UpdateNonClosingBalanceProof(chanID uint64, closeParticipant, nonCloseParticipant common.Address, balanceHash []byte, nonce uint64, additionalHash, closeSignature, nonCloseSignature, closePubKey, nonClosePubKey []byte) ([]byte, error) {
	params := &micropayment.UpdateNonCloseBalanceProof{
		ChanID:              chanID,
		CloseParticipant:    closeParticipant,
		NonCloseParticipant: nonCloseParticipant,
		BalanceHash:         balanceHash,
		Nonce:               nonce,
		AdditionalHash:      additionalHash,
		CloseSignature:      closeSignature,
		NonCloseSignature:   nonCloseSignature,
		ClosePubKey:         closePubKey,
		NonClosePubKey:      nonClosePubKey,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_UPDATE_NONCLOSING_BPF,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (self *ContractProxy) SettleChannel(chanID uint64, participant1 common.Address, p1TransferredAmount, p1LockedAmount uint64, p1LocksRoot []byte, participant2 common.Address, p2TransferredAmount, p2LockedAmount uint64, p2LocksRoot []byte) ([]byte, error) {
	params := &micropayment.SettleChannelInfo{
		ChanID:              chanID,
		Participant1:        participant1,
		P1TransferredAmount: p1TransferredAmount,
		P1LockedAmount:      p1LockedAmount,
		P1LocksRoot:         p1LocksRoot,
		Participant2:        participant2,
		P2TransferredAmount: p2TransferredAmount,
		P2LockedAmount:      p2LockedAmount,
		P2LocksRoot:         p2LocksRoot,
	}
	signer := &account.Account{
		PrivateKey: self.JsonrpcClient.Account.PrivateKey,
		PublicKey:  self.JsonrpcClient.Account.PublicKey,
		Address:    self.JsonrpcClient.Account.Address,
		SigScheme:  self.JsonrpcClient.Account.SigScheme,
	}
	tx, err := contract.NewNativeContractRawTransaction(
		self.gasPrice,
		self.gasLimit,
		self.version,
		self.GetContractAddress(),
		micropayment.MP_SETTLE_CHANNEL,
		[]interface{}{params},
		signer,
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	v, err := daseinNetUtils.GetUint256(data)
	if err != nil {
		return nil, err
	}
	return v[:], nil
}

func (self *ContractProxy) GetChannelInfo(channelID uint64, participant1, participant2 common.Address) (*micropayment.ChannelInfo, error) {
	params := &micropayment.GetChanInfo{
		ChannelID:    channelID,
		Participant1: participant1,
		Participant2: participant2,
	}

	tx, err := contract.PreExecNativeContractRawTransaction(
		self.version,
		self.GetContractAddress(),
		micropayment.MP_GET_CHANNELINFO,
		[]interface{}{params},
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, true)
	if err != nil {
		log.Errorf("Preexcute contract error, Msg:", err.Error())
		return nil, err
	}
	preResult := &daseinNetUtils.PreExecResult{}
	err = json.Unmarshal(data, &preResult)
	if err != nil {
		return nil, err
	}
	buf, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	channelInfo := &micropayment.ChannelInfo{}
	source := common.NewZeroCopySource(buf)
	channelInfo.SettleBlockHeight, err = utils.DecodeVarUint(source)
	if err != nil {
		return nil, err
	}
	channelInfo.ChannelState, err = utils.DecodeVarUint(source)
	if err != nil {
		return nil, err
	}
	return channelInfo, nil
}

func (self *ContractProxy) GetChannelParticipantInfo(channelID uint64, participant1, participant2 common.Address) (*micropayment.Participant, error) {
	params := &micropayment.GetChanInfo{
		ChannelID:    channelID,
		Participant1: participant1,
		Participant2: participant2,
	}

	tx, err := contract.PreExecNativeContractRawTransaction(
		self.version,
		self.GetContractAddress(),
		micropayment.MP_GET_CHANNEL_PARTICIPANTINFO,
		[]interface{}{params},
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, true)
	if err != nil {
		log.Errorf("Preexcute contract error, Msg:", err.Error())
		return nil, err
	}
	preResult := &daseinNetUtils.PreExecResult{}
	err = json.Unmarshal(data, &preResult)
	if err != nil {
		return nil, err
	}
	buf, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	source := common.NewZeroCopySource(buf)
	participant := &micropayment.Participant{}
	err = participant.Deserialization(source)
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (self *ContractProxy) GetOntBalance(address common.Address) (uint64, error) {
	result, err := self.JsonrpcClient.sendRpcRequest("getbalance", []interface{}{address.ToBase58()})
	if err != nil {
		return 0, err
	}
	balance := &httpcom.BalanceOfRsp{}
	err = json.Unmarshal(result, balance)
	if err != nil {
		return 0, fmt.Errorf("json.Unmarshal error:%s", err)
	}
	return strconv.ParseUint(balance.Ont, 10, 64)
}

func (self *ContractProxy) GetOngBalance(address common.Address) (uint64, error) {
	result, err := self.JsonrpcClient.sendRpcRequest("getbalance", []interface{}{address.ToBase58()})
	if err != nil {
		return 0, err
	}
	balance := &httpcom.BalanceOfRsp{}
	err = json.Unmarshal(result, balance)
	if err != nil {
		return 0, fmt.Errorf("json.Unmarshal error:%s", err)
	}
	return strconv.ParseUint(balance.Ong, 10, 64)
}

// GetEndpointByAddress get endpoint by user wallet address
func (self *ContractProxy) GetEndpointByAddress(nodeAddress common.Address) (*micropayment.NodeInfo, error) {
	tx, err := contract.PreExecNativeContractRawTransaction(
		self.version,
		self.GetContractAddress(),
		micropayment.MP_FIND_ENDPOINT,
		[]interface{}{nodeAddress},
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return nil, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, true)
	if err != nil {
		log.Errorf("Preexcute contract error, Msg:", err.Error())
		return nil, err
	}
	preResult := &daseinNetUtils.PreExecResult{}
	err = json.Unmarshal(data, &preResult)
	if err != nil {
		return nil, err
	}
	buf, err := preResult.Result.ToByteArray()
	if err != nil {
		return nil, err
	}
	nodeInfo := &micropayment.NodeInfo{}
	err = nodeInfo.Deserialize(bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	return nodeInfo, nil
}

func (self *ContractProxy) GetChannelIdentifier(participant1WalletAddr, participant2WalletAddr common.Address) (uint64, error) {
	params := &micropayment.GetChannelId{
		Participant1WalletAddr: participant1WalletAddr,
		Participant2WalletAddr: participant2WalletAddr,
	}
	tx, err := contract.PreExecNativeContractRawTransaction(
		self.version,
		self.GetContractAddress(),
		micropayment.MP_GET_CHANNELID,
		[]interface{}{params},
	)
	if err != nil {
		log.Errorf("Construct native invoke tx error, Msg:", err.Error())
		return 0, err
	}
	data, err := self.JsonrpcClient.SendRawTransaction(tx, true)
	if err != nil {
		log.Errorf("Preexcute contract error, Msg:", err.Error())
		return 0, err
	}
	preResult := &daseinNetUtils.PreExecResult{}
	err = json.Unmarshal(data, &preResult)
	if err != nil {
		return 0, err
	}
	buf, err := preResult.Result.ToByteArray()
	if err != nil {
		return 0, err
	}
	fmt.Printf("buf:%v\n", buf)
	valStr := fmt.Sprintf("%s", types.BigIntFromBytes(buf))
	return strconv.ParseUint(valStr, 10, 64)

}

// func (this *MPayContract) FilterAllLogs(toBlock uint32) {

// }

// func (this *MPayContract) FilterNewLogs(fromBlock, toBlock uint32) {

// }
