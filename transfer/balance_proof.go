package transfer

import (
	"bytes"
	"crypto/sha256"

	"github.com/oniio/oniChain/common/log"
	mpay "github.com/oniio/oniChain/smartcontract/service/native/micropayment"
	"github.com/saveio/pylons/common"
)

//[TODO] import from channel_contracts.constants import MessageTypeId
func PackBalanceProof(nonce common.Nonce, balanceHash common.BalanceHash, additionalHash common.AdditionalHash,
	channelId common.ChannelID, tokenNetworkAddr common.TokenNetworkAddress, chainId common.ChainID,
	msgType int) []byte {

	log.Debug("[LockedTransfer DataToSign] balanceHash: ", balanceHash)
	log.Debug("[LockedTransfer DataToSign] additionalHash: ", additionalHash)

	var buf bytes.Buffer

	buf.Write([]byte(mpay.SIGNATURE_PREFIX))
	//[TODO]: should not use with a fixed length, this method may called to construct other packed info
	// currently it is only used for the close channel.
	buf.Write([]byte(mpay.CLOSE_MESSAGE_LENGTH))

	buf.Write(Uint64ToBytes(uint64(msgType)))
	buf.Write(Uint64ToBytes(uint64(channelId)))
	buf.Write(balanceHash[:])
	buf.Write(Uint64ToBytes(uint64(nonce)))
	buf.Write(additionalHash)
	// ignore the token network address and chain ID now
	//buf.Write(tokenNetworkAddr[:])
	//buf.Write(Uint64ToBytes(uint64(chainId)))

	result := sha256.Sum256(buf.Bytes())
	return result[:]
}

func PackBalanceProofUpdate(nonce common.Nonce, balanceHash common.BalanceHash, addtionalHash common.AdditionalHash,
	channelId common.ChannelID, tokenNetworkAddr common.TokenNetworkAddress, chainId common.ChainID, closeSignature common.Signature) []byte {
	var buf bytes.Buffer

	//[TODO] should reuse the packBalanceProof when it returns the []byte instead of the hash
	buf.Write([]byte(mpay.SIGNATURE_PREFIX))
	buf.Write([]byte(mpay.BALANCEPROOF_UPDATE_MESSAGE_LENGTH))
	buf.Write(Uint64ToBytes(uint64(mpay.BalanceProofUpdate)))
	buf.Write(Uint64ToBytes(uint64(channelId)))
	buf.Write(balanceHash[:])
	buf.Write(Uint64ToBytes(uint64(nonce)))
	buf.Write(addtionalHash)
	buf.Write(closeSignature)

	result := sha256.Sum256(buf.Bytes())
	return result[:]
}
