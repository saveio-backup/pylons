package transfer

import (
	"bytes"
	"crypto/rand"
	"errors"
	chainComm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"math"
	"math/big"
	"sort"
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
	NodeAddressesToNetworkStates map[common.Address]string
	Address                      common.Address
	PaymentMapping               PaymentMappingState
	PendingTransactions          []Event
	QueueIdsToQueues             QueueIdsToQueuesType
}

func NewChainState() *ChainState {
	result := new(ChainState)

	result.IdentifiersToPaymentNetworks = make(map[common.PaymentNetworkID]*PaymentNetworkState)
	result.NodeAddressesToNetworkStates = make(map[common.Address]string)
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
	//	serializedData, _ := jsonext.Marshal(chainState)
	//	res, _ := jsonext.UnmarshalExt(serializedData, nil, CreateObjectByClassId)
	//	result, _ = res.(*ChainState)
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
		value.NodeAddressesToNetworkStates = make(map[common.Address]string)
		for addr, state := range chainState.NodeAddressesToNetworkStates {
			value.NodeAddressesToNetworkStates[addr] = state
		}
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
	NetworkGraphName string
	Nodes            []string
	Edges            []Edge
}

type TokenNetworkState struct {
	Address                      common.TokenNetworkID
	TokenAddress                 common.TokenAddress
	NetworkGraph                 *TokenNetworkGraph
	ChannelIdentifiersToChannels map[common.ChannelID]*NettingChannelState
	PartnerAddressesToChannels   map[common.Address]map[common.ChannelID]*NettingChannelState
}

func NewTokenNetworkState(localAddr common.Address) *TokenNetworkState {
	result := &TokenNetworkState{}
	result.ChannelIdentifiersToChannels = make(map[common.ChannelID]*NettingChannelState)
	result.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
	result.NetworkGraph = &TokenNetworkGraph{}

	localNode := chainComm.Address(localAddr)
	result.NetworkGraph.Nodes = append(result.NetworkGraph.Nodes, localNode.ToBase58())
	return result
}

func (self *TokenNetworkState) AddRoute(addr1 common.Address, addr2 common.Address, channelId common.ChannelID) {
	node1 := chainComm.Address(addr1)
	node2 := chainComm.Address(addr2)
	log.Infof("addRoute addr: %v addr: %v", node1.ToBase58(), node2.ToBase58())
	self.NetworkGraph.Nodes = append(self.NetworkGraph.Nodes, node1.ToBase58())
	self.NetworkGraph.Nodes = append(self.NetworkGraph.Nodes, node2.ToBase58())

	edge1 := Edge{NodeA: node1.ToBase58(), NodeB: node2.ToBase58(), Distance: 1}
	edge2 := Edge{NodeA: node2.ToBase58(), NodeB: node1.ToBase58(), Distance: 1}

	self.NetworkGraph.Edges = append(self.NetworkGraph.Edges, edge1)
	self.NetworkGraph.Edges = append(self.NetworkGraph.Edges, edge2)
}

func (self *TokenNetworkState) DelRoute(channelId common.ChannelID) {
	//networkGraphState := self.NetworkGraph
	//if _, ok := networkGraphState.ChannelIdentifierToParticipants[channelId]; ok {
	//	participants := networkGraphState.ChannelIdentifierToParticipants[channelId]
	//
	//	p1 := networkGraphState.NodeAddressesToNodeId[participants[1]]
	//	networkGraphState.NodeIdToNodeAddresses[p1] = participants[1]
	//	delete(networkGraphState.NodeIdToNodeAddresses, p1)
	//	delete(networkGraphState.NodeAddressesToNodeId, participants[1])
	//
	//
	//	p2 := networkGraphState.NodeAddressesToNodeId[participants[2]]
	//	networkGraphState.NodeIdToNodeAddresses[p2] = participants[2]
	//	delete(networkGraphState.NodeIdToNodeAddresses, p2)
	//	delete(networkGraphState.NodeAddressesToNodeId, participants[2])
	//
	//	networkGraphState.Graph().(graph.EdgeRemover).RemoveEdge(p1, p2)
	//	networkGraphState.Graph().(graph.EdgeRemover).RemoveEdge(p2, p1)
	//	delete(networkGraphState.ChannelIdentifierToParticipants, channelId)
	//
	//	oAddr1 := chainComm.Address(participants[1])
	//	oAddr2 := chainComm.Address(participants[2])
	//	fmt.Println("[DelRoute]:", oAddr1.ToBase58(), oAddr2.ToBase58())
	//}
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
