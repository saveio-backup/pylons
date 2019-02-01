package transfer

import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/utils/jsonext"
)

type SecretHashToLock map[common.SecretHash]HashTimeLockState
type SecretHashToPartialUnlockProof map[common.SecretHash]UnlockPartialProofState
type QueueIdsToQueuesType map[QueueIdentifier][]Event

const ChannelStateClosed string = "closed"
const ChannelStateClosing string = "waiting_for_close"
const ChannelStateOpened string = "opened"
const ChannelStateSettled string = "settled"
const ChannelStateSettling string = "waiting_for_settle"
const ChannelStateUnusable string = "channel_unusable"

const TxnExecSucc string = "success"
const TxnExecFail string = "failure"

const NetworkUnknown string = "unknown"
const NetworkUnreachable string = "unreachable"
const NetworkReachable string = "reachable"

func GetMsgID() common.MessageID {
	rand.Seed(time.Now().UnixNano())
	messageId := rand.Int63n(math.MaxInt64)
	return common.MessageID(messageId)
}

type TransactionOrderHeap []TransactionOrder

func (h TransactionOrderHeap) Len() int {
	return len(h)
}

func (h TransactionOrderHeap) Less(i, j int) bool {
	return h[i].BlockHeight < h[j].BlockHeight
}

func (h TransactionOrderHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *TransactionOrderHeap) Push(v TransactionOrder) {
	*h = append(*h, v)
	sort.Sort(*h)
}

func (h *TransactionOrderHeap) Pop() TransactionOrder {
	result := (*h)[0]
	*h = (*h)[1:]
	sort.Sort(*h)
	return result
}

type InitiatorTask struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ManagerState           State
}

type MediatorTask struct {
	TokenNetworkIdentifier common.TokenNetworkID
	MediatorState          State
}

type TargetTask struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	TargetState            State
}

type ChainState struct {
	BlockHeight                  common.BlockHeight
	ChainId                      common.ChainID
	IdentifiersToPaymentnetworks map[common.PaymentNetworkID]*PaymentNetworkState
	NodeAddressesToNetworkstates map[common.Address]string
	Address                      common.Address
	PaymentMapping               PaymentMappingState
	PendingTransactions          []Event
	QueueIdsToQueues             QueueIdsToQueuesType
}

func NewChainState() *ChainState {
	result := new(ChainState)

	result.IdentifiersToPaymentnetworks = make(map[common.PaymentNetworkID]*PaymentNetworkState)
	result.NodeAddressesToNetworkstates = make(map[common.Address]string)
	result.PaymentMapping.SecrethashesToTask = make(map[common.SecretHash]State)
	result.PendingTransactions = []Event{}
	result.QueueIdsToQueues = make(QueueIdsToQueuesType)

	return result
}

//Adjust PaymentNetworkState in IdentifiersToPaymentnetworks
func (self *ChainState) AdjustChainState() {
	for _, v := range self.IdentifiersToPaymentnetworks {
		v.AdjustPaymentNetworkState()
	}

	return
}

func DeepCopy(src State) *ChainState {
	var result *ChainState

	if src == nil {
		return nil
	} else if chainState, ok := src.(*ChainState); ok {
		serializedData, _ := jsonext.Marshal(chainState)
		res, _ := jsonext.UnmarshalExt(serializedData, nil, CreateObjectByClassId)
		result, _ = res.(*ChainState)
		result.AdjustChainState()
	}

	return result
}

type PaymentNetworkState struct {
	Address                         common.PaymentNetworkID
	TokenIdentifiersToTokenNetworks map[common.TokenNetworkID]*TokenNetworkState
	tokenAddressesToTokenNetworks   map[common.TokenAddress]*TokenNetworkState
}

func (self *PaymentNetworkState) GetAddress() common.Address {
	return common.Address(self.Address)
}

