package constants

const (
	AddrLen                           = 20
	EdgeIdLen                         = 40
	HashLen                           = 32
	SecretLen                         = 32
	MaxMsgQueue                       = 10000

	AlarmInterval                     = 1000
	SnapshotStateChangeCount          = 5000
	PollForComfirmed                  = 15

	OpenChannelRetryTimeOut           = 300 //ms
	OpenChannelRetryTimes             = 5
	DepositRetryTimeout               = 300 //ms
	DepositRetryTimes                 = 5

	DefaultSettleTimeout              = 500
	DefaultRevealTimeout              = 50
	DefaultWithdrawTimeout            = 5

	DefaultNumberOfMaxBlockDelay      = 3
	DefaultNumberOfConfirmationsBlock = 0
)
