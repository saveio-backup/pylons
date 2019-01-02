package proxies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/oniio/oniChannel/network/rpc"
	"github.com/oniio/oniChannel/typing"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
)

type Discovery struct {
	// TODO: use blockchainservice to repliace client, proxy
	JsonrpcClient    *rpc.RpcClient
	DiscoveryAddress typing.Address
	NodeAddress      typing.Address
	//[TODO] uncomment below line after ContractProxy class is ready
	Proxy *rpc.ContractProxy
}

/* endpoint: ip + port; for example: 127.0.0.1:20336 */
func (self *Discovery) RegisterEndpoint(nodeAddress typing.Address, endpoint string) ([]byte, error) {
	if bytes.Compare(nodeAddress[:], self.JsonrpcClient.Account.Address[:]) != 0 {
		return nil, errors.New("node address don't match this account address")
	}
	h, p, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Errorf("parse endpoint err:%s\n", err)
		return nil, err
	}
	ip := net.ParseIP(h)

	txHash, err := self.Proxy.RegisterPaymentEndPoint(ip, []byte(p), common.Address(nodeAddress))
	if err != nil {
		log.Errorf("RegisterPaymentEndPoint err:%s\n", err)
		return nil, err
	}
	confirmed, err := self.JsonrpcClient.PollForTxConfirmed(time.Duration(60)*time.Second, txHash)
	if err != nil || !confirmed {
		log.Errorf("PollForTxConfirmed err:%s\n", err)
		return nil, err
	}
	return txHash, nil
}

func (self *Discovery) EndpointByAddress(nodeAddress typing.Address) string {
	info, err := self.Proxy.GetEndpointByAddress(common.Address(nodeAddress))
	if err != nil {
		log.Errorf("GetEndpointByAddress err:%s", err)
		return ""
	}
	return fmt.Sprintf("%s:%s", net.IP(info.IP), string(info.Port))
}

func (self *Discovery) Version() string {
	return hex.EncodeToString([]byte{self.Proxy.GetVersion()})
}
