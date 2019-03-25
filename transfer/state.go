package transfer

import (
	"bytes"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
)

type SecretHashToLock map[common.SecretHash]*HashTimeLockState
type SecretHashToPartialUnlockProof map[common.SecretHash]*UnlockPartialProofState
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
	for {
		b := new(big.Int).SetInt64(math.MaxInt64)
		if id, err := rand.Int(rand.Reader, b); err == nil {
			messageId := id.Int64()
			return common.MessageID(messageId)
		}
	}
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
	IdentifiersToPaymentNetworks map[common.PaymentNetworkID]*PaymentNetworkState
	NodeAddressesToNetworkStates *sync.Map
	Address                      common.Address
	PaymentMapping               PaymentMappingState
	PendingTransactions          []Event
	QueueIdsToQueues             QueueIdsToQueuesType
}

func NewChainState() *ChainState {
	result := new(ChainState)

	result.IdentifiersToPaymentNetworks = make(map[common.PaymentNetworkID]*PaymentNetworkState)
	result.NodeAddressesToNetworkStates = new(sync.Map)
	result.PaymentMapping.SecretHashesToTask = make(map[common.SecretHash]State)
	result.PendingTransactions = []Event{}
	result.QueueIdsToQueues = make(QueueIdsToQueuesType)

	return result
}

//Adjust PaymentNetworkState in IdentifiersToPaymentNetworks
func (self *ChainState) AdjustChainState() {
	for _, v := range self.IdentifiersToPaymentNetworks {
		v.AdjustPaymentNetworkState()
	}

	return
}

func DeepCopy(src State) *ChainState {
	//var result *ChainState
	//
	//if src == nil {
	//	return nil
	//} else if chainState, ok := src.(*ChainState); ok {
	//	serializedData, err := jsonext.Marshal(chainState)
	//	if err != nil {
	//		log.Error("[DeepCopy] jsonext.Marshal Error: ", err.Error())
	//	}
	//	if serializedData == nil {
	//		log.Error("[DeepCopy] jsonext.Marshal Error: serializedData is nil.")
	//	}
	//
	//	result = NewChainState()
	//	_, err = jsonext.UnmarshalExt(serializedData, &result, CreateObjectByClassId)
	//	if err != nil {
	//		log.Error("[DeepCopy] jsonext.UnmarshalExt Error: ", err.Error())
	//	}
	//	result.AdjustChainState()
	//}
	//
	//return result

	if src == nil {
		return nil
	} else if chainState, ok := src.(*ChainState); ok {
		var value ChainState
		value.ChainId = chainState.ChainId
		value.Address = chainState.Address
		value.BlockHeight = chainState.BlockHeight

		value.IdentifiersToPaymentNetworks = make(map[common.PaymentNetworkID]*PaymentNetworkState)
		for id, state := range chainState.IdentifiersToPaymentNetworks {
			value.IdentifiersToPaymentNetworks[id] = &PaymentNetworkState{}
			value.IdentifiersToPaymentNetworks[id].Address = state.Address
			value.IdentifiersToPaymentNetworks[id].TokenAddressesToTokenIdentifiers = make(map[common.TokenAddress]common.TokenNetworkID)
			for tokenAddress, tokenNetworkId := range state.TokenAddressesToTokenIdentifiers {
				value.IdentifiersToPaymentNetworks[id].TokenAddressesToTokenIdentifiers[tokenAddress] = tokenNetworkId
			}
			value.IdentifiersToPaymentNetworks[id].TokenIdentifiersToTokenNetworks = make(map[common.TokenNetworkID]*TokenNetworkState)
			for tokenNetworkId, tokenNetworkState := range state.TokenIdentifiersToTokenNetworks {
				var tokenNwState TokenNetworkState
				tokenNwState.Address = tokenNetworkState.Address
				tokenNwState.TokenAddress = tokenNetworkState.TokenAddress
				tokenNwState.NetworkGraph = tokenNetworkState.NetworkGraph
				tokenNwState.ChannelIdentifiersToChannels = make(map[common.ChannelID]*NettingChannelState)
				for channelId, channelState := range tokenNetworkState.ChannelIdentifiersToChannels {
					tokenNwState.ChannelIdentifiersToChannels[channelId] = channelState
				}
				tokenNwState.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
				for addr, mp := range tokenNetworkState.PartnerAddressesToChannels {
					tokenNwState.PartnerAddressesToChannels[addr] = mp
				}
				value.IdentifiersToPaymentNetworks[id].TokenIdentifiersToTokenNetworks[tokenNetworkId] = &tokenNwState
			}
		}
		value.NodeAddressesToNetworkStates = new(sync.Map)
		chainState.NodeAddressesToNetworkStates.Range(func(addr, state interface{}) bool {
			value.NodeAddressesToNetworkStates.Store(addr, state)
			return true
		})

		value.PaymentMapping.SecretHashesToTask = make(map[common.SecretHash]State)
		for hash, itf := range chainState.PaymentMapping.SecretHashesToTask {
			value.PaymentMapping.SecretHashesToTask[hash] = itf
		}
		for i := 0; i < len(chainState.PendingTransactions); i++ {
			value.PendingTransactions = append(value.PendingTransactions, chainState.PendingTransactions[i])
		}

		value.QueueIdsToQueues = make(map[QueueIdentifier][]Event)
		for k, v := range chainState.QueueIdsToQueues {
			value.QueueIdsToQueues[k] = v
		}
		return &value
	}
	return nil
}

