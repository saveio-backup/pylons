package transfer

import (
	"github.com/saveio/pylons/common"
	"reflect"
	"testing"
)

func Test_handleReceiveDirectTransfer(t *testing.T) {
	state := &NettingChannelState{
		Identifier:              0,
		ChainId:                 0,
		TokenAddress:            common.Address{},
		PaymentNetworkId:        common.PaymentNetworkID{},
		TokenNetworkId:          common.TokenNetworkID{},
		RevealTimeout:           0,
		SettleTimeout:           0,
		FeeSchedule:             nil,
		OurState:                nil,
		PartnerState:            nil,
		DepositTransactionQueue: nil,
		OpenTransaction:         nil,
		CloseTransaction:        nil,
		SettleTransaction:       nil,
		UpdateTransaction:       nil,
		OurUnlockTransaction:    nil,
		WithdrawTransaction:     nil,
	}
	direct := &ReceiveTransferDirect{
		AuthenticatedSenderStateChange: AuthenticatedSenderStateChange{},
		TokenNetworkId:                 common.TokenNetworkID{},
		MessageId:                      0,
		PaymentId:                      0,
		BalanceProof:                   nil,
	}
	type args struct {
		channelState   *NettingChannelState
		directTransfer *ReceiveTransferDirect
	}
	tests := []struct {
		name string
		args args
		want TransitionResult
	}{
		{
			args: args{
				channelState:   state,
				directTransfer: direct,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleReceiveDirectTransfer(tt.args.channelState, tt.args.directTransfer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleReceiveDirectTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}
