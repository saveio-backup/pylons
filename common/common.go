package common

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strconv"

	"crypto/sha256"
	"os"

	chainCom "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/utils/jsonext"
)

func AddressEqual(address1 Address, address2 Address) bool {
	result := true

	for i := 0; i < constants.ADDR_LEN; i++ {
		if address1[i] != address2[i] {
			result = false
			break
		}
	}

	return result
}

func Keccak256Compare(one *Keccak256, two *Keccak256) int {
	result := 0

	for i := 0; i < constants.HASH_LEN; i++ {
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
	for i := 0; i < constants.ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *PaymentNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.ADDR_LEN; i++ {
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
	for i := 0; i < constants.ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenNetworkID) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.ADDR_LEN; i++ {
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
	for i := 0; i < constants.ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *Address) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.ADDR_LEN; i++ {
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
	copy(tmp[:], self[0:constants.ADDR_LEN])
	return tmp
}

func (self EdgeId) GetAddr2() Address {
	var tmp Address
	copy(tmp[:], self[constants.ADDR_LEN:])
	return tmp
}

func (self EdgeId) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.EDGEID_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.EDGEID_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *EdgeId) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.EDGEID_LEN; i++ {
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
	for i := 0; i < constants.ADDR_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.ADDR_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *TokenAddress) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.ADDR_LEN; i++ {
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
	for i := 0; i < constants.HASH_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HASH_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *SecretHash) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HASH_LEN; i++ {
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
	for i := 0; i < constants.HASH_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HASH_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *BalanceHash) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HASH_LEN; i++ {
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

func (self Locksroot) String() string {
	str, err := jsonext.Marshal(self)
	if err != nil {
		return "emptyLocksroot"
	}
	return string(str)
}

func (self Locksroot) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.HASH_LEN; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self[i]), 10)
		e.Write(b)
		if i < constants.HASH_LEN-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	return e.Bytes(), nil
}

func (self *Locksroot) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.HASH_LEN; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("Locksroot TextUnmarshaler error!")
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
