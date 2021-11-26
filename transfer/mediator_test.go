package transfer

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"testing"
)

func TestGetAmountWithoutFees(t *testing.T) {
	schedule := &FeeScheduleState{
		Flat:             constants.DefaultMediationFeeFlat,
		Proportional:     constants.DefaultMediationFeeProportional,
	}

	type args struct {
		amountWithFees   common.TokenAmount
		payerFeeSchedule *FeeScheduleState
	}
	tests := []struct {
		name string
		args args
		want common.PaymentWithFeeAmount
	}{
		{
			args: args{amountWithFees: 1, payerFeeSchedule: schedule},
		},
		{
			args: args{amountWithFees: 10, payerFeeSchedule: schedule},
		},
		{
			args: args{amountWithFees: 100, payerFeeSchedule: schedule},
		},
		{
			args: args{amountWithFees: 1000, payerFeeSchedule: schedule},
		},
		{
			args: args{amountWithFees: 10000, payerFeeSchedule: schedule},
		},
		{
			args: args{amountWithFees: 100000, payerFeeSchedule: schedule},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAmountWithoutFees(tt.args.amountWithFees, tt.args.payerFeeSchedule)
			t.Log(tt.args.amountWithFees, got, float64(tt.args.amountWithFees) - float64(got))
		})
	}
}
