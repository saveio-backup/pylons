package channel

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/smartcontract/service/native/utils"
	ch "github.com/oniio/oniChannel/channelservice"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/transfer"
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

	DBPath        string
	SettleTimeout string
	RevealTimeout string
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

func NewChannelService(config *ChannelConfig, account *account.Account) (*Channel, error) {
	settleTimeout, revealTimeout, err := getTimeout(config)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	blockChainService := network.NewBlockchainService(config.ClientType, config.ChainNodeURL, account)
	if blockChainService == nil {
		log.Fatal("creating blockChain service failed")
		return nil, errors.New("creating blockChain service failed")
	}

	startBlock, err := blockChainService.ChainClient.GetCurrentBlockHeight()
	if err != nil {
		log.Fatal("can not get current block height from blockChain service")
		return nil, fmt.Errorf("GetCurrentBlockHeight error:%s", err)
	}

	// construct the option map
	option := map[string]string{
		"database_path":  config.DBPath,
		"protocol":       config.Protocol,
		"listenAddr":     config.ListenAddress,
		"mappingAddr":    config.MappingAddress,
		"settle_timeout": strconv.Itoa(settleTimeout),
		"reveal_timeout": strconv.Itoa(revealTimeout),
	}

	service := ch.NewChannelService(blockChainService, common.BlockHeight(startBlock),
		common.Address(utils.MicroPayContractAddress), new(ch.MessageHandler), option)
	log.Info("channel service created, use account ", blockChainService.GetAccount().Address.ToBase58())

	channel := &Channel{Config: config, Service: service}
	return channel, nil
}

func getTimeout(config *ChannelConfig) (settle int, reveal int, err error) {
	settleTimeout := constants.SETTLE_TIMEOUT
	if config.SettleTimeout != "" {
		settleTimeout, err = strconv.Atoi(config.SettleTimeout)
		if err != nil {
			log.Fatal("invalid settle timeout")
			return 0, 0, err
		}
	}

	revealTimeout := constants.DEFAULT_REVEAL_TIMEOUT
	if config.RevealTimeout != "" {
		revealTimeout, err = strconv.Atoi(config.RevealTimeout)
		if err != nil {
			log.Fatal("invalid reveal timeout")
			return 0, 0, err
		}
	}

	if settleTimeout < 2*revealTimeout {
		log.Fatalf("settle timeout(%d) should be at least double of revealTimeout(%d)",
			settleTimeout, revealTimeout)
		return 0, 0, err
	}

	return settleTimeout, revealTimeout, nil
}

func (this *Channel) StartService() error {
	return this.Service.Start()
}

func (this *Channel) Stop() {
	this.Service.Stop()
}

func (this *Channel) RegisterReceiveNotification(notificationChannel chan *transfer.EventPaymentReceivedSuccess) {
	this.Service.ReceiveNotificationChannels[notificationChannel] = struct{}{}
}

func (this *Channel) GetVersion() string {
	return Version
}
