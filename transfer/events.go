package transfer

import (
	"github.com/saveio/pylons/common"
)

type ContractSendChannelClose struct {
	ContractSendEvent
	ChannelId      common.ChannelID
	TokenAddress   common.TokenAddress
	TokenNetworkId common.TokenNetworkID
	BalanceProof   *BalanceProofSignedState
}

type ContractSendChannelSettle struct {
	ContractSendEvent
	ChannelId      common.ChannelID
	TokenNetworkId common.TokenNetworkAddress
}

type ContractSendChannelUpdateTransfer struct {
	ContractSendExpireAbleEvent
	ChannelId      common.ChannelID
	TokenNetworkId common.TokenNetworkID
	BalanceProof   *BalanceProofSignedState
}

type ContractSendChannelBatchUnlock struct {
	ContractSendEvent
	TokenAddress   common.TokenAddress
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant    common.Address
}

type ContractSendSecretReveal struct {
	ContractSendExpireAbleEvent
	Secret common.Secret
}

type EventPaymentSentSuccess struct {
	PaymentNetworkId common.PaymentNetworkID
	TokenNetworkId   common.TokenNetworkID
	Identifier       common.PaymentID
	Amount           common.TokenAmount
	Target           common.Address
}

type EventPaymentSentFailed struct {
	PaymentNetworkId common.PaymentNetworkID
	TokenNetworkId   common.TokenNetworkID
	Identifier       common.PaymentID
	Target           common.Address
	Reason           string
}

type EventPaymentReceivedSuccess struct {
	PaymentNetworkId common.PaymentNetworkID
	TokenNetworkId   common.TokenNetworkID
	Identifier       common.PaymentID
	Amount           common.TokenAmount
	Initiator        common.Address
}

type EventTransferReceivedInvalidDirectTransfer struct {
	Identifier common.PaymentID
	Reason     string
}

type SendDirectTransfer struct {
	SendMessageEvent
	PaymentId    common.PaymentID
	BalanceProof *BalanceProofUnsignedState
	TokenAddress common.TokenAddress
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
	Participant    common.Address
	TokenNetworkId common.TokenNetworkID
	WithdrawAmount common.TokenAmount
}

type SendWithdraw struct {
	SendMessageEvent
	Participant          common.Address
	TokenNetworkId       common.TokenNetworkID
	WithdrawAmount       common.TokenAmount
	ParticipantSignature common.Signature
	ParticipantAddress   common.Address
	ParticipantPublicKey common.PubKey
}

type ContractSendChannelWithdraw struct {
	ContractSendEvent
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant    common.Address
	TotalWithdraw  common.TokenAmount

	ParticipantSignature common.Signature
	ParticipantAddress   common.Address
	ParticipantPublicKey common.PubKey

	PartnerSignature common.Signature
	PartnerAddress   common.Address
	PartnerPublicKey common.PubKey
}

type EventWithdrawRequestSentFailed struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	WithdrawAmount common.TokenAmount
	Reason         string
}

type EventInvalidReceivedWithdrawRequest struct {
	ChannelId     common.ChannelID
	Participant   common.Address
	TotalWithdraw common.TokenAmount
	Reason        string
}

type EventInvalidReceivedWithdraw struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Participant    common.Address
	TotalWithdraw  common.TokenAmount
	Reason         string
}

type EventWithdrawRequestTimeout struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Reason         string
}
type SendCooperativeSettleRequest struct {
	SendMessageEvent
	TokenNetworkId      common.TokenNetworkID
	Participant1        common.Address
	Participant1Balance common.TokenAmount
	Participant2        common.Address
	Participant2Balance common.TokenAmount
}

type SendCooperativeSettle struct {
	SendMessageEvent
	TokenNetworkId        common.TokenNetworkID
	Participant1          common.Address
	Participant1Balance   common.TokenAmount
	Participant2          common.Address
	Participant2Balance   common.TokenAmount
	Participant1Signature common.Signature
	Participant1Address   common.Address
	Participant1PublicKey common.PubKey
}

type ContractSendChannelCooperativeSettle struct {
	ContractSendEvent
	TokenNetworkId        common.TokenNetworkID
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

type EventCooperativeSettleRequestSentFailed struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Reason         string
}

type EventInvalidReceivedCooperativeSettleRequest struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Reason         string
}

type EventInvalidReceivedCooperativeSettle struct {
	TokenNetworkId common.TokenNetworkID
	ChannelId      common.ChannelID
	Reason         string
}

type EventInvalidReceivedTransferRefund struct {
	PaymentId common.PaymentID
	Reason    string
}

type EventInvalidReceivedLockExpired struct {
	SecretHash common.SecretHash
	Reason     string
}

type EventInvalidReceivedLockedTransfer struct {
	PaymentId common.PaymentID
	Reason    string
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
	PaymentId    common.PaymentID
	TokenAddress common.TokenAddress
	Secret       common.Secret
	SecretHash   common.SecretHash
	BalanceProof *BalanceProofUnsignedState
}

type SendSecretRequest struct {
	SendMessageEvent
	PaymentId  common.PaymentID
	Amount     common.TokenAmount
	Expiration common.BlockExpiration
	SecretHash common.SecretHash
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
			Recipient: sendLockedTransferEvent.Recipient,
			ChannelId: sendLockedTransferEvent.ChannelId,
			MessageId: sendLockedTransferEvent.MessageId,
		},
		Transfer: sendLockedTransferEvent.Transfer,
	}
	return sendRefundTransfer
}

//-------------------------------------------------------------------------------------------------