type PaymentNetworkState struct {
	Address                          common.PaymentNetworkID
	TokenIdentifiersToTokenNetworks  map[common.TokenNetworkID]*TokenNetworkState
	TokenAddressesToTokenIdentifiers map[common.TokenAddress]common.TokenNetworkID
}

func (self *PaymentNetworkState) GetAddress() common.Address {
	return common.Address(self.Address)
}

func NewPaymentNetworkState() *PaymentNetworkState {
	result := new(PaymentNetworkState)

	result.TokenIdentifiersToTokenNetworks = make(map[common.TokenNetworkID]*TokenNetworkState)
	result.TokenAddressesToTokenIdentifiers = make(map[common.TokenAddress]common.TokenNetworkID)

	return result
}

//Adjust TokenNetworkState in TokenIdentifiersToTokenNetworks
//rebuild tokenAddressesToTokenNetworks
func (self *PaymentNetworkState) AdjustPaymentNetworkState() {
	self.TokenAddressesToTokenIdentifiers = make(map[common.TokenAddress]common.TokenNetworkID)

	for _, v := range self.TokenIdentifiersToTokenNetworks {
		v.AdjustTokenNetworkState()
		self.TokenAddressesToTokenIdentifiers[v.TokenAddress] = v.Address
	}
	return
}

type TokenNetworkGraph struct {
	Nodes map[common.Address]int64
	Edges map[common.EdgeId]int64
}

type TokenNetworkState struct {
	Address                      common.TokenNetworkID
	TokenAddress                 common.TokenAddress
	NetworkGraph                 TokenNetworkGraph
	ChannelIdentifiersToChannels map[common.ChannelID]*NettingChannelState
	PartnerAddressesToChannels   map[common.Address]map[common.ChannelID]*NettingChannelState
}

func NewTokenNetworkState(localAddr common.Address) *TokenNetworkState {
	result := &TokenNetworkState{}
	result.ChannelIdentifiersToChannels = make(map[common.ChannelID]*NettingChannelState)
	result.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)

	result.NetworkGraph = TokenNetworkGraph{}
	result.NetworkGraph.Nodes = make(map[common.Address]int64)
	result.NetworkGraph.Edges = make(map[common.EdgeId]int64)

	result.NetworkGraph.Nodes[localAddr] = 1
	return result
}