func NewPaymentNetworkState() *PaymentNetworkState {
	result := new(PaymentNetworkState)

	result.TokenIdentifiersToTokenNetworks = make(map[common.TokenNetworkID]*TokenNetworkState)
	result.tokenAddressesToTokenNetworks = make(map[common.TokenAddress]*TokenNetworkState)

	return result
}

//Adjust TokenNetworkState in TokenIdentifiersToTokenNetworks
//rebuild tokenAddressesToTokenNetworks
func (self *PaymentNetworkState) AdjustPaymentNetworkState() {
	self.tokenAddressesToTokenNetworks = make(map[common.TokenAddress]*TokenNetworkState)

	for _, v := range self.TokenIdentifiersToTokenNetworks {
		v.AdjustTokenNetworkState()
		self.tokenAddressesToTokenNetworks[v.TokenAddress] = v
	}

	return
}

type TokenNetworkState struct {
	Address                      common.TokenNetworkID
	TokenAddress                 common.TokenAddress
	NetworkGraph                 *TokenNetworkGraphState
	ChannelIdentifiersToChannels map[common.ChannelID]*NettingChannelState
	partnerAddressesToChannels   map[common.Address]map[common.ChannelID]*NettingChannelState
}

func NewTokenNetworkState() *TokenNetworkState {
	result := new(TokenNetworkState)
	result.ChannelIdentifiersToChannels = make(map[common.ChannelID]*NettingChannelState)
	result.partnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)

	return result
}

func (self *TokenNetworkState) GetTokenAddress() common.TokenAddress {
	return self.TokenAddress
}

//rebuild partnerAddressesToChannels
func (self *TokenNetworkState) AdjustTokenNetworkState() {
	self.partnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
	for _, v := range self.ChannelIdentifiersToChannels {
		if self.partnerAddressesToChannels[v.PartnerState.Address] == nil {
			self.partnerAddressesToChannels[v.PartnerState.Address] = make(map[common.ChannelID]*NettingChannelState)
		}
		self.partnerAddressesToChannels[v.PartnerState.Address][v.Identifier] = v
	}

	return
}

type TokenNetworkGraphState struct {
	//[TODO] introduce Graph struct!
}

type PaymentMappingState struct {
	SecrethashesToTask map[common.SecretHash]State
}

type RouteState struct {
	NodeAddress       common.Address
	ChannelIdentifier common.ChannelID
}

type BalanceProofUnsignedState struct {
	Nonce                  common.Nonce
	TransferredAmount      common.TokenAmount
	LockedAmount           common.TokenAmount
	LocksRoot              common.Locksroot
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	ChainId                common.ChainID
}

type BalanceProofSignedState struct {
	Nonce                  common.Nonce
	TransferredAmount      common.TokenAmount
	LockedAmount           common.TokenAmount
	LocksRoot              common.Locksroot
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	MessageHash            common.Keccak256
	Signature              common.Signature
	Sender                 common.Address
	ChainId                common.ChainID
	PublicKey              common.PubKey
}

type HashTimeLockState struct {
	Amount     common.TokenAmount
	Expiration common.BlockHeight
	Secrethash common.Keccak256
	Encoded    []byte
	Lockhash   common.LockHash
}

type UnlockPartialProofState struct {
	Lock   HashTimeLockState
	Secret common.Secret
}

type UnlockProofState struct {
	MerkelProof []common.Keccak256
	LockEncoded []byte
	Secret      common.Secret
}

type TransactionExecutionStatus struct {
	StartedBlockHeight  common.BlockHeight
	FinishedBlockHeight common.BlockHeight
	Result              string
}

type MerkleTreeState struct {
	Layers [][]common.Keccak256
}

func (self *MerkleTreeState) init() {

	self.Layers = append(self.Layers, []common.Keccak256{})
	self.Layers = append(self.Layers, []common.Keccak256{})

	emptyRoot := common.Keccak256{}
	self.Layers[1] = append(self.Layers[1], emptyRoot)
}

