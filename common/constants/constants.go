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

	DEFAULT_MEDIATION_FEE_MARGIN 		= 0.003
	DEFAULT_MEDIATION_FEE_FLAT   		= 1
	DEFAULT_MEDIATION_FEE_PROPORTIONAL  = 1000000
)
