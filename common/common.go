package common

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"math/rand"
	"os"
	"strconv"
	"sync"

	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/utils/jsonext"
	chainCom "github.com/saveio/themis/common"

	"math/big"
)

var randSrc rand.Source
var randLock sync.RWMutex

func SetRandSeed(id int, selfAddr Address) {
	bigInt := big.NewInt(0).SetBytes(selfAddr[:])
	seed := int64(id) + bigInt.Int64()
	randSrc = rand.NewSource(seed)
}

func GetMsgID() MessageID {
	randLock.Lock()
	defer randLock.Unlock()
	randMsgId := MessageID(randSrc.Int63())
	return randMsgId
}

func AddressEqual(address1 Address, address2 Address) bool {
	result := true

	for i := 0; i < constants.AddrLen; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}

func Keccak256Compare(one *Keccak256, two *Keccak256) int {
	result := 0

	for i := 0; i < constants.HashLen; i++ {
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

func (p Keccak256Slice) Len() int {
	return len(p)
}

func (p Keccak256Slice) Less(i, j int) bool {
	if Keccak256Compare(&p[i], &p[j]) < 0 {
		return true
	} else {
		return false
	}
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
	for i := 0; i < constants.AddrLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.AddrLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *PaymentNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.AddrLen; i++ {
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
	for i := 0; i < constants.AddrLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.AddrLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.AddrLen; i++ {
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
	for i := 0; i < constants.AddrLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.AddrLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *Address) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.AddrLen; i++ {
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

func (self EdgeId) GetAddr1() Address {
	var tmp Address
	copy(tmp[:], self[0:constants.AddrLen])
	return tmp
}

func (self EdgeId) GetAddr2() Address {
	var tmp Address
	copy(tmp[:], self[constants.AddrLen:])
	return tmp
}

func (self EdgeId) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.EdgeIdLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.EdgeIdLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *EdgeId) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.EdgeIdLen; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("EdgeId TextUnmarshaler error!")
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
	for i := 0; i < constants.AddrLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.AddrLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenAddress) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.AddrLen; i++ {
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
	for i := 0; i < constants.HashLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HashLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *SecretHash) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HashLen; i++ {
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

func (self BalanceHash) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyBalanceHash"
	}
	return string(str)
}

func (self BalanceHash) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.HashLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HashLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *BalanceHash) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HashLen; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("BalanceHash TextUnmarshaler error!")
		} else {
			self[i] = byte(res)
		}

		startIdx = toIdx
	}

	return nil
}

func (self LocksRoot) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyLocksRoot"
	}
	return string(str)
}

func (self LocksRoot) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.HashLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HashLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *LocksRoot) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HashLen; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("LocksRoot TextUnmarshaler error!")
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

func LocksRootEmpty(a LocksRoot) bool {
	for _, v := range a {
		if v != 0 {
			return false
		}
	}

	return true
}

func GetHash(data []byte) SecretHash {
	return sha256.Sum256(data)
}

func SecretRandom(len int) []byte {
	buf := make([]byte, len)
	if _, err := rand.Read(buf); err == nil {
		return buf
	} else {
		return nil
	}
}

func ToBase58(address Address) string {
	addr := chainCom.Address(address)
	return addr.ToBase58()
}
func FromBase58(addr string) (Address, error) {
	addressTmp, err := chainCom.AddressFromBase58(addr)
	if err != nil {
		return Address{}, err
	}
	var addrTmp Address
	copy(addrTmp[:], addressTmp[:])
	return addrTmp, nil
}
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func IsEmptyBalanceHash(balanceHash BalanceHash) bool {
	var empty BalanceHash

	return empty == balanceHash
}
