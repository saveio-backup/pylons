package transfer

import (
	"github.com/oniio/oniChannel/common"
)

type Block struct {
	BlockHeight common.BlockHeight
	GasLimit    common.BlockGasLimit
	BlockHash   common.BlockHash
}

type ActionCancelPayment struct {
	PaymentIdentifier common.PaymentID
}

type ActionChannelClose struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
}

type ActionCancelTransfer struct {
	TransferIdentifier common.TransferID
}

type ActionTransferDirect struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ReceiverAddress        common.Address
	PaymentIdentifier      common.PaymentID
	Amount                 common.TokenAmount
}

type ActionWithdraw struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
	Partner                common.Address
	TotalWithdraw          common.TokenAmount
}

type ReceiveWithdrawRequest struct {
	TokenNetworkIdentifier common.TokenNetworkID
	MessageIdentifier      common.MessageID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
	TotalWithdraw          common.TokenAmount
	ParticipantSignature   common.Signature
	ParticipantAddress     common.Address
	ParticipantPublicKey   common.PubKey
}

type ReceiveWithdraw struct {
	ReceiveWithdrawRequest
	PartnerSignature common.Signature
	PartnerAddress   common.Address
	PartnerPublicKey common.PubKey
}

type ContractReceiveChannelWithdraw struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
	TotalWithdraw          common.TokenAmount
}

type ContractReceiveChannelNew struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelState           *NettingChannelState
	ChannelIdentifier      common.ChannelID
}

type ContractReceiveChannelClosed struct {
	ContractReceiveStateChange
	TransactionFrom        common.Address
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
}

type ActionInitChain struct {
	BlockHeight common.BlockHeight
	OurAddress  common.Address
	ChainId     common.ChainID
}

type ActionNewTokenNetwork struct {
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetwork             *TokenNetworkState
}

type ContractReceiveChannelNewBalance struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	DepositTransaction     TransactionChannelNewBalance
}

type ContractReceiveChannelSettled struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
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
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetwork             *TokenNetworkState
}

type ContractReceiveSecretReveal struct {
	ContractReceiveStateChange
	SecretRegistryAddress common.SecretRegistryAddress
	SecretHash            common.SecretHash
	Secret                common.Secret
}

type ContractReceiveChannelBatchUnlock struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	Participant            common.Address
	Partner                common.Address
	Locksroot              common.Locksroot
	UnlockedAmount         common.TokenAmount
	ReturnedTokens         common.TokenAmount
}

type ContractReceiveRouteNew struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant1           common.Address
	Participant2           common.Address
}

type ContractReceiveRouteClosed struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
}

type ContractReceiveUpdateTransfer struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Nonce                  common.Nonce
}

type ReceiveTransferDirect struct {
	AuthenticatedSenderStateChange
	TokenNetworkIdentifier common.TokenNetworkID
	MessageIdentifier      common.MessageID
	PaymentIdentifier      common.PaymentID
	BalanceProof           *BalanceProofSignedState
}

type ReceiveUnlock struct {
	AuthenticatedSenderStateChange
	MessageIdentifier common.MessageID
	Secret            common.Secret
	SecretHash        common.SecretHash
	BalanceProof      *BalanceProofSignedState
}

type ReceiveDelivered struct {
	AuthenticatedSenderStateChange
	MessageIdentifier common.MessageID
}

type ReceiveProcessed struct {
	AuthenticatedSenderStateChange
	MessageIdentifier common.MessageID
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
	RegistryAddress   common.Address
	ChannelIdentifier common.ChannelID
	Routes            []RouteState
}

type ReceiveLockExpired struct {
	BalanceProof      *BalanceProofSignedState
	SecretHash        common.SecretHash
	MessageIdentifier common.MessageID
}

type ReceiveSecretRequest struct {
	PaymentIdentifier common.PaymentID
	Amount            common.PaymentAmount
	Expiration        common.BlockExpiration
	SecretHash        common.SecretHash
	Sender            common.Address
	MessageIdentifier common.MessageID
}

//Need calculte Secret hash when construct this struct.
type ReceiveSecretReveal struct {
	Secret            common.Secret
	Sender            common.Address
	MessageIdentifier common.MessageID
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
