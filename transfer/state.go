package transfer

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network/dns"
	thecom "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
)

type SecretHashToLock map[common.SecretHash]*HashTimeLockState
type SecretHashToPartialUnlockProof map[common.SecretHash]*UnlockPartialProofState
type QueueIdsToQueuesType map[QueueId][]Event

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
	TokenNetworkId common.TokenNetworkID
	ManagerState   State
}

type MediatorTask struct {
	TokenNetworkId common.TokenNetworkID
	MediatorState  State
}

type TargetTask struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	TargetState    State
}

type ChainState struct {
	BlockHeight                  common.BlockHeight
	ChainId                      common.ChainID
	IdentifiersToPaymentNetworks map[common.PaymentNetworkID]*PaymentNetworkState
	Address                      common.Address
	PaymentMapping               PaymentMappingState
	PendingTransactions          []Event
	QueueIdsToQueues             QueueIdsToQueuesType
}

func NewChainState() *ChainState {
	result := new(ChainState)

	result.IdentifiersToPaymentNetworks = make(map[common.PaymentNetworkID]*PaymentNetworkState)
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
				tokenNwState.DnsAddrsMap = tokenNetworkState.DnsAddrsMap
				tokenNwState.Address = tokenNetworkState.Address
				tokenNwState.TokenAddress = tokenNetworkState.TokenAddress
				tokenNwState.NetworkGraph = tokenNetworkState.NetworkGraph
				tokenNwState.ChannelsMap = make(map[common.ChannelID]*NettingChannelState)
				for channelId, channelState := range tokenNetworkState.ChannelsMap {
					tokenNwState.ChannelsMap[channelId] = channelState
				}
				tokenNwState.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
				for addr, mp := range tokenNetworkState.PartnerAddressesToChannels {
					tokenNwState.PartnerAddressesToChannels[addr] = mp
				}
				value.IdentifiersToPaymentNetworks[id].TokenIdentifiersToTokenNetworks[tokenNetworkId] = &tokenNwState
			}
		}

		value.PaymentMapping.SecretHashesToTask = make(map[common.SecretHash]State)
		for hash, itf := range chainState.PaymentMapping.SecretHashesToTask {
			value.PaymentMapping.SecretHashesToTask[hash] = itf
		}
		for i := 0; i < len(chainState.PendingTransactions); i++ {
			value.PendingTransactions = append(value.PendingTransactions, chainState.PendingTransactions[i])
		}

		value.QueueIdsToQueues = make(QueueIdsToQueuesType)
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
	Nodes *sync.Map
	Edges *sync.Map
}

type TokenNetworkState struct {
	DnsAddrsMap                map[common.Address]int64
	Address                    common.TokenNetworkID
	TokenAddress               common.TokenAddress
	NetworkGraph               TokenNetworkGraph
	ChannelsMap                map[common.ChannelID]*NettingChannelState
	PartnerAddressesToChannels map[common.Address]map[common.ChannelID]*NettingChannelState
	FeeScheduleMap		       map[common.Address]*FeeScheduleState
}

func NewTokenNetworkState(localAddr common.Address) *TokenNetworkState {
	result := &TokenNetworkState{}
	result.ChannelsMap = make(map[common.ChannelID]*NettingChannelState)
	result.PartnerAddressesToChannels = make(map[common.Address]map[common.ChannelID]*NettingChannelState)
	result.NetworkGraph = TokenNetworkGraph{}
	result.NetworkGraph.Nodes = new(sync.Map)
	result.NetworkGraph.Edges = new(sync.Map)
	result.NetworkGraph.Nodes.Store(localAddr, int64(1))
	result.DnsAddrsMap = make(map[common.Address]int64)
	result.FeeScheduleMap = make(map[common.Address]*FeeScheduleState)
	return result
}

func (self *TokenNetworkState) GetAllDnsFromChain() map[common.Address]int64 {
	fmt.Printf("[GetAllDnsFromChain] begin\n")
	dnsAddrsMap := make(map[common.Address]int64)
	dnsNodes, err := dns.Client.GetAllDnsNodes()
	if err != nil {
		log.Errorf("[GetAllDnsFromChain] GetAllDnsNodes error: %s", err.Error())
		return nil
	}

	for _, info := range dnsNodes {
		var tmpAddr common.Address
		copy(tmpAddr[:], info.WalletAddr[:])
		dnsAddrsMap[tmpAddr] = 1
	}
	return dnsAddrsMap
}

func (self *TokenNetworkState) GetAllDns() map[common.Address]int64 {
	dnsAddrsMap := make(map[common.Address]int64)
	for addr, _ := range self.DnsAddrsMap {
		dnsAddrsMap[addr] = 1
	}
	return dnsAddrsMap
}

