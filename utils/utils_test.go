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
			want: "11.3963587",
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

func TestRemoveRightZero(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{s: "0.0000000001"},
			want: "0.0000000001",
		},
		{
			args: args{s: "0.0000000010"},
			want: "0.000000001",
		},
		{
			args: args{s: "0.0000000100"},
			want: "0.00000001",
		},
		{
			args: args{s: "0.1000000000"},
			want: "0.1",
		},
		{
			args: args{s: "1.0"},
			want: "1",
		},
		{
			args: args{s: "10.0"},
			want: "10",
		},
		{
			args: args{s: "1"},
			want: "1",
		},
		{
			args: args{s: "10"},
			want: "10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveRightZero(tt.args.s); got != tt.want {
				t.Errorf("RemoveRightZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
