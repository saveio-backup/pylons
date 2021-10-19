package common

import (
	"container/list"

	"github.com/saveio/pylons/common/constants"
)

type PaymentType int

const (
	PAYMENT_DIRECT PaymentType = iota
	PAYMENT_MEDIATED
)

type EdgeId [constants.EdgeIdLen]byte

type Address [constants.AddrLen]byte

type Keccak256Slice []Keccak256

type AddressHex string

type BlockExpiration int

type Balance uint64

type BalanceHash [constants.HashLen]byte

type BlockGasLimit int

type BlockGasPrice int

type BlockHash []byte

type BlockHeight uint32

type BlockTimeout int

type ChannelID uint32

type ChannelState int

type LocksRoot [constants.HashLen]byte

type LockHash [constants.HashLen]byte

type MerkleTreeLeaves list.List

type MessageID uint64

type Nonce uint64

type AdditionalHash []byte

type NetworkTimeout float64

type PaymentID int

type PaymentAmount uint64

type FeeAmount uint64

type ProportionalFeeAmount uint64

type PaymentWithFeeAmount FeeAmount

type PaymentNetworkID [20]byte

type ChainID int

type Keccak256 [32]byte

type TokenAddress [constants.AddrLen]byte

type TokenNetworkAddress [constants.AddrLen]byte

type TokenNetworkID [20]byte

type TokenAmount uint64

type TransferID []byte

type Secret []byte

type EncSecret []byte
type SecretHash [constants.HashLen]byte

type SecretRegistryAddress [constants.AddrLen]byte

type Signature []byte

type TransactionHash []byte

type PubKey []byte

var EmptySecretHash = [32]byte{0x00}

var EmptyHashKeccak = [32]byte{0x00}

var EmptySecret = [32]byte{0x00}

var EmptyAddress = Address{0x00}

var EmptyTokenAddress = TokenAddress{0x00}
