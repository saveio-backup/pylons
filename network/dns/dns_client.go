package dns

import (
	"sync"

	"github.com/saveio/themis-go-sdk/client"
	"github.com/saveio/themis-go-sdk/dns"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
)

var once sync.Once

var Client *dns.Dns

func InitDnsClient(rpcServiceUrls []string, acc *account.Account) {
	once.Do(func() {
		log.Infof("[InitDnsClient] rpcServiceUrls: %v", rpcServiceUrls)
		Client = &dns.Dns{}
		Client.Client = &client.ClientMgr{}
		Client.Client.NewRpcClient().SetAddress(rpcServiceUrls)
		Client.DefAcc = acc
	})
}
