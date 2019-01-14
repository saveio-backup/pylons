package transfer

import (
	"github.com/oniio/oniChannel/typing"
)

type Block struct {
	BlockHeight typing.BlockHeight
	GasLimit    typing.BlockGasLimit
	BlockHash   typing.BlockHash
}

type ActionCancelPayment struct {
	PaymentIdentifier typing.PaymentID
}

type ActionChannelClose struct {
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
}

type ActionCancelTransfer struct {
	TransferIdentifier typing.TransferID
}

type ActionTransferDirect struct {
	TokenNetworkIdentifier typing.TokenNetworkID
	ReceiverAddress        typing.Address
	PaymentIdentifier      typing.PaymentID
	Amount                 typing.TokenAmount
}

type ContractReceiveChannelNew struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelState           *NettingChannelState
	ChannelIdentifier      typing.ChannelID
}

type ContractReceiveChannelClosed struct {
	ContractReceiveStateChange
	TransactionFrom        typing.Address
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
}

type ActionInitChain struct {
	BlockHeight typing.BlockHeight
	OurAddress  typing.Address
	ChainId     typing.ChainID
}

type ActionNewTokenNetwork struct {
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetwork             *TokenNetworkState
}

type ContractReceiveChannelNewBalance struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	DepositTransaction     TransactionChannelNewBalance
}

type ContractReceiveChannelSettled struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
}

type ActionLeaveAllNetworks struct {
}

type ActionChangeNodeNetworkState struct {
	NodeAddress  typing.Address
	NetworkState string
}

type ContractReceiveNewPaymentNetwork struct {
	ContractReceiveStateChange
	PaymentNetwork *PaymentNetworkState
}

type ContractReceiveNewTokenNetwork struct {
	ContractReceiveStateChange
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetwork             *TokenNetworkState
}

type ContractReceiveSecretReveal struct {
	ContractReceiveStateChange
	SecretRegistryAddress typing.SecretRegistryAddress
	Secrethash            typing.SecretHash
	Secret                typing.Secret
}

type ContractReceiveChannelBatchUnlock struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	Participant            typing.Address
	Partner                typing.Address
	Locksroot              typing.Locksroot
	UnlockedAmount         typing.TokenAmount
	ReturnedTokens         typing.TokenAmount
}

type ContractReceiveRouteNew struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	Participant1           typing.Address
	Participant2           typing.Address
}

type ContractReceiveRouteClosed struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
}

type ContractReceiveUpdateTransfer struct {
	ContractReceiveStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	Nonce                  typing.Nonce
}

type ReceiveTransferDirect struct {
	AuthenticatedSenderStateChange
	TokenNetworkIdentifier typing.TokenNetworkID
	MessageIdentifier      typing.MessageID
	PaymentIdentifier      typing.PaymentID
	BalanceProof           *BalanceProofSignedState
}

type ReceiveUnlock struct {
	AuthenticatedSenderStateChange
	MessageIdentifier typing.MessageID
	Secret            typing.Secret
	secrethash        typing.SecretHash
	BalanceProof      BalanceProofSignedState
}

type ReceiveDelivered struct {
	AuthenticatedSenderStateChange
	MessageIdentifier typing.MessageID
}

type ReceiveProcessed struct {
	AuthenticatedSenderStateChange
	MessageIdentifier typing.MessageID
}
