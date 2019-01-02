package transfer

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniChannel/utils/jsonext"
)

type SecretHashToLock map[typing.SecretHash]HashTimeLockState
type SecretHashToPartialUnlockProof map[typing.SecretHash]UnlockPartialProofState
type QueueIdsToQueuesType map[QueueIdentifier][]Event

const ChannelStateClosed string = "closed"
const ChannelStateClosing string = "waiting_for_close"
const ChannelStateOpened string = "opened"
const ChannelStateSettled string = "settled"
const ChannelStateSettling string = "waiting_for_settle"
const ChannelStateUnusable string = "channel_unusable"

const TransactionExecutionStatusSuccess string = "success"
const TransactionExecutionStatusFailure string = "failure"

const NodeNetworkUnknown string = "unknown"
const NodeNetworkUnreachable string = "unreachable"
const NodeNetworkReachable string = "reachable"

func MessageIdentifierFromPrng(prng *rand.Rand) typing.MessageID {
	messageId := prng.Int63n(math.MaxInt64)
	return typing.MessageID(messageId)
}

type TransactionOrderHeap []TransactionOrder

func (h TransactionOrderHeap) Len() int {
	return len(h)
}

func (h TransactionOrderHeap) Less(i, j int) bool {
	return h[i].BlockNumber < h[j].BlockNumber
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
	TokenNetworkIdentifier typing.TokenNetworkID
	ManagerState           State
}

type MediatorTask struct {
	TokenNetworkIdentifier typing.TokenNetworkID
	MediatorState          State
}

type TargetTask struct {
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	TargetState            State
}

type ChainState struct {
	BlockNumber                  typing.BlockNumber
	ChainId                      typing.ChainID
	IdentifiersToPaymentnetworks map[typing.PaymentNetworkID]*PaymentNetworkState
	NodeAddressesToNetworkstates map[typing.Address]string
	Address                      typing.Address
	PaymentMapping               PaymentMappingState
	PendingTransactions          []Event
	PseudoRandomGenerator        *rand.Rand
	QueueIdsToQueues             QueueIdsToQueuesType
}

func NewChainState() *ChainState {
	result := new(ChainState)

	result.IdentifiersToPaymentnetworks = make(map[typing.PaymentNetworkID]*PaymentNetworkState)
	result.NodeAddressesToNetworkstates = make(map[typing.Address]string)
	result.PaymentMapping.SecrethashesToTask = make(map[typing.SecretHash]State)
	result.PseudoRandomGenerator = rand.New(rand.NewSource(time.Now().Unix()))
	result.PendingTransactions = []Event{}
	result.QueueIdsToQueues = make(QueueIdsToQueuesType)

	return result
}

//Adjust PaymentNetworkState in IdentifiersToPaymentnetworks
func (self *ChainState) AdjustChainState() {
	for _, v := range self.IdentifiersToPaymentnetworks {
		v.AdjustPaymentNetworkState()
	}

	self.PseudoRandomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
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
	Address                         typing.PaymentNetworkID
	TokenIdentifiersToTokenNetworks map[typing.TokenNetworkID]*TokenNetworkState
	tokenAddressesToTokenNetworks   map[typing.TokenAddress]*TokenNetworkState
}

func (self *PaymentNetworkState) GetAddress() typing.Address {
	return typing.Address(self.Address)
}

func NewPaymentNetworkState() *PaymentNetworkState {
	result := new(PaymentNetworkState)

	result.TokenIdentifiersToTokenNetworks = make(map[typing.TokenNetworkID]*TokenNetworkState)
	result.tokenAddressesToTokenNetworks = make(map[typing.TokenAddress]*TokenNetworkState)

	return result
}

//Adjust TokenNetworkState in TokenIdentifiersToTokenNetworks
//rebuild tokenAddressesToTokenNetworks
func (self *PaymentNetworkState) AdjustPaymentNetworkState() {
	self.tokenAddressesToTokenNetworks = make(map[typing.TokenAddress]*TokenNetworkState)

	for _, v := range self.TokenIdentifiersToTokenNetworks {
		v.AdjustTokenNetworkState()
		self.tokenAddressesToTokenNetworks[v.TokenAddress] = v
	}

	return
}

type TokenNetworkState struct {
	Address                      typing.TokenNetworkID
	TokenAddress                 typing.TokenAddress
	NetworkGraph                 *TokenNetworkGraphState
	ChannelIdentifiersToChannels map[typing.ChannelID]*NettingChannelState
	partnerAddressesToChannels   map[typing.Address]map[typing.ChannelID]*NettingChannelState
}

func NewTokenNetworkState() *TokenNetworkState {
	result := new(TokenNetworkState)
	result.ChannelIdentifiersToChannels = make(map[typing.ChannelID]*NettingChannelState)
	result.partnerAddressesToChannels = make(map[typing.Address]map[typing.ChannelID]*NettingChannelState)

	return result
}

