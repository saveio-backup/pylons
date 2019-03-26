package transfer

import (
	"bytes"
	"crypto/sha256"

	mpay "github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/oniio/oniChannel/common"
)

func PackCooperativeSettle(channelId common.ChannelID, participant1 common.Address, participant1Balance common.TokenAmount,
	participant2 common.Address, participant2Balance common.TokenAmount) []byte {

	var buf bytes.Buffer

	buf.Write([]byte(mpay.SIGNATURE_PREFIX))
	buf.Write([]byte(mpay.COSETTLE_MESSAGE_LENGTH))
	buf.Write(Uint64ToBytes(uint64(mpay.COOPERATIVESETTLE)))
	buf.Write(Uint64ToBytes(uint64(channelId)))
	buf.Write(participant1[:])
	buf.Write(Uint64ToBytes(uint64(participant1Balance)))
	buf.Write(participant2[:])
	buf.Write(Uint64ToBytes(uint64(participant2Balance)))

	result := sha256.Sum256(buf.Bytes())

	return result[:]
}
