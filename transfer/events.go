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
	ContractSendExpireAbleEvent
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
	ContractSendExpireAbleEvent
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

type EventInvalidReceivedUnlock struct {
	SecretHash common.SecretHash
	Reason     string
}

type SendProcessed struct {
	SendMessageEvent
}

type SendWithdrawRequest struct {
	SendMessageEvent
	WithdrawAmount common.TokenAmount
}

type SendWithdraw struct {
	SendMessageEvent
	WithdrawAmount common.TokenAmount
}

type ContractSendChannelWithdraw struct {
	ContractSendEvent
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
	TotalWithdraw          common.TokenAmount

	ParticipantSignature common.Signature
	ParticipantAddress   common.Address
	ParticipantPublicKey common.PubKey

	PartnerSignature common.Signature
	PartnerAddress   common.Address
	PartnerPublicKey common.PubKey
}

type EventInvalidReceivedTransferRefund struct {
	PaymentIdentifier common.PaymentID
	Reason            string
}

type EventInvalidReceivedLockExpired struct {
	SecretHash common.SecretHash
	Reason     string
}

type EventInvalidReceivedLockedTransfer struct {
	PaymentIdentifier common.PaymentID
	Reason            string
}

//-------------------------------------------------------------------------------------------------

type SendLockExpired struct {
	SendMessageEvent
	BalanceProof *BalanceProofUnsignedState
	SecretHash   common.SecretHash
}

type SendLockedTransfer struct {
	SendMessageEvent
	Transfer *LockedTransferUnsignedState
}

//NOTE, need caculate SecretHash based on Secret when construct SendSecretReveal!
type SendSecretReveal struct {
	SendMessageEvent
	Secret     common.Secret
	SecretHash common.SecretHash
}

//NOTE, need caculate SecretHash based on Secret when construct SendSecretReveal!
type SendBalanceProof struct {
	SendMessageEvent
	PaymentIdentifier common.PaymentID
	TokenAddress      common.TokenAddress
	Secret            common.Secret
	SecretHash        common.SecretHash
	BalanceProof      *BalanceProofUnsignedState
}

type SendSecretRequest struct {
	SendMessageEvent
	PaymentIdentifier common.PaymentID
	Amount            common.TokenAmount
	Expiration        common.BlockExpiration
	SecretHash        common.SecretHash
}

//NOTE, 'balance_proof': self.transfer.balance_proof is skipped.
//sqlite may query the transfer.balance_proof field of json object! need work around it!
type SendRefundTransfer struct {
	SendMessageEvent
	Transfer *LockedTransferUnsignedState
}

type EventUnlockSuccess struct {
	Identifier common.PaymentID
	SecretHash common.SecretHash
}

type EventUnlockFailed struct {
	Identifier common.PaymentID
	SecretHash common.SecretHash
	Reason     string
}

type EventUnlockClaimSuccess struct {
	Identifier common.PaymentID
	SecretHash common.SecretHash
}

type EventUnlockClaimFailed struct {
	Identifier common.PaymentID
	SecretHash common.SecretHash
	Reason     string
}

type EventUnexpectedSecretReveal struct {
	SecretHash common.SecretHash
	Reason     string
}

//-------------------------------------------------------------------------------------------------

func RefundFromSendmediated(sendLockedTransferEvent *SendLockedTransfer) *SendRefundTransfer {
	sendRefundTransfer := &SendRefundTransfer{
		SendMessageEvent: SendMessageEvent{
			Recipient:         sendLockedTransferEvent.Recipient,
			ChannelIdentifier: sendLockedTransferEvent.ChannelIdentifier,
			MessageIdentifier: sendLockedTransferEvent.MessageIdentifier,
		},
		Transfer: sendLockedTransferEvent.Transfer,
	}
	return sendRefundTransfer
}

//-------------------------------------------------------------------------------------------------
