package service

import (
	"testing"
)

func TestParseEvent(t *testing.T) {
	type args struct {
		event map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			args: args{event: map[string]interface{}{
				"walletAddr": "",
			}},
		},
		{
			args: args{event: map[string]interface{}{
				"walletAddr": []interface{}{},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEvent(tt.args.event)
			t.Log(len(got))
			for k, v := range got {
				t.Log(k, v)
			}
		})
	}
}
