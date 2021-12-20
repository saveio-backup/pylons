package transfer

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"reflect"
	"sync"
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

func Test_getAmountLocked(t *testing.T) {
	state := NewNettingChannelEndState()
	sh1 := common.GetHash(common.SecretRandom(constants.SecretLen))
	sh2 := common.GetHash(common.SecretRandom(constants.SecretLen))
	//state.SecretHashesToLockedLocks = map[common.SecretHash]*HashTimeLockState{
	//	sh1: {Amount: common.TokenAmount(1)},
	//	sh2: {Amount: common.TokenAmount(2)},
	//}
	var m sync.Map
	m.Store(sh1, &HashTimeLockState{Amount: common.TokenAmount(1)})
	m.Store(sh2, &HashTimeLockState{Amount: common.TokenAmount(2)})
	state.SecretHashesToLockedLocks = m

	type args struct {
		endState *NettingChannelEndState
	}
	tests := []struct {
		name string
		args args
		want common.Balance
	}{
		{
			args: args{endState: NewNettingChannelEndState()},
			want: common.Balance(0),
		},
		{
			args: args{endState: state},
			want: common.Balance(3),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAmountLocked(tt.args.endState); got != tt.want {
				t.Errorf("getAmountLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAmountLocked_with_write(t *testing.T) {
	state := NewNettingChannelEndState()
	sh1 := common.GetHash(common.SecretRandom(constants.SecretLen))
	sh2 := common.GetHash(common.SecretRandom(constants.SecretLen))
	sh3 := common.GetHash(common.SecretRandom(constants.SecretLen))
	//state.SecretHashesToLockedLocks = map[common.SecretHash]*HashTimeLockState{
	//	sh1: {Amount: common.TokenAmount(1)},
	//	sh2: {Amount: common.TokenAmount(2)},
	//	sh3: {Amount: common.TokenAmount(3)},
	//}
	var m sync.Map
	m.Store(sh1, &HashTimeLockState{Amount: common.TokenAmount(1)})
	m.Store(sh2, &HashTimeLockState{Amount: common.TokenAmount(2)})
	m.Store(sh3, &HashTimeLockState{Amount: common.TokenAmount(3)})
	state.SecretHashesToLockedLocks = m

	type args struct {
		endState *NettingChannelEndState
	}
	tests := []struct {
		name string
		args args
		want common.Balance
	}{
		{
			args: args{endState: NewNettingChannelEndState()},
			want: common.Balance(0),
		},
		{
			args: args{endState: state},
			want: common.Balance(6),
		},
	}

	for i := 0; i < 1000; i++ {
		go func() {
			sh := common.GetHash(common.SecretRandom(constants.SecretLen))
			//state.SecretHashesToLockedLocks[sh] = &HashTimeLockState{
			//	Amount: common.TokenAmount(1),
			//}
			state.SecretHashesToLockedLocks.Store(sh, &HashTimeLockState{
				Amount: common.TokenAmount(1),
			})
		}()
	}

	var wg sync.WaitGroup
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					got := getAmountLocked(tt.args.endState)
					if got == 0 {
						t.Errorf("getAmountLocked() = %v, want %v", got, tt.want)
					}
					wg.Done()
				}()
			}
		})
	}
	wg.Wait()
}

func Test_getAmountLocked_with_write2(t *testing.T) {
	state := NewNettingChannelEndState()
	sh1 := common.GetHash(common.SecretRandom(constants.SecretLen))
	sh2 := common.GetHash(common.SecretRandom(constants.SecretLen))
	sh3 := common.GetHash(common.SecretRandom(constants.SecretLen))
	state.SecretHashesToOnChainUnLockedLocks = map[common.SecretHash]*UnlockPartialProofState{
		sh1: {Lock: &HashTimeLockState{Amount: 1}},
		sh2: {Lock: &HashTimeLockState{Amount: 2}},
		sh3: {Lock: &HashTimeLockState{Amount: 3}},
	}

	type args struct {
		endState *NettingChannelEndState
	}
	tests := []struct {
		name string
		args args
		want common.Balance
	}{
		{
			name: "zero",
			args: args{endState: NewNettingChannelEndState()},
			want: common.Balance(0),
		},
		{
			name: "go",
			args: args{endState: state},
			want: common.Balance(6),
		},
	}

	for i := 0; i < 1000; i++ {
		go func() {
			sh := common.GetHash(common.SecretRandom(constants.SecretLen))
			state.SecretHashesToOnChainUnLockedLocks[sh] = &UnlockPartialProofState{
				Lock: &HashTimeLockState{Amount: 1},
			}
		}()
	}

	var wg sync.WaitGroup
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 1000; i++ {
				wg.Add(1)
				go func() {
					got := getAmountLocked(tt.args.endState)
					if got == 0 {
						t.Errorf("getAmountLocked() = %v, want %v", got, tt.want)
					}
					wg.Done()
				}()
			}
		})
	}
	wg.Wait()
}
