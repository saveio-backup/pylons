package transfer

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/oniio/oniChannel/typing"
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

func HashBalanceData(transferredAmount typing.TokenAmount,
	lockedAmount typing.TokenAmount, locksroot typing.Locksroot) []byte {

	empty := typing.Locksroot{}

	if transferredAmount == 0 && lockedAmount == 0 && compareLocksroot(locksroot, empty) == true {
		return empty[:]
	}

	buf := new(bytes.Buffer)
	buf.Write(Uint64ToBytes(uint64(transferredAmount)))
	buf.Write(Uint64ToBytes(uint64(lockedAmount)))
	if !compareLocksroot(locksroot, empty) {
		buf.Write(locksroot[:])
	}

	//[TODO] make sure sha256.Sum256 is similar with web3.utils.soliditySha3
	sum := sha256.Sum256(buf.Bytes())

	return sum[:]
}
