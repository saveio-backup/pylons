package proxies

import (
	"fmt"
	"net"
	"time"

	chainsdk "github.com/oniio/dsp-go-sdk/chain"
	sdkcomm "github.com/oniio/dsp-go-sdk/common"
	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/typing"
)

type Discovery struct {
	ChainClient *chainsdk.Chain
	NodeAddress typing.Address
}

/* endpoint: ip + port; for example: 127.0.0.1:20336 */
func (self *Discovery) RegisterEndpoint(nodeAddress typing.Address, endpoint string) ([]byte, error) {

	h, p, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Error("parse endpoint err:", err)
		return nil, err
	}
	ip := net.ParseIP(h)

	txHash, err := self.ChainClient.Native.Channel.RegisterPaymentEndPoint(ip, []byte(p), common.Address(nodeAddress))
	if err != nil {
		log.Error("RegisterPaymentEndPoint err:", err)
		return nil, err
	}
	confirmed, err := self.ChainClient.PollForTxConfirmed(time.Duration(60)*time.Second, txHash)
	if err != nil {
		log.Warn("PollForTxConfirmed err:%", err)
		return nil, err
	}
	if !confirmed {
		log.Warn("PollForTxConfirmed err:%", err)
		return nil, err
	}
	return txHash, nil
}

func (self *Discovery) EndpointByAddress(nodeAddress typing.Address) string {
	info, err := self.ChainClient.Native.Channel.GetEndpointByAddress(common.Address(nodeAddress))
	if err != nil {
		log.Errorf("GetEndpointByAddress err:%s", err)
		return ""
	}
	return fmt.Sprintf("%s:%s", net.IP(info.IP), string(info.Port))
}

func (self *Discovery) Version() string {
	return sdkcomm.DSP_SDK_VERSION
}
