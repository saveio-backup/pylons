package common

import (
	"container/list"

	"github.com/oniio/oniChannel/common/constants"
)

type PaymentType int

const (
	PAYMENT_DIRECT PaymentType = iota
	PAYMENT_MEDIATED
)

type EdgeId [constants.EDGEID_LEN]byte

type Address [constants.ADDR_LEN]byte

type Keccak256Slice []Keccak256

type AddressHex string

type BlockExpiration int

type Balance uint64

type BalanceHash []byte

type BlockGasLimit int

type BlockGasPrice int

type BlockHash []byte

type BlockHeight uint32

type BlockTimeout int

type ChannelID uint32

type ChannelState int

type Locksroot [constants.HASH_LEN]byte

type LockHash [constants.HASH_LEN]byte

type MerkleTreeLeaves list.List

type MessageID int64

type Nonce uint64

type AdditionalHash []byte

type NetworkTimeout float64

type PaymentID int

type PaymentAmount uint64

type PaymentNetworkID [20]byte

type ChainID int

type Keccak256 [32]byte

type TokenAddress [constants.ADDR_LEN]byte

type TokenNetworkAddress [constants.ADDR_LEN]byte

type TokenNetworkID [20]byte

type TokenAmount uint64

type TransferID []byte

type Secret []byte

type SecretHash [constants.HASH_LEN]byte

type SecretRegistryAddress [constants.ADDR_LEN]byte

type Signature []byte

type TransactionHash []byte

type PubKey []byte

var EmptySecretHash = [32]byte{0x00}

var EmptyHashKeccak = [32]byte{0x00}

var EmptySecret = [32]byte{0x00}

var EmptyAddress = Address{0x00}

var EmptyTokenAddress = TokenAddress{0x00}
