package transfer

import (
	"github.com/saveio/pylons/common"
)

type Block struct {
	BlockHeight common.BlockHeight
	GasLimit    common.BlockGasLimit
	BlockHash   common.BlockHash
}

type ActionCancelPayment struct {
	PaymentId common.PaymentID
}

type ActionChannelClose struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
}

type ActionCancelTransfer struct {
	TransferId common.TransferID
}

type ActionTransferDirect struct {
	TokenNetworkId  common.TokenNetworkID
	ReceiverAddress common.Address
	PaymentId       common.PaymentID
	Amount          common.TokenAmount
}

type ActionWithdraw struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant    common.Address
	Partner        common.Address
	TotalWithdraw  common.TokenAmount
}

type ReceiveWithdrawRequest struct {
	TokenNetworkId       common.TokenNetworkID
	MessageId            common.MessageID
	ChannelId            common.ChannelID
	Participant          common.Address
	TotalWithdraw        common.TokenAmount
	ParticipantSignature common.Signature
	ParticipantAddress   common.Address
	ParticipantPublicKey common.PubKey
}

type ReceiveWithdraw struct {
	ReceiveWithdrawRequest
	PartnerSignature common.Signature
	PartnerAddress   common.Address
	PartnerPublicKey common.PubKey
}

type ContractReceiveChannelWithdraw struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant    common.Address
	TotalWithdraw  common.TokenAmount
}

type ActionCooperativeSettle struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
}

type ReceiveCooperativeSettleRequest struct {
	TokenNetworkId        common.TokenNetworkID
	MessageId             common.MessageID
	ChannelId             common.ChannelID
	Participant1          common.Address
	Participant1Balance   common.TokenAmount
	Participant2          common.Address
	Participant2Balance   common.TokenAmount
	Participant1Signature common.Signature
	Participant1Address   common.Address
	Participant1PublicKey common.PubKey
}

type ReceiveCooperativeSettle struct {
	TokenNetworkId        common.TokenNetworkID
	MessageId             common.MessageID
	ChannelId             common.ChannelID
	Participant1          common.Address
	Participant1Balance   common.TokenAmount
	Participant2          common.Address
	Participant2Balance   common.TokenAmount
	Participant1Signature common.Signature
	Participant1Address   common.Address
	Participant1PublicKey common.PubKey
	Participant2Signature common.Signature
	Participant2Address   common.Address
	Participant2PublicKey common.PubKey
}

type ContractReceiveChannelCooperativeSettled struct {
	ContractReceiveStateChange
	TokenNetworkId     common.TokenNetworkID
	ChannelId          common.ChannelID
	Participant1Amount common.TokenAmount
	Participant2Amount common.TokenAmount
}

type ContractReceiveChannelNew struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelState   *NettingChannelState
	ChannelId      common.ChannelID
}

type ContractReceiveChannelClosed struct {
	ContractReceiveStateChange
	TransactionFrom common.Address
	TokenNetworkId  common.TokenNetworkID
	ChannelId       common.ChannelID
}

type ActionInitChain struct {
	BlockHeight common.BlockHeight
	OurAddress  common.Address
	ChainId     common.ChainID
}

type ActionNewTokenNetwork struct {
	PaymentNetworkId common.PaymentNetworkID
	TokenNetwork     *TokenNetworkState
}

type ContractReceiveChannelNewBalance struct {
	ContractReceiveStateChange
	TokenNetworkId     common.TokenNetworkID
	ChannelId          common.ChannelID
	DepositTransaction TransactionChannelNewBalance
}

type ContractReceiveChannelSettled struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
}

type ActionLeaveAllNetworks struct {
}

type ActionChangeNodeNetworkState struct {
	NodeAddress  common.Address
	NetworkState string
}

type ContractReceiveNewPaymentNetwork struct {
	ContractReceiveStateChange
	PaymentNetwork *PaymentNetworkState
}

type ContractReceiveNewTokenNetwork struct {
	ContractReceiveStateChange
	PaymentNetworkId common.PaymentNetworkID
	TokenNetwork     *TokenNetworkState
}

type ContractReceiveSecretReveal struct {
	ContractReceiveStateChange
	SecretRegistryAddress common.SecretRegistryAddress
	SecretHash            common.SecretHash
	Secret                common.Secret
}

type ContractReceiveChannelBatchUnlock struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	Participant    common.Address
	Partner        common.Address
	LocksRoot      common.LocksRoot
	UnlockedAmount common.TokenAmount
	ReturnedTokens common.TokenAmount
}

type ContractReceiveRouteNew struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant1   common.Address
	Participant2   common.Address
}

type ContractReceiveRouteClosed struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
}

type ContractReceiveUpdateTransfer struct {
	ContractReceiveStateChange
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Nonce          common.Nonce
}

type ReceiveTransferDirect struct {
	AuthenticatedSenderStateChange
	TokenNetworkId common.TokenNetworkID
	MessageId      common.MessageID
	PaymentId      common.PaymentID
	BalanceProof   *BalanceProofSignedState
}

type ReceiveUnlock struct {
	AuthenticatedSenderStateChange
	MessageId    common.MessageID
	Secret       common.Secret
	SecretHash   common.SecretHash
	BalanceProof *BalanceProofSignedState
}

type ReceiveDelivered struct {
	AuthenticatedSenderStateChange
	MessageId common.MessageID
}

type ReceiveProcessed struct {
	AuthenticatedSenderStateChange
	MessageId common.MessageID
}

//------------------------------------------------

type BalanceProofStateChange struct {
	AuthenticatedSenderStateChange
	BalanceProof *BalanceProofSignedState
}

type ActionInitInitiator struct {
	TransferDescription *TransferDescriptionWithSecretState
	Routes              []RouteState
}

type ActionInitMediator struct {
	Routes       []RouteState
	FromRoute    *RouteState
	FromTransfer *LockedTransferSignedState
}

//NOTO 'balance_proof': self.balance_proof wirte to json!
//don't need use same logic, just change the json query string!
type ActionInitTarget struct {
	Route    *RouteState
	Transfer *LockedTransferSignedState
}

type ActionCancelRoute struct {
	RegistryAddress common.Address
	ChannelId       common.ChannelID
	Routes          []RouteState
}

type ReceiveLockExpired struct {
	BalanceProof *BalanceProofSignedState
	SecretHash   common.SecretHash
	MessageId    common.MessageID
}

type ReceiveSecretRequest struct {
	PaymentId  common.PaymentID
	Amount     common.PaymentAmount
	Expiration common.BlockExpiration
	SecretHash common.SecretHash
	Sender     common.Address
	MessageId  common.MessageID
}

//Need calculte Secret hash when construct this struct.
type ReceiveSecretReveal struct {
	Secret    common.Secret
	Sender    common.Address
	MessageId common.MessageID
}

//Need calculte Secret hash when construct this struct.
type ReceiveTransferRefundCancelRoute struct {
	Transfer   *LockedTransferSignedState
	Routes     []RouteState
	SecretHash common.SecretHash
	Secret     common.Secret
}

type ReceiveTransferRefund struct {
	BalanceProofStateChange
	Transfer *LockedTransferSignedState
	Routes   []RouteState
}
