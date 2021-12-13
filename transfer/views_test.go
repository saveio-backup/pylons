package transfer

import (
	"github.com/saveio/pylons/common"
	"reflect"
	"testing"
)

func TestGetNeighbours(t *testing.T) {
	type args struct {
		chainState *ChainState
	}
	tests := []struct {
		name string
		args args
		want []common.Address
	}{
		{
			args: args{chainState: NewChainState()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNeighbours(tt.args.chainState); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNeighbours() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockHeight(t *testing.T) {
	type args struct {
		chainState *ChainState
	}
	tests := []struct {
		name string
		args args
		want common.BlockHeight
	}{
		{
			args: args{chainState: NewChainState()},
			want: common.BlockHeight(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBlockHeight(tt.args.chainState); got != tt.want {
				t.Errorf("GetBlockHeight() = %v, want %v", got, tt.want)
			}
		})
	}
}