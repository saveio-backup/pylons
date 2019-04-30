package transfer

import (
	"bytes"
	"crypto/sha256"

	mpay "github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/saveio/pylons/common"
)

func PackWithdraw(channelId common.ChannelID, participant common.Address, withdrawAmount common.TokenAmount) []byte {
	var buf bytes.Buffer

	buf.Write([]byte(mpay.SIGNATURE_PREFIX))
	buf.Write([]byte(mpay.WITHDRAW_MESSAGE_LENGTH))
	buf.Write(Uint64ToBytes(uint64(mpay.WITHDRAW)))
	buf.Write(Uint64ToBytes(uint64(channelId)))
	buf.Write(participant[:])
	buf.Write(Uint64ToBytes(uint64(withdrawAmount)))

	result := sha256.Sum256(buf.Bytes())
	return result[:]
}
