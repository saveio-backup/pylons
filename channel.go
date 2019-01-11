package channel

import (
	"errors"
	"fmt"
	"net"

	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common/log"
	ch "github.com/oniio/oniChannel/channelservice"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

var Version = "0.1"

type Channel struct {
	Config  *ChannelConfig
	Service *ch.ChannelService
}

type ChannelConfig struct {
	ClientType   string
	ChainNodeURL string

	// Transport config
	ListenAddress  string // ip + port
	Protocol       string // tcp or kcp
	MappingAddress string // ip + port, used to register in Endpoint contract when use address mapping

	DBPath string
}

func DefaultChannelConfig() *ChannelConfig {
	config := &ChannelConfig{
		ClientType:    "rpc",
		ChainNodeURL:  "http://localhost:20336",
		ListenAddress: "127.0.0.1:3001",
		Protocol:      "tcp",
		DBPath:        ".",
	}
	return config
}

func NewChannel(config *ChannelConfig, account *account.Account) (*Channel, error) {
	blockChainService := network.NewBlockchainService(config.ClientType, config.ChainNodeURL, account)
	if blockChainService == nil {
		log.Fatal("createing blockchain service failed")
		return nil, errors.New("createing blockchain service failed")
	}

	transport := setupTransport(blockChainService, config)

	startBlock, err := blockChainService.Client.GetCurrentBlockHeight()
	if err != nil {
		log.Fatal("can not get current block height from blockchain service")
		return nil, fmt.Errorf("GetCurrentBlockHeight error:%s", err)
	}
	ipPort := config.ListenAddress
	if config.MappingAddress != "" {
		ipPort = config.MappingAddress
	}

	h, p, err := net.SplitHostPort(ipPort)
	if err != nil {
		log.Fatal("invalid listenning url ", ipPort)
		return nil, err
	}

	// construct the option map
	option := map[string]string{
		"database_path": config.DBPath,
		"host":          h,
		"port":          p,
	}

	service := ch.NewChannelService(
		blockChainService,
		typing.BlockHeight(startBlock),
		transport,
		new(ch.ChannelEventHandler),
		new(ch.MessageHandler),
		option)
	log.Info("channel service created, use account ", blockChainService.GetAccount().Address.ToBase58())
	channel := &Channel{
		Config:  config,
		Service: service,
	}
	return channel, nil
}

func setupTransport(blockChainService *network.BlockchainService, config *ChannelConfig) *transport.Transport {

	trans := transport.NewTransport(config.Protocol, discovery)
	trans.SetAddress(config.ListenAddress)
	trans.SetMappingAddress(config.MappingAddress)
	trans.SetKeys(blockChainService.GetAccount().PubKey())

	return trans
}

func (this *Channel) StartService() {
	this.Service.Start()
}

func (this *Channel) Stop() {
	this.Service.Stop()
}

func (this *Channel) RegisterReceiveNotification(notificaitonChannel chan *transfer.EventPaymentReceivedSuccess) {
	this.Service.ReceiveNotificationChannels[notificaitonChannel] = struct{}{}
}

func (this *Channel) GetVersion() string {
	return Version
}
