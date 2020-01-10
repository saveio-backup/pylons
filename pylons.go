package pylons

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network"
	"github.com/saveio/pylons/service"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/smartcontract/service/native/utils"
)

var Version = "0.1"

type ChannelConfig struct {
	ClientType    string
	ChainNodeURLs []string

	DBPath        string
	BlockDelay    string
	SettleTimeout string
	RevealTimeout string
}

type Channel struct {
	Config  *ChannelConfig
	Service *service.ChannelService
}

func DefaultChannelConfig() *ChannelConfig {
	config := &ChannelConfig{
		ClientType:    "rpc",
		ChainNodeURLs: []string{"http://localhost:20336"},
		DBPath:        ".",
		BlockDelay:    "3",
	}
	return config
}

func NewChannelService(channelConfig *ChannelConfig, account *account.Account) (*Channel, error) {
	settleTimeout, revealTimeout, err := getTimeout(channelConfig)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	blockChainService := network.NewBlockChainService(channelConfig.ClientType, channelConfig.ChainNodeURLs, account)
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
		"database_path":  channelConfig.DBPath,
		"block_delay":    channelConfig.BlockDelay,
		"settle_timeout": strconv.Itoa(settleTimeout),
		"reveal_timeout": strconv.Itoa(revealTimeout),
	}

	if channelConfig.BlockDelay != "" {
		blockDelay, err := strconv.Atoi(channelConfig.BlockDelay)
		if err != nil {
			log.Fatal("Invalid BlockDelay")
			return nil, fmt.Errorf("invalid BlockDelay error:%s", err.Error())
		}
		common.SetMaxBlockDelay(blockDelay)
		log.Infof("[NewChannelService] SetMaxBlockDelay blockDelay: %d", blockDelay)
	}

	service, err := service.NewChannelService(blockChainService, common.BlockHeight(startBlock),
		common.Address(utils.MicroPayContractAddress), new(service.MessageHandler), option)
	log.Info("channel service created, use account ", blockChainService.GetAccount().Address.ToBase58())
	if err != nil {
		return nil, fmt.Errorf("[NewChannelService] NewChannelService error: %s", err.Error())
	}

	channel := &Channel{Config: channelConfig, Service: service}
	return channel, nil
}

func getTimeout(config *ChannelConfig) (settle int, reveal int, err error) {
	settleTimeout := constants.DefaultSettleTimeout
	if config.SettleTimeout != "" {
		settleTimeout, err = strconv.Atoi(config.SettleTimeout)
		if err != nil {
			log.Fatal("invalid settle timeout")
			return 0, 0, err
		}
	}

	revealTimeout := constants.DefaultRevealTimeout
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
		return 0, 0, fmt.Errorf("settle timeout(%d) should be at least double of revealTimeout(%d)",
			settleTimeout, revealTimeout)
	}

	return settleTimeout, revealTimeout, nil
}

func (this *Channel) StartPylons() error {
	return this.Service.StartService()
}

func (this *Channel) StopPylons() {
	this.Service.StopService()
}

func (this *Channel) RegisterReceiveNotification(notificationChannel chan *transfer.EventPaymentReceivedSuccess) {
	this.Service.ReceiveNotificationChannels[notificationChannel] = struct{}{}
}

func (this *Channel) GetVersion() string {
	return Version
}
