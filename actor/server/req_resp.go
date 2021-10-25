package server

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

type ChannelInfo struct {
	ChannelId     uint32
	Balance       uint64
	BalanceFormat string
	Address       string
	HostAddr      string
	TokenAddr     string
}

type ChannelsInfoResp struct {
	Balance       uint64
	BalanceFormat string
	Channels      []*ChannelInfo
}

type StartPylonsRet struct {
	Done chan bool
	Err  error
}

type StartPylonsReq struct {
	Ret *StartPylonsRet
}

type StopPylonsRet struct {
	Done chan bool
	Err  error
}

type StopPylonsReq struct {
	Ret *StopPylonsRet
}

type VersionRet struct {
	Version string
	Done    chan bool
	Err     error
}

type VersionReq struct {
	Ret *VersionRet
}

type OpenChannelRet struct {
	ChannelID common.ChannelID
	Done      chan bool
	Err       error
}

type OpenChannelReq struct {
	TokenAddress common.TokenAddress
	Target       common.Address
	Ret          *OpenChannelRet
}

type SetTotalChannelDepositRet struct {
	Done chan bool
	Err  error
}

type SetTotalChannelDepositReq struct {
	TokenAddress common.TokenAddress
	PartnerAdder common.Address
	TotalDeposit common.TokenAmount
	Ret          *SetTotalChannelDepositRet
}

type DirectTransferRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type DirectTransferReq struct {
	Target     common.Address
	Amount     common.TokenAmount
	Identifier common.PaymentID
	Ret        *DirectTransferRet
}

type MediaTransferRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type MediaTransferReq struct {
	RegisterAddress common.PaymentNetworkID
	TokenAddress    common.TokenAddress
	Media           common.Address
	Target          common.Address
	Amount          common.TokenAmount
	Identifier      common.PaymentID
	Ret             *MediaTransferRet
}

type CanTransferRet struct {
	Result bool
	Done   chan bool
	Err    error
}

type CanTransferReq struct {
	Target common.Address
	Amount common.TokenAmount
	Ret    *CanTransferRet
}

type WithdrawRet struct {
	Success bool
	Done    chan bool
	Err     error
}

type WithdrawReq struct {
	TokenAddress   common.TokenAddress
	PartnerAddress common.Address
	TotalWithdraw  common.TokenAmount
	Ret            *WithdrawRet
}

type ChannelReachableRet struct {
	Result bool
	Done   chan bool
	Err    error
}

type ChannelReachableReq struct {
	Target common.Address
	Ret    *ChannelReachableRet
}

type CloseChannelRet struct {
	Result bool
	Done   chan bool
	Err    error
}

type CloseChannelReq struct {
	Target common.Address
	Ret    *CloseChannelRet
}

type GetTotalDepositBalanceRet struct {
	Ret  uint64
	Done chan bool
	Err  error
}

type GetTotalDepositBalanceReq struct {
	Target common.Address
	Ret    *GetTotalDepositBalanceRet
}

type GetTotalWithdrawRet struct {
	Ret  uint64
	Done chan bool
	Err  error
}

type GetTotalWithdrawReq struct {
	Target common.Address
	Ret    *GetTotalWithdrawRet
}

type GetAvailableBalanceRet struct {
	Ret  uint64
	Done chan bool
	Err  error
}

type GetAvailableBalanceReq struct {
	PartnerAddress common.Address
	Ret            *GetAvailableBalanceRet
}

type GetCurrentBalanceRet struct {
	Ret  uint64
	Done chan bool
	Err  error
}

type GetCurrentBalanceReq struct {
	PartnerAddress common.Address
	Ret            *GetCurrentBalanceRet
}

type CooperativeSettleRet struct {
	Done chan bool
	Err  error
}

type CooperativeSettleReq struct {
	PartnerAddress common.Address
	Ret            *CooperativeSettleRet
}

type GetUnitPricesRet struct {
	Ret  uint64
	Done chan bool
	Err  error
}

type GetUnitPricesReq struct {
	Asset int32
	Ret   *GetUnitPricesRet
}

type SetUnitPricesRet struct {
	Ret  bool
	Done chan bool
	Err  error
}

type SetUnitPricesReq struct {
	Asset int32
	Price uint64
	Ret   *SetUnitPricesRet
}

type HealthyCheckNodeRet struct {
	Done chan bool
	Err  error
}

type HealthyCheckNodeReq struct {
	Address common.Address
	Ret     *HealthyCheckNodeRet
}

type RegisterReceiveNotificationRet struct {
	NotificationChannel chan *transfer.EventPaymentReceivedSuccess
	Done                chan bool
	Err                 error
}

type RegisterReceiveNotificationReq struct {
	Ret *RegisterReceiveNotificationRet
}

type GetAllChannelsRet struct {
	Ret  *ChannelsInfoResp
	Done chan bool
	Err  error
}

type GetAllChannelsReq struct {
	Ret *GetAllChannelsRet
}

type LastFilterBlockHeightRet struct {
	Height uint32
	Done   chan bool
	Err    error
}

type LastFilterBlockHeightReq struct {
	Ret *LastFilterBlockHeightRet
}

type PaymentResultResp struct {
	Result bool
	Reason string
}

type GetPaymentResultRet struct {
	Ret  *PaymentResultResp
	Done chan bool
	Err  error
}

type GetPaymentResultReq struct {
	Target     common.Address
	Identifier common.PaymentID
	Ret        *GetPaymentResultRet
}

type GetFeeRet struct {
	Fee  uint64
	Done chan bool
	Err  error
}

type GetFeeReq struct {
	ChannelId common.ChannelID
	Ret GetFeeRet
}

type SetFeeRet struct {
	Done chan bool
	Err  error
}

type SetFeeReq struct {
	ChannelId common.ChannelID
	WalletAddr common.Address
	Flat common.FeeAmount
	Ret SetFeeRet
}
