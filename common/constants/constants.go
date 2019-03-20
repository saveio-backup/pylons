package constants

const (
	COUNT_OF_CONFIRM_BLOCK      int     = 0
	SNAPSHOT_STATE_CHANGE_COUNT int     = 5000
	ADDR_LEN                    int     = 20
	EDGEID_LEN                  int     = 40
	HASH_LEN                    int     = 32
	SECRET_LEN                  int     = 32
	POLL_FOR_COMFIRMED          int     = 15
	OPEN_CHANNEL_RETRY_TIMEOUT  float32 = 3
	OPEN_CHANNEL_RETRY_TIMES    int     = 5
	DEPOSIT_RETRY_TIMEOUT       float32 = 3
	DEPOSIT_RETRY_TIMES         int     = 5
	SETTLE_TIMEOUT              int     = 10000000
	ALARM_INTERVAL              int     = 500
	DEFAULT_REVEAL_TIMEOUT      int     = 1000
	MAX_MSG_QUEUE               uint32  = 5000
)
