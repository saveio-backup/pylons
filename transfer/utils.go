package transfer

import (
	"bytes"
	"encoding/binary"
	"reflect"

	"github.com/saveio/pylons/common"
)

func BytesToUint64(data []byte) uint64 {
	var n uint64
	bytesBuffer := bytes.NewBuffer(data)
	binary.Read(bytesBuffer, binary.LittleEndian, &n)
	return n
}

func Uint64ToBytes(n uint64) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.LittleEndian, n)
	return bytesBuffer.Bytes()
}

func HashBalanceData(transferredAmount common.TokenAmount,
	lockedAmount common.TokenAmount, locksRoot common.LocksRoot) common.BalanceHash {
	var balanceHash common.BalanceHash

	empty := common.LocksRoot{}

	if transferredAmount == 0 && lockedAmount == 0 && compareLocksroot(locksRoot, empty) == true {
		return balanceHash
	}

	buf := new(bytes.Buffer)
	buf.Write(Uint64ToBytes(uint64(transferredAmount)))
	buf.Write(Uint64ToBytes(uint64(lockedAmount)))
	buf.Write(locksRoot[:])

	//[TODO] make sure sha256.Sum256 is similar with web3.utils.soliditySha3
	sum := common.GetHash(buf.Bytes())

	return common.BalanceHash(sum)
}

func IsValidSecretReveal(stateChangeSecret common.Secret, transferSecretHash common.SecretHash, secret common.Secret) bool {
	secretHash := common.GetHash(stateChangeSecret)
	return (0 != bytes.Compare(secret, common.EmptySecret[:])) && secretHash == transferSecretHash
}

func IsStateNil(state State) bool {
	if state == nil {
		return true
	}
	return reflect.ValueOf(state).IsNil()
}