func (self *TokenNetworkState) AddRoute(addr1 common.Address, addr2 common.Address, channelId common.ChannelID) {
	log.Infof("AddRoute addr: %v addr: %v", common.ToBase58(addr1), common.ToBase58(addr2))

	networkGraphState := self.NetworkGraph

	var node1Node2, node2Node1 common.EdgeId
	copy(node1Node2[:constants.ADDR_LEN], addr1[:])
	copy(node1Node2[constants.ADDR_LEN:], addr2[:])

	copy(node2Node1[:constants.ADDR_LEN], addr2[:])
	copy(node2Node1[constants.ADDR_LEN:], addr1[:])

	if _, ok := networkGraphState.Edges[node1Node2]; !ok {
		networkGraphState.Edges[node1Node2] = 1
		if _, ok := networkGraphState.Nodes[addr1]; ok {
			networkGraphState.Nodes[addr1] = networkGraphState.Nodes[addr1] + 1
		} else {
			networkGraphState.Nodes[addr1] = 1
		}
	} else {
		return
	}

	if _, ok := networkGraphState.Edges[node2Node1]; !ok {
		networkGraphState.Edges[node2Node1] = 1
		if _, ok := networkGraphState.Nodes[addr2]; ok {
			networkGraphState.Nodes[addr2] = networkGraphState.Nodes[addr2] + 1
		} else {
			networkGraphState.Nodes[addr2] = 1
		}
	} else {
		return
	}
}

func (self *TokenNetworkState) DelRoute(channelId common.ChannelID) {
	networkGraphState := self.NetworkGraph
	if _, ok := self.ChannelIdentifiersToChannels[channelId]; ok {
		addr1 := self.ChannelIdentifiersToChannels[channelId].OurState.Address
		addr2 := self.ChannelIdentifiersToChannels[channelId].PartnerState.Address

		log.Infof("DelRoute addr: %v addr: %v", common.ToBase58(addr1), common.ToBase58(addr2))

		var node1Node2, node2Node1 common.EdgeId
		copy(node1Node2[:constants.ADDR_LEN], addr1[:])
		copy(node1Node2[constants.ADDR_LEN:], addr2[:])

		copy(node2Node1[:constants.ADDR_LEN], addr2[:])
		copy(node2Node1[constants.ADDR_LEN:], addr1[:])

		if _, ok := networkGraphState.Edges[node1Node2]; ok {
			delete(networkGraphState.Edges, node1Node2)
			if _, ok := networkGraphState.Nodes[addr1]; ok {
				if networkGraphState.Nodes[addr1] == 1 {
					delete(networkGraphState.Nodes, addr1)
				}
				networkGraphState.Nodes[addr1] = networkGraphState.Nodes[addr1] - 1

			}
		}

		if _, ok := networkGraphState.Edges[node2Node1]; ok {
			delete(networkGraphState.Edges, node2Node1)
			if _, ok := networkGraphState.Nodes[addr2]; ok {
				if networkGraphState.Nodes[addr2] == 1 {
					delete(networkGraphState.Nodes, addr2)
				}
				networkGraphState.Nodes[addr2] = networkGraphState.Nodes[addr2] - 1
			}
		}
	}
}

func (self *TokenNetworkState) GetTokenAddress() common.TokenAddress {
	return self.TokenAddress
}

//rebuild PartnerAddressesToChannels
func (self *TokenNetworkState) AdjustTokenNetworkState() {
	self.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
	for _, v := range self.ChannelIdentifiersToChannels {
		if self.PartnerAddressesToChannels[v.PartnerState.Address] == nil {
			self.PartnerAddressesToChannels[v.PartnerState.Address] = make(map[common.ChannelID]*NettingChannelState)
		}
		self.PartnerAddressesToChannels[v.PartnerState.Address][v.Identifier] = v
	}

	return
}

type PaymentMappingState struct {
	SecretHashesToTask map[common.SecretHash]State
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
	SecretHash common.Keccak256
	Encoded    []byte
	LockHash   common.LockHash
}

func (self *HashTimeLockState) CalcLockHash() common.LockHash {
	buf := new(bytes.Buffer)
	log.Debug("[CalcLockHash]: ", self.Amount, self.Expiration, self.SecretHash)
	buf.Write(Uint64ToBytes(uint64(self.Amount)))
	buf.Write(Uint64ToBytes(uint64(self.Expiration)))
	buf.Write(self.SecretHash[:])

	encoded := buf.Bytes()
	hash := common.LockHash(common.GetHash(encoded))
	return hash
}

type UnlockPartialProofState struct {
	Lock   *HashTimeLockState
	Secret common.Secret
}

type UnlockProofState struct {
	MerkleProof []common.Keccak256
	LockEncoded []byte
	Secret      common.Secret
}

type TransactionExecutionStatus struct {
	StartedBlockHeight  common.BlockHeight
	FinishedBlockHeight common.BlockHeight
	Result              string
}

