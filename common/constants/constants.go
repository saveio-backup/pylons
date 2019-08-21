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
	DefaultPollForConfirmed         = 15
	DefaultOpenChannelRetryTimeOut  = 300 //ms
	DefaultOpenChannelRetryTimes    = 5
	DefaultDepositRetryTimeout      = 300 //ms
	DefaultDepositRetryTimes        = 5

	DefaultWithdrawTimeout            = 5
	DefaultNumberOfMaxBlockDelay      = 3
	DefaultNumberOfConfirmationsBlock = 0
)
