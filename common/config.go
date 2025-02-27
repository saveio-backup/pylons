package common

import (
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/common/log"
)

type PylonsConfig struct {
	MaxMsgQueue              int
	AlarmInterval            int
	SnapshotStateChangeCount int
	PollForConfirmed         int
	OpenChannelRetryTimeOut  int //ms
	OpenChannelRetryTimes    int
	DepositRetryTimeout      int //ms
	DepositRetryTimes        int
	WithdrawTimeout          int
	MaxBlockDelay            int
	ConfirmBlockCount        int
	MediationFeeConfig       MediationFeeConfig
	RouteStrategy			 string
	RoutePenaltyConfig	     RoutePenaltyConfig
}

var DefaultConfig = &PylonsConfig{
	MaxMsgQueue:              constants.DefaultMaxMsgQueue,
	AlarmInterval:            constants.DefaultAlarmInterval,
	SnapshotStateChangeCount: constants.DefaultSnapshotStateChangeCount,
	PollForConfirmed:         constants.DefaultPollForConfirmed,
	OpenChannelRetryTimeOut:  constants.DefaultOpenChannelRetryTimeOut,
	OpenChannelRetryTimes:    constants.DefaultOpenChannelRetryTimes,
	DepositRetryTimeout:      constants.DefaultDepositRetryTimeout,
	DepositRetryTimes:        constants.DefaultDepositRetryTimes,
	WithdrawTimeout:          constants.DefaultWithdrawTimeout,
	MaxBlockDelay:            constants.DefaultNumberOfMaxBlockDelay,
	ConfirmBlockCount:        constants.DefaultNumberOfConfirmationsBlock,
	MediationFeeConfig: MediationFeeConfig{
		TokenToFlatFee: map[TokenAddress]FeeAmount{
			TokenAddress(usdt.USDT_CONTRACT_ADDRESS): FeeAmount(constants.DefaultMediationFeeFlat),
		},
		TokenToProportionalFee:          map[TokenAddress]ProportionalFeeAmount{
			TokenAddress(usdt.USDT_CONTRACT_ADDRESS): ProportionalFeeAmount(constants.DefaultMediationFeeProportional),
		},
		TokenToProportionalImbalanceFee: nil,
		CapMediationFees:                false,
	},
	RouteStrategy: constants.RouteStrategyDiverse,
	RoutePenaltyConfig: RoutePenaltyConfig{
		FeePenalty:       constants.DefaultFeePenalty,
		DiversityPenalty: constants.DefaultDiversityPenalty,
	},
}

var Config *PylonsConfig

func init() {
	Config = DefaultConfig
}

func SetWithdrawTimeout(withdrawTimeout int) {
	if Config == nil {
		log.Error("[SetWithdrawTimeout] Config is nil")
		panic("[SetWithdrawTimeout] Config is nil")
	}
	Config.WithdrawTimeout = withdrawTimeout
}

func SetMaxBlockDelay(maxBlockDelay int) {
	if Config == nil {
		log.Error("[SetMaxBlockDelay] Config is nil")
		panic("[SetMaxBlockDelay] Config is nil")
	}
	Config.MaxBlockDelay = maxBlockDelay
}

type MediationFeeConfig struct {
	TokenToFlatFee                  map[TokenAddress]FeeAmount
	TokenToProportionalFee          map[TokenAddress]ProportionalFeeAmount
	TokenToProportionalImbalanceFee map[TokenAddress]ProportionalFeeAmount
	CapMediationFees                bool
}

type RoutePenaltyConfig struct {
	FeePenalty float64
	DiversityPenalty float64
}
