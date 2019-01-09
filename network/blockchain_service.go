package network

import (
	"fmt"
	"sync"
	"time"

	chainsdk "github.com/oniio/dsp-go-sdk/chain"
	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/core/types"

	"github.com/oniio/oniChannel/network/contract"
	"github.com/oniio/oniChannel/network/proxies"
	"github.com/oniio/oniChannel/typing"
)

type BlockchainService struct {
	Address       typing.Address
	Account       *account.Account
	mutex         sync.Mutex
	Client        *chainsdk.Chain
	currentHeight uint32

	discovery                  *proxies.Discovery
	tokenNetwork               *proxies.TokenNetwork
	identifierToPaymentChannel map[typing.ChannelID]*proxies.PaymentChannel

	discoveryCreateLock      sync.Mutex
	tokenNetworkCreateLock   sync.Mutex
	paymentChannelCreateLock sync.Mutex
}

/*
if account is nil, get account from wallet.dat;
else use account passed from caller
*/
func NewBlockchainService(clientType string, url string, account *account.Account) *BlockchainService {
	if clientType == "" || url == "" {
		fmt.Printf("chain node url is invalid\n")
		return nil
	}

	this := &BlockchainService{}
	this.identifierToPaymentChannel = make(map[typing.ChannelID]*proxies.PaymentChannel)

	this.Client = chainsdk.NewChain()
	switch clientType {
	case "rpc":
		this.Client.NewRpcClient().SetAddress(url)
	case "ws":
		err := this.Client.NewWebSocketClient().Connect(url)
		if err != nil {
			fmt.Printf("connect websocket error:%s", err)
			return nil
		}
	case "rest":
		this.Client.NewRestClient().SetAddress(url)
	default:
		fmt.Printf("node url type is invalid\n")
	}

	this.currentHeight, _ = this.BlockHeight()

	if account == nil {
		fmt.Printf("NewBlockchainservice Account is nil\n")
		return nil
	}
	this.Account = account
	this.Address = typing.Address(account.Address)
	return this
}

func (this *BlockchainService) GetAccount() *account.Account {
	return this.Account
}

func (this *BlockchainService) UsingContract(contractAddr common.Address) *BlockchainService {
	contractManager := &contract.ContractManager{}
	contractManager.ContractAddress = contractAddr

	/*
		this.ContractManager = contractManager
		if contractAddr == contract.MPAY_CONTRACT_ADDRESS {
			this.ContractManager.Contract.MPayContract.Account = this.Client.Account
		}
	*/
	return this
}

func (this *BlockchainService) BlockHeight() (uint32, error) {
	if height, err := this.Client.GetCurrentBlockHeight(); err == nil {
		return height, nil
	} else {
		return uint32(0), err
	}
}

func (this *BlockchainService) GetBlock(param interface{}) (*types.Block, error) {
	switch (param).(type) {
	case string:
		identifier := param.(string)
		if block, err := this.Client.GetBlockByHash(identifier); err == nil {
			return block, nil
		} else {
			return nil, err
		}
	case uint32:
		identifier := param.(uint32)
		if block, err := this.Client.GetBlockByHeight(identifier); err == nil {
			return block, nil
		} else {
			return nil, err
		}
	default:
		return nil, nil
	}
	return nil, nil
}

func (this *BlockchainService) NextBlock() uint32 {
	currentBlock, _ := this.BlockHeight()
	targetBlockHeight := currentBlock + 1
	for currentBlock < targetBlockHeight {
		currentBlock, _ = this.BlockHeight()
		time.Sleep(time.Second * 1)
	}
	return currentBlock
}

func (this *BlockchainService) GetNewEntries(fromBlock, toBlock uint32) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	//this.ContractManager.MPay.FilterNewLogs(fromBlock, toBlock)
}

func (this *BlockchainService) GetAllEntries() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	//toBlock, _ := this.BlockHeight()
	//this.ContractManager.MPay.FilterAllLogs(toBlock)
}

func (this *BlockchainService) SecretRegistry(address common.Address) {

}

func (this *BlockchainService) Discovery() *proxies.Discovery {
	discovery := &proxies.Discovery{
		ChainClient: this.Client,
		NodeAddress: this.Address,
	}

	return discovery
}

func (this *BlockchainService) TokenNetwork(address typing.Address) *proxies.TokenNetwork {
	this.tokenNetworkCreateLock.Lock()
	defer this.tokenNetworkCreateLock.Unlock()

	//[TODO] should pass *rpc.RpcClient and *rpc.ContractProxy to NewTokenNetwork ?
	if this.tokenNetwork == nil {

		this.tokenNetwork = proxies.NewTokenNetwork(this.Client, address)
	}

	return this.tokenNetwork
}

func (this *BlockchainService) PaymentChannel(tokenNetworkAddress typing.Address,
	channelId typing.ChannelID, args map[string]interface{}) *proxies.PaymentChannel {

	this.paymentChannelCreateLock.Lock()
	defer this.paymentChannelCreateLock.Unlock()

	if channel, exist := this.identifierToPaymentChannel[channelId]; exist {
		return channel
	}

	tokenNetwork := this.TokenNetwork(tokenNetworkAddress)

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
