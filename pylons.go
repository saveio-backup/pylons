package pylons

import (
	"errors"
	"fmt"
	"strconv"

	ch "github.com/saveio/pylons/channelservice"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var Version = "0.1"

type ChannelConfig struct {
	ClientType    string
	ChainNodeURLs []string

	ListenAddress string // protocol + ip + port

	DBPath        string
	SettleTimeout string
	RevealTimeout string
}

type Channel struct {
	Config  *ChannelConfig
	Service *ch.ChannelService
}

func DefaultChannelConfig() *ChannelConfig {
	config := &ChannelConfig{
		ClientType:    "rpc",
		ChainNodeURLs: []string{"http://localhost:20336"},
		ListenAddress: "tpc://127.0.0.1:3001",
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

	blockChainService := network.NewBlockChainService(config.ClientType, config.ChainNodeURLs, account)
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
		"listenAddr":     config.ListenAddress,
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
	return this.Service.StartService()
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