type NettingChannelEndState struct {
	Address                            common.Address
	ContractBalance                    common.TokenAmount
	SecretHashesToLockedlocks          SecretHashToLock
	SecretHashesToUnlockedlocks        SecretHashToPartialUnlockProof
	SecretHashesToOnchainUnlockedlocks SecretHashToPartialUnlockProof
	Merkletree                         *MerkleTreeState
	BalanceProof                       *BalanceProofSignedState
}

func (self *NettingChannelEndState) GetContractBalance() common.TokenAmount {
	return self.ContractBalance
}

func (self *NettingChannelEndState) GetGasBalance() common.TokenAmount {
	var amount common.TokenAmount

	// the balanceProof could be nil in following cases:
	// outState.BalanceProof is nil when no payment has been sent
	// partnetState.BalanceProof is nil when no payment has been received
	if self.BalanceProof != nil {
		return self.BalanceProof.TransferredAmount
	}

	return amount
}

func (self *NettingChannelEndState) GetAddress() common.Address {
	return self.Address
}

func NewNettingChannelEndState() *NettingChannelEndState {
	result := new(NettingChannelEndState)

	result.SecretHashesToLockedlocks = make(SecretHashToLock)
	result.SecretHashesToUnlockedlocks = make(SecretHashToPartialUnlockProof)
	result.SecretHashesToOnchainUnlockedlocks = make(SecretHashToPartialUnlockProof)

	//Construct empty merkle tree with No leaves and empty root!
	result.Merkletree = new(MerkleTreeState)
	result.Merkletree.init()

	return result
}

type NettingChannelState struct {
	Identifier               common.ChannelID
	ChainId                  common.ChainID
	TokenAddress             common.Address
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetworkIdentifier   common.TokenNetworkID
	RevealTimeout            common.BlockHeight
	SettleTimeout            common.BlockHeight
	OurState                 *NettingChannelEndState
	PartnerState             *NettingChannelEndState
	DepositTransactionQueue  TransactionOrderHeap
	OpenTransaction          *TransactionExecutionStatus
	CloseTransaction         *TransactionExecutionStatus
	SettleTransaction        *TransactionExecutionStatus
	UpdateTransaction        *TransactionExecutionStatus
	OurUnlockTransaction     *TransactionExecutionStatus
}

func (self *NettingChannelState) GetIdentifier() common.ChannelID {
	return self.Identifier
}

func (self *NettingChannelState) GetChannelEndState(side int) *NettingChannelEndState {
	if side == 0 {
		return self.OurState
	} else {
		return self.PartnerState
	}
}

func (self *NettingChannelState) GetContractBalance(ownAddress common.Address, targetAddress common.Address, partnerAddress common.Address) (common.TokenAmount, error) {
	var result common.TokenAmount

	ourState := self.GetChannelEndState(0)
	partnerState := self.GetChannelEndState(1)

	if addressEqual(targetAddress, ownAddress) {
		result = ourState.GetContractBalance()
	} else if addressEqual(partnerAddress, ownAddress) {
		result = partnerState.GetContractBalance()
	} else {
		return 0, errors.New("target address must be one of the channel participants!")
	}
	return result, nil
}

func (self *NettingChannelState) GetGasBalance(ownAddress common.Address, targetAddress common.Address, partnerAddress common.Address) (common.TokenAmount, error) {
	var result common.TokenAmount

	ourState := self.GetChannelEndState(0)
	partnerState := self.GetChannelEndState(1)

	if addressEqual(targetAddress, ownAddress) {
		result = ourState.GetGasBalance()
	} else if addressEqual(partnerAddress, ownAddress) {
		result = partnerState.GetGasBalance()
	} else {
		return 0, errors.New("target address must be one of the channel participants!")
	}
	return result, nil
}

type TransactionChannelNewBalance struct {
	ParticipantAddress common.Address
	ContractBalance    common.TokenAmount
	DepositBlockHeight common.BlockHeight
}

type TransactionOrder struct {
	BlockHeight common.BlockHeight
	Transaction TransactionChannelNewBalance
}

func addressEqual(address1 common.Address, address2 common.Address) bool {
	result := true

	for i := 0; i < 20; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}
