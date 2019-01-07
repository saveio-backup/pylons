package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"sync/atomic"

	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChain/core/types"
	"github.com/oniio/oniChannel/network/utils"
)

type OntologyClient interface {
	GetCurrentBlockHeight() ([]byte, error)
	GetCurrentBlockHash() ([]byte, error)
	GetVersion() ([]byte, error)
	GetNetworkId() ([]byte, error)
	GetBlockByHash(hash string) ([]byte, error)
	GetBlockByHeight(height uint32) ([]byte, error)
	GetBlockHash(height uint32) ([]byte, error)
	GetBlockHeightByTxHash(txHash string) ([]byte, error)
	GetBlockTxHashesByHeight(height uint32) ([]byte, error)
	GetRawTransaction(txHash string) ([]byte, error)
	GetSmartContract(contractAddress string) ([]byte, error)
	GetSmartContractEvent(txHash string) ([]byte, error)
	GetSmartContractEventByBlock(blockHeight uint32) ([]byte, error)
	GetStorage(contractAddress string, key []byte) ([]byte, error)
	GetMerkleProof(txHash string) ([]byte, error)
	GetMemPoolTxState(txHash string) ([]byte, error)
	GetMemPoolTxCount() ([]byte, error)
	SendRawTransaction(tx *types.Transaction, isPreExec bool) ([]byte, error)
}

const (
	RPC_GET_VERSION                 = "getversion"
	RPC_GET_TRANSACTION             = "getrawtransaction"
	RPC_SEND_TRANSACTION            = "sendrawtransaction"
	RPC_GET_BLOCK                   = "getblock"
	RPC_GET_BLOCK_COUNT             = "getblockcount"
	RPC_GET_BLOCK_HASH              = "getblockhash"
	RPC_GET_CURRENT_BLOCK_HASH      = "getbestblockhash"
	RPC_GET_SMART_CONTRACT_EVENT    = "getsmartcodeevent"
	RPC_GET_STORAGE                 = "getstorage"
	RPC_GET_SMART_CONTRACT          = "getcontractstate"
	RPC_GET_MERKLE_PROOF            = "getmerkleproof"
	RPC_GET_NETWORK_ID              = "getnetworkid"
	RPC_GET_MEM_POOL_TX_COUNT       = "getmempooltxcount"
	RPC_GET_MEM_POOL_TX_STATE       = "getmempooltxstate"
	RPC_GET_BLOCK_TX_HASH_BY_HEIGHT = "getblocktxsbyheight"
	RPC_GET_BLOCK_HEIGHT_BY_TX_HASH = "getblockheightbytxhash"
)

//JsonRpc version
const JSON_RPC_VERSION = "2.0"

type RpcClient struct {
	addr         string
	httpClient   *http.Client
	Account      *account.Account
	qid          uint64
	currentHeigh uint32
}

//JsonRpcRequest object in rpc
type JsonRpcRequest struct {
	Version string        `json:"jsonrpc"`
	Id      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

//JsonRpcResponse object response for JsonRpcRequest
type JsonRpcResponse struct {
	Id     string          `json:"id"`
	Error  int64           `json:"error"`
	Desc   string          `json:"desc"`
	Result json.RawMessage `json:"result"`
}

//NewRpcClient return RpcClient instance
func NewRpcClient(serverAddr string) *RpcClient {
	client := &RpcClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   5,
				DisableKeepAlives:     false, //enable keepalive
				IdleConnTimeout:       time.Second * 300,
				ResponseHeaderTimeout: time.Second * 300,
			},
			Timeout: time.Second * 300, //timeout for http response
		},
		addr: serverAddr,
	}
	height, err := client.GetCurrentBlockHeight()
	if err != nil {
		return nil
	}
	_height, err := utils.GetUint32(height)
	if err != nil {
		return nil
	}
	client.currentHeigh = _height
	return client
}

func (this *RpcClient) getNextQid() string {
	return fmt.Sprintf("%d", atomic.AddUint64(&this.qid, 1))
}

