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
	Secrethash            common.SecretHash
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
	secrethash        common.SecretHash
	BalanceProof      BalanceProofSignedState
}

type ReceiveDelivered struct {
	AuthenticatedSenderStateChange
	MessageIdentifier common.MessageID
}

type ReceiveProcessed struct {
	AuthenticatedSenderStateChange
	MessageIdentifier common.MessageID
}
