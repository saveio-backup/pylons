package transfer

import "github.com/saveio/pylons/common"

type FeeScheduleState struct {
	CapFees bool
	Flat common.FeeAmount
	Proportional common.ProportionalFeeAmount
	ImbalancePenalty []map[common.TokenAmount]common.FeeAmount
}


