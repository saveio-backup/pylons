package typing

import (
	"bytes"
	"container/list"
	"errors"
	"strconv"

	"github.com/oniio/oniChannel/utils/jsonext"
)

const ADDR_LEN = 20
const HASH_LEN = 32

type Address [ADDR_LEN]byte

func AddressEqual(address1 Address, address2 Address) bool {
	result := true

	for i := 0; i < 20; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}

func Keccak256Compare(one *Keccak256, two *Keccak256) int {
	result := 0

	for i := 0; i < 32; i++ {
		if one[i] > two[i] {
			result = 1
			break
		} else if one[i] < two[i] {
			result = -1
			break
		}
	}

	return result
}

type Keccak256Slice []Keccak256

func (p Keccak256Slice) Len() int {
	return len(p)
}

func (p Keccak256Slice) Less(i, j int) bool {
	if Keccak256Compare(&p[i], &p[j]) < 0 {
		return true
	} else {
		return false
	}

	return true
}

func (p Keccak256Slice) Swap(i, j int) {
	var temp Keccak256

	temp = p[i]
	p[i] = p[j]
	p[j] = temp

	return
}

func (self PaymentNetworkID) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyPaymentNetworkID"
	}
	return string(str)
}

func (self PaymentNetworkID) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *PaymentNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < ADDR_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("PaymentNetworkID TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func (self TokenNetworkID) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyTokenNetworkID"
	}
	return string(str)
}

func (self TokenNetworkID) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < ADDR_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("PaymentNetworkID TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func (self Address) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyAddress"
	}
	return string(str)
}

func (self Address) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *Address) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < ADDR_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("PaymentNetworkID TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func (self TokenAddress) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyAddress"
	}
	return string(str)
}

func (self TokenAddress) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenAddress) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < ADDR_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("PaymentNetworkID TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func (self SecretHash) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptySecretHash"
	}
	return string(str)
}

func (self SecretHash) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < HASH_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < HASH_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *SecretHash) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < HASH_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("PaymentNetworkID TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func SliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func LocksrootEmpty(a Locksroot) bool {
	for _, v := range a {
		if v != 0 {
			return false
		}
	}

	return true
}

type AddressHex string

type BlockExpiration int

type Balance int

type BalanceHash []byte

type BlockGasLimit int

type BlockGasPrice int

type BlockHash []byte

type BlockHeight uint32

type BlockTimeout int

type ChannelID int

type ChannelState int

type InitiatorAddress [ADDR_LEN]byte

type Locksroot [HASH_LEN]byte

type LockHash [HASH_LEN]byte

type MerkleTreeLeaves list.List

type MessageID int64

type Nonce int

type AdditionalHash []byte

type NetworkTimeout float64

type PaymentID int

type PaymentAmount uint64

type PaymentNetworkID [20]byte

type ChainID int

type Keccak256 [32]byte

type TargetAddress [ADDR_LEN]byte

type TokenAddress [ADDR_LEN]byte

type TokenNetworkAddress [ADDR_LEN]byte

type TokenNetworkID [20]byte

type TokenAmount uint64

type TransferID []byte

type Secret []byte

type SecretHash [HASH_LEN]byte

type SecretRegistryAddress []byte

type Signature []byte

type TransactionHash []byte

type PubKey []byte
