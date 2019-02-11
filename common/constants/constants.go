package constants

const (
	COUNT_OF_CONFIRM_BLOCK      int     = 0
	SNAPSHOT_STATE_CHANGE_COUNT int     = 500
	ADDR_LEN                    int     = 20
	HASH_LEN                    int     = 32
	POLL_FOR_COMFIRMED          int     = 15
	OPEN_CHANNEL_RETRY_TIMEOUT  float32 = 3
	OPEN_CHANNEL_RETRY_TIMES    int     = 5
	DEPOSIT_RETRY_TIMEOUT       float32 = 3
	DEPOSIT_RETRY_TIMES         int     = 5
	SETTLE_TIMEOUT              int     = 10000000
	ALARM_INTERVAL              int     = 3000
	DEFAULT_REVEAL_TIMEOUT      int     = 1000
	MAX_MSG_QUEUE               uint32  = 5000
)
