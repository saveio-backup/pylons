package utils

const InitialPort int = 38647

const DefaultTransportRetriesBeforeBackoff int = 5
const DefaultTransportThrottleCapacity int = 10
const DefaultTransportThrottleFillRate int = 10
const DefaultTransportUdpRetryInterval int = 1

const DefaultRevealTimeout int = 50
const DefaultSettleTimeout int = 500
const DefaultRetryTimeout float32 = 0.5
const DefaultJoinableFundsTarget float32 = 0.4
const DefaultInitialChannelTarget int = 3
const DefaultWaitForSettle bool = true

// set to 0 to speed up debugging
const DefaultNumberOfConfirmationsBlock int = 0
const DefaultChannelSyncTimeout int = 5

const DefaultNatKeepaliveRetries int = 5
const DefaultNatKeepaliveTimeout int = 5
const DefaultNatInvitationTimeout int = 15

const DefaultShutdownTimeout int = 2

const OracleBlocknumberDriftTolerange int = 3