//SetAddress set rpc server address. Simple http://localhost:20336
func (this *RpcClient) SetAddress(addr string) *RpcClient {
	this.addr = addr
	return this
}

//SetHttpClient set http client to RpcClient. In most cases SetHttpClient is not necessary
func (this *RpcClient) SetHttpClient(httpClient *http.Client) *RpcClient {
	this.httpClient = httpClient
	return this
}

//GetVersion return the version of ontology
func (this *RpcClient) GetVersion() ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_VERSION, []interface{}{})
}

func (this *RpcClient) GetNetworkId() ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_NETWORK_ID, []interface{}{})
}

//GetBlockByHash return block with specified block hash in hex string code
func (this *RpcClient) GetBlockByHash(hash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK, []interface{}{hash})
}

//GetBlockByHeight return block by specified block height
func (this *RpcClient) GetBlockByHeight(height uint32) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK, []interface{}{height})
}

//GetBlockCount return the total block count of ontology
func (this *RpcClient) GetBlockCount() ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK_COUNT, []interface{}{})
}

func (this *RpcClient) GetCurrentBlockHeight() ([]byte, error) {
	data, err := this.GetBlockCount()
	if err != nil {
		return nil, err
	}
	count, err := utils.GetUint32(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(count - 1)
}

//GetCurrentBlockHash return the current block hash of ontology
func (this *RpcClient) GetCurrentBlockHash() ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_CURRENT_BLOCK_HASH, []interface{}{})
}

//GetBlockHash return block hash by block height
func (this *RpcClient) GetBlockHash(height uint32) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK_HASH, []interface{}{height})
}

//GetStorage return smart contract storage item.
//addr is smart contact address
//key is the key of value in smart contract
func (this *RpcClient) GetStorage(contractAddress string, key []byte) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_STORAGE, []interface{}{contractAddress, hex.EncodeToString(key)})
}

//GetSmartContractEvent return smart contract event execute by invoke transaction by hex string code
func (this *RpcClient) GetSmartContractEvent(txHash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_SMART_CONTRACT_EVENT, []interface{}{txHash})
}

func (this *RpcClient) GetSmartContractEventByBlock(blockHeight uint32) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_SMART_CONTRACT_EVENT, []interface{}{blockHeight})
}

//GetRawTransaction return transaction by transaction hash
func (this *RpcClient) GetRawTransaction(txHash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_TRANSACTION, []interface{}{txHash})
}

//GetSmartContract return smart contract deployed in ontology by specified smart contract address
func (this *RpcClient) GetSmartContract(contractAddress string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_SMART_CONTRACT, []interface{}{contractAddress})
}

//GetMerkleProof return the merkle proof whether tx is exist in ledger. Param txHash is in hex string code
func (this *RpcClient) GetMerkleProof(txHash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_MERKLE_PROOF, []interface{}{txHash})
}

func (this *RpcClient) GetMemPoolTxState(txHash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_MEM_POOL_TX_STATE, []interface{}{txHash})
}

func (this *RpcClient) GetMemPoolTxCount() ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_MEM_POOL_TX_COUNT, []interface{}{})
}

func (this *RpcClient) GetBlockHeightByTxHash(txHash string) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK_HEIGHT_BY_TX_HASH, []interface{}{txHash})
}

func (this *RpcClient) GetBlockTxHashesByHeight(height uint32) ([]byte, error) {
	return this.sendRpcRequest(RPC_GET_BLOCK_TX_HASH_BY_HEIGHT, []interface{}{height})
}

func (this *RpcClient) SendRawTransaction(tx *types.Transaction, isPreExec bool) ([]byte, error) {
	var buffer bytes.Buffer
	err := tx.Serialize(&buffer)
	if err != nil {
		return nil, fmt.Errorf("serialize error:%s", err)
	}
	txData := hex.EncodeToString(buffer.Bytes())
	params := []interface{}{txData}
	if isPreExec {
		params = append(params, 1)
	}
	return this.sendRpcRequest(RPC_SEND_TRANSACTION, params)
}

