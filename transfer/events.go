package transfer

import (
	"github.com/oniio/oniChannel/common"
)

type ContractSendChannelClose struct {
	ContractSendEvent
	ChannelIdentifier      common.ChannelID
	TokenAddress           common.TokenAddress
	TokenNetworkIdentifier common.TokenNetworkID
	BalanceProof           *BalanceProofSignedState
}

type ContractSendChannelSettle struct {
	ContractSendEvent
	ChannelIdentifier      common.ChannelID
	TokenNetworkIdentifier common.TokenNetworkAddress
}

type ContractSendChannelUpdateTransfer struct {
	ContractSendExpirableEvent
	ChannelIdentifier      common.ChannelID
	TokenNetworkIdentifier common.TokenNetworkID
	BalanceProof           *BalanceProofSignedState
}

type ContractSendChannelBatchUnlock struct {
	ContractSendEvent
	TokenAddress           common.TokenAddress
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
}

type ContractSendSecretReveal struct {
	ContractSendExpirableEvent
	Secret common.Secret
}

type EventPaymentSentSuccess struct {
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetworkIdentifier   common.TokenNetworkID
	Identifier               common.PaymentID
	Amount                   common.TokenAmount
	Target                   common.Address
}

type EventPaymentSentFailed struct {
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetworkIdentifier   common.TokenNetworkID
	Identifier               common.PaymentID
	Target                   common.Address
	Reason                   string
}

type EventPaymentReceivedSuccess struct {
	PaymentNetworkIdentifier common.PaymentNetworkID
	TokenNetworkIdentifier   common.TokenNetworkID
	Identifier               common.PaymentID
	Amount                   common.TokenAmount
	Initiator                common.InitiatorAddress
}

type EventTransferReceivedInvalidDirectTransfer struct {
	Identifier common.PaymentID
	Reason     string
}

type SendDirectTransfer struct {
	SendMessageEvent
	PaymentIdentifier common.PaymentID
	BalanceProof      *BalanceProofUnsignedState
	TokenAddress      common.TokenAddress
}

type SendProcessed struct {
	SendMessageEvent
}
