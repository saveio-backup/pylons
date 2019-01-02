package proxies

import (
	"fmt"
	"testing"

	"github.com/oniio/oniChannel/typing"

	"github.com/oniio/oniChannel/account"
	"github.com/oniio/oniChannel/network/rpc"
	"github.com/stretchr/testify/assert"
)

func TestRegisterEndpoint(t *testing.T) {
	client := rpc.NewRpcClient("http://127.0.0.1:20336")
	wallet, err := account.OpenWallet("../rpc/wallet.dat")
	assert.Nil(t, err)
	client.Account, err = wallet.GetDefaultAccount([]byte("pwd"))
	assert.Nil(t, err)
	proxy := rpc.NewContractProxy(client)
	discovery := &Discovery{
		JsonrpcClient: client,
		Proxy:         proxy,
	}
	var addr typing.Address
	copy(addr[:], client.Account.Address[:])
	txHash, err := discovery.RegisterEndpoint(addr, "10.0.0.1:8080")
	assert.Nil(t, err)
	fmt.Printf("registerEndpoint txHash:%s\n", txHash)
}