//sendRpcRequest send Rpc request to ontology
func (this *RpcClient) sendRpcRequest(method string, params []interface{}) ([]byte, error) {
	rpcReq := &JsonRpcRequest{
		Version: JSON_RPC_VERSION,
		Id:      this.getNextQid(),
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("JsonRpcRequest json.Marsha error:%s", err)
	}
	resp, err := this.httpClient.Post(this.addr, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("http post request:%s error:%s", data, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc response body error:%s", err)
	}
	rpcRsp := &JsonRpcResponse{}
	err = json.Unmarshal(body, rpcRsp)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal JsonRpcResponse:%s error:%s", body, err)
	}
	if rpcRsp.Error != 0 {
		return nil, fmt.Errorf("JsonRpcResponse error code:%d desc:%s result:%s", rpcRsp.Error, rpcRsp.Desc, rpcRsp.Result)
	}
	return rpcRsp.Result, nil
}

func (this *RpcClient) WaitForGenerateBlock(timeout time.Duration, blockCount ...uint32) (bool, error) {
	count := uint32(2)
	if len(blockCount) > 0 && blockCount[0] > 0 {
		count = blockCount[0]
	}
	blockHeight, err := this.GetCurrentBlockHeight()
	if err != nil {
		return false, fmt.Errorf("GetCurrentBlockHeight error:%s", err)
	}
	_blockHeight, _ := utils.GetUint32(blockHeight)

	secs := int(timeout / time.Second)
	if secs <= 0 {
		secs = 1
	}
	for i := 0; i < secs; i++ {
		time.Sleep(time.Second)
		curBlockHeigh, err := this.GetCurrentBlockHeight()
		if err != nil {
			continue
		}
		_curBlockHeigh, _ := utils.GetUint32(curBlockHeigh)
		if _curBlockHeigh-_blockHeight >= count {
			return true, nil
		}
	}
	return false, fmt.Errorf("timeout after %d (s)", secs)
}

func (this *RpcClient) ScanSpecialBlockByTag(tag string, height chan uint32) {
	for {
		newBlockGenerate, err := this.WaitForGenerateBlock(time.Second*1, 1)
		if !newBlockGenerate {
			continue
		}
		if err != nil {
			log.Errorf("wait for generate block err: ", err)
		}
		this.currentHeigh += 1
		block, err := this.GetSmartContractEventByBlock(this.currentHeigh) //raw event without unmarshal
		if err != nil {
			log.Errorf("get smart contract event by block err, msg:%s", err)
		}

		event, err := utils.GetSmartContractEvent(block) //Unmarshal
		if err != nil {
			log.Errorf("get smart contract event by block err, msg:%s", err)
		}
		if event == nil {
			continue
		}
		for _, notify := range event.Notify {
			if _, ok := notify.States.(string); ok {
				if tag == notify.States {
					height <- this.currentHeigh
				}
			}
		}
	}
}

func (this *RpcClient) PollForTxConfirmed(timeout time.Duration, txHash []byte) (bool, error) {
	if len(txHash) == 0 {
		return false, fmt.Errorf("txHash is empty")
	}
	txHashStr := hex.EncodeToString(common.ToArrayReverse(txHash))
	secs := int(timeout / time.Second)
	if secs <= 0 {
		secs = 1
	}
	for i := 0; i < secs; i++ {
		time.Sleep(time.Second)
		ret, err := this.GetBlockHeightByTxHash(txHashStr)
		if err != nil || len(ret) == 0 {
			continue
		}
		_, err = strconv.ParseUint(string(ret), 10, 64)
		if err != nil {
			continue
		}
		return true, nil
	}
	return false, fmt.Errorf("timeout after %d (s)", secs)
}

func (this *RpcClient) FilterAllLogs(toBlock uint32) {

}

func (this *RpcClient) FilterNewLogs(fromBlock, toBlock uint32) {

}
