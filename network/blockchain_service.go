package network

import (
	"sync"

	chainsdk "github.com/saveio/themis-go-sdk"
	chnsdk "github.com/saveio/themis-go-sdk/channel"
	"github.com/saveio/themis/account"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/core/types"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/network/proxies"
)

type BlockchainService struct {
	Address       common.Address
	Account       *account.Account
	mutex         sync.Mutex
	ChainClient   *chainsdk.Chain
	ChannelClient *chnsdk.Channel
	currentHeight uint32

	tokenNetwork               *proxies.TokenNetwork
	secretRegistry             *proxies.SecretRegistry
	identifierToPaymentChannel map[common.ChannelID]*proxies.PaymentChannel

	discoveryCreateLock      sync.Mutex
	tokenNetworkCreateLock   sync.Mutex
	paymentChannelCreateLock sync.Mutex
	secretRegistryCreateLock sync.Mutex
}

func NewBlockchainService(clientType string, url string, account *account.Account) *BlockchainService {
	if clientType == "" || url == "" {
		log.Error("chain node url is invalid")
		return nil
	}

	this := &BlockchainService{}
	this.identifierToPaymentChannel = make(map[common.ChannelID]*proxies.PaymentChannel)

	this.ChainClient = chainsdk.NewChain()
	switch clientType {
	case "rpc":
		this.ChainClient.NewRpcClient().SetAddress(url)
	case "ws":
		err := this.ChainClient.NewWebSocketClient().Connect(url)
		if err != nil {
			log.Error("connect websocket error:", err)
			return nil
		}
	case "rest":
		this.ChainClient.NewRestClient().SetAddress(url)
	default:
		log.Error("node url type is invalid")
	}

	if account == nil {
		log.Error("NewBlockchainservice Account is nil")
		return nil
	}

	this.Account = account
	this.ChainClient.SetDefaultAccount(account)
	this.ChannelClient = this.ChainClient.Native.Channel
	log.Info("blockchain service link to", url)
	this.currentHeight, _ = this.BlockHeight()
	this.Address = common.Address(account.Address)
	return this
}

func (this *BlockchainService) GetAccount() *account.Account {
	return this.Account
}

func (this *BlockchainService) BlockHeight() (uint32, error) {
	if height, err := this.ChainClient.GetCurrentBlockHeight(); err == nil {
		return height, nil
	} else {
		return uint32(0), err
	}
}

func (this *BlockchainService) GetBlock(param interface{}) (*types.Block, error) {
	switch (param).(type) {
	case string:
		identifier := param.(string)
		if block, err := this.ChainClient.GetBlockByHash(identifier); err == nil {
			return block, nil
		} else {
			return nil, err
		}
	case uint32:
		identifier := param.(uint32)
		if block, err := this.ChainClient.GetBlockByHeight(identifier); err == nil {
			return block, nil
		} else {
			return nil, err
		}
	default:
		return nil, nil
	}
}

func (this *BlockchainService) SecretRegistry(address common.SecretRegistryAddress) *proxies.SecretRegistry {
	this.secretRegistryCreateLock.Lock()
	defer this.secretRegistryCreateLock.Unlock()

	if this.secretRegistry == nil {

		this.secretRegistry = proxies.NewSecretRegistry(this.ChainClient, address)
	}

	return this.secretRegistry
}

func (this *BlockchainService) NewTokenNetwork(address common.Address) *proxies.TokenNetwork {
	this.tokenNetworkCreateLock.Lock()
	defer this.tokenNetworkCreateLock.Unlock()

	if this.tokenNetwork == nil {

		this.tokenNetwork = proxies.NewTokenNetwork(this.ChainClient, address)
	}

	return this.tokenNetwork
}

func (this *BlockchainService) PaymentChannel(tokenNetworkAddress common.Address,
	channelId common.ChannelID, args map[string]interface{}) *proxies.PaymentChannel {

	this.paymentChannelCreateLock.Lock()
	defer this.paymentChannelCreateLock.Unlock()

	if channel, exist := this.identifierToPaymentChannel[channelId]; exist {
		return channel
	}

	tokenNetwork := this.NewTokenNetwork(tokenNetworkAddress)

	if args == nil {
		return nil
	}

	channel, err := proxies.NewPaymentChannel(tokenNetwork, channelId, args)
	if err != nil {
		return nil
	} else {
		if channel != nil {
			this.identifierToPaymentChannel[channelId] = channel
		}
	}

	return channel
}