type NettingChannelEndState struct {
	Address                            common.Address
	ContractBalance                    common.TokenAmount
	SecretHashesToLockedLocks          SecretHashToLock
	SecretHashesToUnLockedLocks        SecretHashToPartialUnlockProof
	SecretHashesToOnChainUnLockedLocks SecretHashToPartialUnlockProof
	MerkleTree                         *MerkleTreeState
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
	//if self.BalanceProof != nil {
	//	return self.BalanceProof.TransferredAmount
	//}

	if self.BalanceProof != nil {
		return self.ContractBalance - self.BalanceProof.TransferredAmount
	} else {
		return self.ContractBalance
	}

	return amount
}

func (self *NettingChannelEndState) GetAddress() common.Address {
	return self.Address
}

func NewNettingChannelEndState() *NettingChannelEndState {
	result := new(NettingChannelEndState)

	result.SecretHashesToLockedLocks = make(SecretHashToLock)
	result.SecretHashesToUnLockedLocks = make(SecretHashToPartialUnlockProof)
	result.SecretHashesToOnChainUnLockedLocks = make(SecretHashToPartialUnlockProof)

	//Construct empty merkle tree with No leaves and empty root!
	result.MerkleTree = new(MerkleTreeState)
	result.MerkleTree.init()

	return result
}

type NettingChannelState struct {
	Identifier               common.ChannelID
	ChainId                  common.ChainID
	TokenAddress             common.Address
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetworkIdentifier   common.TokenNetworkID
	RevealTimeout            common.BlockTimeout
	SettleTimeout            common.BlockTimeout
	OurState                 *NettingChannelEndState
	PartnerState             *NettingChannelEndState
	DepositTransactionQueue  TransactionOrderHeap
	OpenTransaction          *TransactionExecutionStatus
	CloseTransaction         *TransactionExecutionStatus
	SettleTransaction        *TransactionExecutionStatus
	UpdateTransaction        *TransactionExecutionStatus
	OurUnlockTransaction     *TransactionExecutionStatus
	WithdrawTransaction      *TransactionExecutionStatus
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

//-----------------------------------------------------------------

type InitiatorPaymentState struct {
	Initiator         *InitiatorTransferState
	CancelledChannels []common.ChannelID
}

type InitiatorTransferState struct {
	TransferDescription   *TransferDescriptionWithSecretState
	ChannelIdentifier     common.ChannelID
	Transfer              *LockedTransferUnsignedState
	RevealSecret          *SendSecretReveal
	ReceivedSecretRequest bool
}

type WaitingTransferState struct {
	Transfer *LockedTransferSignedState
	State    string
}

type MediatorTransferState struct {
	SecretHash      common.SecretHash
	Secret          common.Secret
	TransfersPair   []*MediationPairState
	WaitingTransfer *WaitingTransferState
}

//initial value for State should be set to "secret_request"
//other valid value are "reveal_secret", "expired"
type TargetTransferState struct {
	Route    *RouteState
	Transfer *LockedTransferSignedState
	Secret   *common.Secret
	State    string
}

type LockedTransferUnsignedState struct {
	PaymentIdentifier common.PaymentID
	Token             common.Address
	BalanceProof      *BalanceProofUnsignedState
	Lock              *HashTimeLockState
	Initiator         common.Address
	Target            common.Address
}

type LockedTransferSignedState struct {
	MessageIdentifier common.MessageID
	PaymentIdentifier common.PaymentID
	Token             common.Address
	BalanceProof      *BalanceProofSignedState
	Lock              *HashTimeLockState
	Initiator         common.InitiatorAddress
	Target            common.Address
}

//NOTE, need calcuate SecretHash based on Secret in construct func
type TransferDescriptionWithSecretState struct {
	PaymentNetworkIdentifier common.PaymentNetworkID
	PaymentIdentifier        common.PaymentID
	Amount                   common.TokenAmount
	TokenNetworkIdentifier   common.TokenNetworkID
	Initiator                common.Address
	Target                   common.Address
	Secret                   common.Secret
	SecretHash               common.SecretHash
}

type MediationPairState struct {
	PayeeAddress  common.Address
	PayeeTransfer *LockedTransferUnsignedState
	PayeeState    string
	PayerTransfer *LockedTransferSignedState
	PayerState    string
}
