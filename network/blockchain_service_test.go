package network

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/themis-go-sdk/wallet"
	"github.com/saveio/themis/account"
	"testing"
	"time"
)

var (
	WalletPath = "/Users/smallyu/work/gogs/scan-deploy/node1/wallet.dat"
	WalletPwd  = []byte("pwd")
	service *BlockChainService
)

func TestNewBlockChainService(t *testing.T) {
	w, err := wallet.OpenWallet(WalletPath)
	if err != nil {
		t.Fatal("wallet.Open error:", err)
	}
	a, err := w.GetDefaultAccount(WalletPwd)
	if err != nil {
		t.Fatal("GetDefaultAccount error:", err)
	}

	type args struct {
		clientType string
		url        []string
		account    *account.Account
	}
	tests := []struct {
		name string
		args args
		want *BlockChainService
	}{
		{
			args: args{
				clientType: "rpc",
				url:        []string{"http://localhost:20336"},
				account:    a,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := make(chan bool)
			go func() {
				service = NewBlockChainService(tt.args.clientType, tt.args.url, tt.args.account)
				c<-true
			}()

			select {
			case <-c:
				if service == nil {
					t.Fatal("blockchain service is nil")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("connect blockchain timeout")
			}
		})
	}
}

func TestBlockChainService_BlockHeight(t *testing.T) {
	TestNewBlockChainService(t)
	tests := []struct {
		name    string
		want    common.BlockHeight
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := service
			got, err := this.BlockHeight()
			if (err != nil) != tt.wantErr {
				t.Errorf("BlockHeight() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}

func TestBlockChainService_GetBlock(t *testing.T) {
	TestNewBlockChainService(t)
	type args struct {
		param interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := service
			got, err := this.GetBlock(tt.args.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}