func (self *TokenNetworkState) GetTokenAddress() typing.TokenAddress {
	return self.TokenAddress
}

//rebuild partnerAddressesToChannels
func (self *TokenNetworkState) AdjustTokenNetworkState() {
	self.partnerAddressesToChannels = make(map[typing.Address]map[typing.ChannelID]*NettingChannelState)
	for _, v := range self.ChannelIdentifiersToChannels {
		if self.partnerAddressesToChannels[v.PartnerState.Address] == nil {
			self.partnerAddressesToChannels[v.PartnerState.Address] = make(map[typing.ChannelID]*NettingChannelState)
		}
		self.partnerAddressesToChannels[v.PartnerState.Address][v.Identifier] = v
	}

	return
}

type TokenNetworkGraphState struct {
	//[TODO] introduce Graph struct!
}

type PaymentMappingState struct {
	SecrethashesToTask map[typing.SecretHash]State
}

type RouteState struct {
	NodeAddress       typing.Address
	ChannelIdentifier typing.ChannelID
}

type BalanceProofUnsignedState struct {
	Nonce                  typing.Nonce
	TransferredAmount      typing.TokenAmount
	LockedAmount           typing.TokenAmount
	LocksRoot              typing.Locksroot
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	ChainId                typing.ChainID
}

type BalanceProofSignedState struct {
	Nonce                  typing.Nonce
	TransferredAmount      typing.TokenAmount
	LockedAmount           typing.TokenAmount
	LocksRoot              typing.Locksroot
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	MessageHash            typing.Keccak256
	Signature              typing.Signature
	Sender                 typing.Address
	ChainId                typing.ChainID
	PublicKey              typing.PubKey
}

type HashTimeLockState struct {
	Amount     typing.TokenAmount
	Expiration typing.BlockNumber
	Secrethash typing.Keccak256
	Encoded    []byte
	Lockhash   typing.LockHash
}

type UnlockPartialProofState struct {
	Lock   HashTimeLockState
	Secret typing.Secret
}

type UnlockProofState struct {
	MerkelProof []typing.Keccak256
	LockEncoded []byte
	Secret      typing.Secret
}

type TransactionExecutionStatus struct {
	StartedBlockNumber  typing.BlockNumber
	FinishedBlockNumber typing.BlockNumber
	Result              string
}

type MerkleTreeState struct {
	Layers [][]typing.Keccak256
}

func (self *MerkleTreeState) init() {

	self.Layers = append(self.Layers, []typing.Keccak256{})
	self.Layers = append(self.Layers, []typing.Keccak256{})

	emptyRoot := typing.Keccak256{}
	self.Layers[1] = append(self.Layers[1], emptyRoot)
}

type NettingChannelEndState struct {
	Address                            typing.Address
	ContractBalance                    typing.TokenAmount
	SecretHashesToLockedlocks          SecretHashToLock
	SecretHashesToUnlockedlocks        SecretHashToPartialUnlockProof
	SecretHashesToOnchainUnlockedlocks SecretHashToPartialUnlockProof
	Merkletree                         *MerkleTreeState
	BalanceProof                       *BalanceProofSignedState
}

func (self *NettingChannelEndState) GetContractBalance() typing.TokenAmount {
	return self.ContractBalance
}

func (self *NettingChannelEndState) GetBalance() typing.TokenAmount {
	var amount typing.TokenAmount

	// the balanceProof could be nil in following cases:
	// outState.BalanceProof is nil when no payment has been sent
	// partnetState.BalanceProof is nil when no payment has been received
	if self.BalanceProof != nil {
		return self.BalanceProof.TransferredAmount
	}

	return amount
}

func (self *NettingChannelEndState) GetAddress() typing.Address {
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
	Identifier               typing.ChannelID
	ChainId                  typing.ChainID
	TokenAddress             typing.Address
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetworkIdentifier   typing.TokenNetworkID
	RevealTimeout            typing.BlockNumber
	SettleTimeout            typing.BlockNumber
	OurState                 *NettingChannelEndState
	PartnerState             *NettingChannelEndState
	DepositTransactionQueue  TransactionOrderHeap
	OpenTransaction          *TransactionExecutionStatus
	CloseTransaction         *TransactionExecutionStatus
	SettleTransaction        *TransactionExecutionStatus
	UpdateTransaction        *TransactionExecutionStatus
	OurUnlockTransaction     *TransactionExecutionStatus
}

func (self *NettingChannelState) GetIdentifier() typing.ChannelID {
	return self.Identifier
}

func (self *NettingChannelState) GetChannelEndState(side int) *NettingChannelEndState {
	if side == 0 {
		return self.OurState
	} else {
		return self.PartnerState
	}
}

type TransactionChannelNewBalance struct {
	ParticipantAddress typing.Address
	ContractBalance    typing.TokenAmount
	DepositBlockNumber typing.BlockNumber
}

type TransactionOrder struct {
	BlockNumber typing.BlockNumber
	Transaction TransactionChannelNewBalance
}