func (self *TokenNetworkState) updateDns(addr common.Address) {
	var theAddr thecom.Address
	copy(theAddr[:], addr[:])

	// fmt.Printf("[updateDns] %s\n", theAddr.ToBase58())
	dnsInfo, err := dns.Client.GetDnsNodeByAddr(theAddr)
	if err == nil && dnsInfo != nil {
		if self.DnsAddrsMap == nil {
			self.DnsAddrsMap = make(map[common.Address]int64)
		}
		self.DnsAddrsMap[addr] = 1
		// fmt.Printf("[updateDns] AddDns %s\n", theAddr.ToBase58())
	}
}

func (self *TokenNetworkState) AddRoute(addr1 common.Address, addr2 common.Address, channelId common.ChannelID) {
	log.Infof("[AddRoute] [%v] [%v]", common.ToBase58(addr1), common.ToBase58(addr2))
	self.updateDns(addr1)
	self.updateDns(addr2)

	networkGraphState := self.NetworkGraph

	var node1Node2, node2Node1 common.EdgeId
	copy(node1Node2[:constants.AddrLen], addr1[:])
	copy(node1Node2[constants.AddrLen:], addr2[:])

	copy(node2Node1[:constants.AddrLen], addr2[:])
	copy(node2Node1[constants.AddrLen:], addr1[:])

	if _, ok := networkGraphState.Edges.Load(node1Node2); !ok {
		networkGraphState.Edges.Store(node1Node2, int64(1))
		if v, ok := networkGraphState.Nodes.Load(addr1); ok {
			networkGraphState.Nodes.Store(addr1, v.(int64)+1)
		} else {
			networkGraphState.Nodes.Store(addr1, int64(1))
		}
	} else {
		return
	}

	if _, ok := networkGraphState.Edges.Load(node2Node1); !ok {
		networkGraphState.Edges.Store(node2Node1, int64(1))
		if v, ok := networkGraphState.Nodes.Load(addr2); ok {
			networkGraphState.Nodes.Store(addr2, v.(int64)+1)
		} else {
			networkGraphState.Nodes.Store(addr2, int64(1))
		}
	} else {
		return
	}
}

