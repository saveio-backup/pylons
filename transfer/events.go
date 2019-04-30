package transfer

import (
	"github.com/saveio/pylons/common"
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
	Initiator                common.Address
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
	Participant            common.Address
	TokenNetworkIdentifier common.TokenNetworkID
	WithdrawAmount         common.TokenAmount
}

type SendWithdraw struct {
	SendMessageEvent
	Participant            common.Address
	TokenNetworkIdentifier common.TokenNetworkID
	WithdrawAmount         common.TokenAmount
	ParticipantSignature   common.Signature
	ParticipantAddress     common.Address
	ParticipantPublicKey   common.PubKey
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

type EventWithdrawRequestSentFailed struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	WithdrawAmount         common.TokenAmount
	Reason                 string
}

type EventInvalidReceivedWithdrawRequest struct {
	ChannelIdentifier common.ChannelID
	Participant       common.Address
	TotalWithdraw     common.TokenAmount
	Reason            string
}

type EventInvalidReceivedWithdraw struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant            common.Address
	TotalWithdraw          common.TokenAmount
	Reason                 string
}

type SendCooperativeSettleRequest struct {
	SendMessageEvent
	TokenNetworkIdentifier common.TokenNetworkID
	Participant1           common.Address
	Participant1Balance    common.TokenAmount
	Participant2           common.Address
	Participant2Balance    common.TokenAmount
}

type SendCooperativeSettle struct {
	SendMessageEvent
	TokenNetworkIdentifier common.TokenNetworkID
	Participant1           common.Address
	Participant1Balance    common.TokenAmount
	Participant2           common.Address
	Participant2Balance    common.TokenAmount
	Participant1Signature  common.Signature
	Participant1Address    common.Address
	Participant1PublicKey  common.PubKey
}

type ContractSendChannelCooperativeSettle struct {
	ContractSendEvent
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Participant1           common.Address
	Participant1Balance    common.TokenAmount
	Participant2           common.Address
	Participant2Balance    common.TokenAmount
	Participant1Signature  common.Signature
	Participant1Address    common.Address
	Participant1PublicKey  common.PubKey
	Participant2Signature  common.Signature
	Participant2Address    common.Address
	Participant2PublicKey  common.PubKey
}

type EventCooperativeSettleRequestSentFailed struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Reason                 string
}

type EventInvalidReceivedCooperativeSettleRequest struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Reason                 string
}

type EventInvalidReceivedCooperativeSettle struct {
	TokenNetworkIdentifier common.TokenNetworkID
	ChannelIdentifier      common.ChannelID
	Reason                 string
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
