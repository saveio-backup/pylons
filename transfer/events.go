package transfer

import (
	"github.com/oniio/oniChannel/typing"
)

type ContractSendChannelClose struct {
	ContractSendEvent
	ChannelIdentifier      typing.ChannelID
	TokenAddress           typing.TokenAddress
	TokenNetworkIdentifier typing.TokenNetworkID
	BalanceProof           *BalanceProofSignedState
}

type ContractSendChannelSettle struct {
	ContractSendEvent
	ChannelIdentifier      typing.ChannelID
	TokenNetworkIdentifier typing.TokenNetworkAddress
}

type ContractSendChannelUpdateTransfer struct {
	ContractSendExpirableEvent
	ChannelIdentifier      typing.ChannelID
	TokenNetworkIdentifier typing.TokenNetworkID
	BalanceProof           *BalanceProofSignedState
}

type ContractSendChannelBatchUnlock struct {
	ContractSendEvent
	TokenAddress           typing.TokenAddress
	TokenNetworkIdentifier typing.TokenNetworkID
	ChannelIdentifier      typing.ChannelID
	Participant            typing.Address
}

type ContractSendSecretReveal struct {
	ContractSendExpirableEvent
	Secret typing.Secret
}

type EventPaymentSentSuccess struct {
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetworkIdentifier   typing.TokenNetworkID
	Identifier               typing.PaymentID
	Amount                   typing.TokenAmount
	Target                   typing.TargetAddress
}

type EventPaymentSentFailed struct {
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetworkIdentifier   typing.TokenNetworkID
	Identifier               typing.PaymentID
	Target                   typing.TargetAddress
	Reason                   string
}

type EventPaymentReceivedSuccess struct {
	PaymentNetworkIdentifier typing.PaymentNetworkID
	TokenNetworkIdentifier   typing.TokenNetworkID
	Identifier               typing.PaymentID
	Amount                   typing.TokenAmount
	Initiator                typing.InitiatorAddress
}

type EventTransferReceivedInvalidDirectTransfer struct {
	Identifier typing.PaymentID
	Reason     string
}

type SendDirectTransfer struct {
	SendMessageEvent
	PaymentIdentifier typing.PaymentID
	BalanceProof      *BalanceProofUnsignedState
	TokenAddress      typing.TokenAddress
}

type SendProcessed struct {
	SendMessageEvent
}