func (self *TokenNetworkState) DelRoute(channelId common.ChannelID) {
	networkGraphState := self.NetworkGraph
	if _, ok := self.ChannelsMap[channelId]; ok {
		addr1 := self.ChannelsMap[channelId].OurState.Address
		addr2 := self.ChannelsMap[channelId].PartnerState.Address

		log.Infof("[DelRoute] [%v] [%v]", common.ToBase58(addr1), common.ToBase58(addr2))

		var node1Node2, node2Node1 common.EdgeId
		copy(node1Node2[:constants.AddrLen], addr1[:])
		copy(node1Node2[constants.AddrLen:], addr2[:])

		copy(node2Node1[:constants.AddrLen], addr2[:])
		copy(node2Node1[constants.AddrLen:], addr1[:])

		if _, ok := networkGraphState.Edges.Load(node1Node2); ok {
			networkGraphState.Edges.Delete(node1Node2)
			if v, ok := networkGraphState.Nodes.Load(addr1); ok {
				if v.(int64) <= 1 {
					networkGraphState.Nodes.Delete(addr1)
				} else {
					networkGraphState.Nodes.Store(addr1, v.(int64)-1)
				}
			}
		}

		if _, ok := networkGraphState.Edges.Load(node2Node1); ok {
			networkGraphState.Edges.Delete(node2Node1)
			if v, ok := networkGraphState.Nodes.Load(addr2); ok {
				if v.(int64) <= 1 {
					networkGraphState.Nodes.Delete(addr2)
				} else {
					networkGraphState.Nodes.Store(addr2, v.(int64)-1)
				}
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
	for _, v := range self.ChannelsMap {
		if self.PartnerAddressesToChannels[v.PartnerState.Address] == nil {
			self.PartnerAddressesToChannels[v.PartnerState.Address] = make(map[common.ChannelID]*NettingChannelState)
		}
		self.PartnerAddressesToChannels[v.PartnerState.Address][v.Identifier] = v
	}
	return
}

func (t *TokenNetworkState) AddFeeSchedule(addr common.Address, fee *FeeScheduleState) {
	if t.FeeScheduleMap == nil {
		t.FeeScheduleMap = make(map[common.Address]*FeeScheduleState)
	}
	t.FeeScheduleMap[addr] = fee
}

// ----------------------------------

type PaymentMappingState struct {
	SecretHashesToTask map[common.SecretHash]State
}

type RouteState struct {
	NodeAddress common.Address
	ChannelId   common.ChannelID
}

type BalanceProofUnsignedState struct {
	Nonce             common.Nonce
	TransferredAmount common.TokenAmount
	LockedAmount      common.TokenAmount
	LocksRoot         common.LocksRoot
	BalanceHash       common.BalanceHash
	TokenNetworkId    common.TokenNetworkID
	ChannelId         common.ChannelID
	ChainId           common.ChainID
}

type BalanceProofSignedState struct {
	Nonce             common.Nonce
	TransferredAmount common.TokenAmount
	LockedAmount      common.TokenAmount
	LocksRoot         common.LocksRoot
	BalanceHash       common.BalanceHash
	TokenNetworkId    common.TokenNetworkID
	ChannelId         common.ChannelID
	MessageHash       common.Keccak256
	Signature         common.Signature
	Sender            common.Address
	ChainId           common.ChainID
	PublicKey         common.PubKey
}

type HashTimeLockState struct {
	Amount     common.TokenAmount
	Expiration common.BlockHeight
	SecretHash common.Keccak256
	Encoded    []byte
	LockHash   common.LockHash
}

func (self *HashTimeLockState) CalcLockHash() common.LockHash {
	log.Debug("[CalcLockHash]: ", self.Expiration, self.Amount, common.Keccak256Hex(self.SecretHash))
	hash := common.LockHash(common.GetHash(self.PackData()))
	return hash
}

func (self *HashTimeLockState) PackData() []byte {
	buf := new(bytes.Buffer)

	buf.Write(Uint64ToBytes(uint64(self.Expiration)))
	buf.Write(Uint64ToBytes(uint64(self.Amount)))
	buf.Write(self.SecretHash[:])

	return buf.Bytes()
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
	TotalWithdraw                      common.TokenAmount
	SecretHashesToLockedLocks          SecretHashToLock
	SecretHashesToUnLockedLocks        SecretHashToPartialUnlockProof
	SecretHashesToOnChainUnLockedLocks SecretHashToPartialUnlockProof
	MerkleTree                         *MerkleTreeState
	BalanceProof                       *BalanceProofSignedState
}

func (self *NettingChannelEndState) GetContractBalance() common.TokenAmount {
	return self.ContractBalance
}

func (self *NettingChannelEndState) GetTotalWithdraw() common.TokenAmount {
	return self.TotalWithdraw
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
		return self.ContractBalance - self.TotalWithdraw - self.BalanceProof.TransferredAmount
	} else {
		return self.ContractBalance - self.TotalWithdraw
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
	Identifier              common.ChannelID
	ChainId                 common.ChainID
	TokenAddress            common.Address
	PaymentNetworkId        common.PaymentNetworkID
	TokenNetworkId          common.TokenNetworkID
	RevealTimeout           common.BlockTimeout
	SettleTimeout           common.BlockTimeout
	FeeSchedule 			*FeeScheduleState
	OurState                *NettingChannelEndState
	PartnerState            *NettingChannelEndState
	DepositTransactionQueue TransactionOrderHeap
	OpenTransaction         *TransactionExecutionStatus
	CloseTransaction        *TransactionExecutionStatus
	SettleTransaction       *TransactionExecutionStatus
	UpdateTransaction       *TransactionExecutionStatus
	OurUnlockTransaction    *TransactionExecutionStatus
	WithdrawTransaction     *TransactionExecutionStatus
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

func (t *NettingChannelState) GetFeeSchedule() *FeeScheduleState {
	if t.FeeSchedule == nil {
		t.FeeSchedule = &FeeScheduleState{}
	}
	return t.FeeSchedule
}

func (t *NettingChannelState) SetFeeSchedule(flat common.FeeAmount, pro common.ProportionalFeeAmount) {
	if t.FeeSchedule == nil {
		t.FeeSchedule = &FeeScheduleState{}
	}
	t.FeeSchedule.Flat = flat
	t.FeeSchedule.Proportional = pro
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
	ChannelId             common.ChannelID
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
	PaymentId    common.PaymentID
	Token        common.Address
	BalanceProof *BalanceProofUnsignedState
	Lock         *HashTimeLockState
	Initiator    common.Address
	Target       common.Address
	EncSecret    common.EncSecret
	Mediators    []common.Address
}

type LockedTransferSignedState struct {
	MessageId    common.MessageID
	PaymentId    common.PaymentID
	Token        common.Address
	BalanceProof *BalanceProofSignedState
	Lock         *HashTimeLockState
	Initiator    common.Address
	Target       common.Address
	EncSecret    common.EncSecret
	Mediators    []common.Address
}

//NOTE, need calculate SecretHash based on Secret in construct func
type TransferDescriptionWithSecretState struct {
	PaymentNetworkId common.PaymentNetworkID
	PaymentId        common.PaymentID
	Amount           common.TokenAmount
	TokenNetworkId   common.TokenNetworkID
	Initiator        common.Address
	Target           common.Address
	Secret           common.Secret
	EncSecret        common.EncSecret
	SecretHash       common.SecretHash
}

type MediationPairState struct {
	PayeeAddress  common.Address
	PayeeTransfer *LockedTransferUnsignedState
	PayeeState    string
	PayerTransfer *LockedTransferSignedState
	PayerState    string
}

// FeeScheduleState is struct of mediation fee
type FeeScheduleState struct {
	CapFees bool
	Flat common.FeeAmount
	Proportional common.ProportionalFeeAmount
	ImbalancePenalty []map[common.TokenAmount]common.FeeAmount
}
