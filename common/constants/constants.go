package constants

const (
	AddrLen              = 20
	EdgeIdLen            = 40
	HashLen              = 32
	SecretLen            = 32
	DefaultSettleTimeout = 500
	DefaultRevealTimeout = 50
)

const (
	DefaultMaxMsgQueue              = 10000
	DefaultAlarmInterval            = 1000
	DefaultSnapshotStateChangeCount = 5000
	DefaultPollForConfirmed         = 16
	DefaultOpenChannelRetryTimeOut  = 1000 //ms
	DefaultOpenChannelRetryTimes    = 16
	DefaultDepositRetryTimeout      = 1000 //ms
	DefaultDepositRetryTimes        = 16

	DefaultWithdrawTimeout            = 5
	DefaultNumberOfMaxBlockDelay      = 3
	DefaultNumberOfConfirmationsBlock = 0

	DefaultMediationFeeMargin       = 0.003
	DefaultMediationFeeFlat         = 1
	DefaultMediationFeeProportional = 1000000
	DefaultMediationFeeLimit        = 0.2

	RouteStrategyShort      = "short"
	RouteStrategyCheap      = "cheap"
	RouteStrategyDiverse    = "diverse"
	DefaultFeePenalty       = 1
	DefaultDiversityPenalty = 1
)
