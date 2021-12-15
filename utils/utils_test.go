package utils

import (
	"fmt"
	"github.com/saveio/themis/cmd/utils"
	"testing"
)

func TestA(t *testing.T) {
	a := 1129.6
	fmt.Println(a * 100)

	b := 11296
	fmt.Println(b * 10)
}

func TestB(t *testing.T) {
	a := 11396358700
	aa := utils.FormatUsdt(uint64(a))
	fmt.Println(aa)

	aa2 := FormatUSDT(uint64(a))
	fmt.Println(aa2)
}

func TestFormatUSDT(t *testing.T) {
	type args struct {
		amount uint64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "overflow",
			args: args{amount: 11396358700},
			want: "11.396358700",
		},
		{
			name: "not overflow",
			args: args{amount: 11396358699},
			want: "11.396358699",
		},
		{
			name: "general",
			args: args{amount: 1},
			want: "0.000000001",
		},
		{
			name: "general",
			args: args{amount: 9},
			want: "0.000000009",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatUSDT(tt.args.amount); got != tt.want {
				t.Errorf("FormatUSDT() = %v, want %v", got, tt.want)
			}
		})
	}
}