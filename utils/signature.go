package utils

import (
	"errors"
	"fmt"

	"github.com/oniio/oniChain/core/types"
	"github.com/oniio/oniChain/crypto/keypair"
	s "github.com/oniio/oniChain/crypto/signature"
	"github.com/oniio/oniChannel/typing"
)

func VerifySignature(pubKey keypair.PublicKey, data []byte, signature []byte) error {
	sigObj, err := s.Deserialize(signature)
	if err != nil {
		return errors.New("invalid signature data: " + err.Error())
	}

	if !s.Verify(pubKey, data, sigObj) {
		return errors.New("signature verification failed")
	}

	return nil
}

func GetAddressFromPubKey(pubKey keypair.PublicKey) typing.Address {
	address := types.AddressFromPubKey(pubKey)

	return typing.Address(address)
}

func GetPublicKey(pubKeyBuf []byte) (keypair.PublicKey, error) {
	pubKey, err := keypair.DeserializePublicKey(pubKeyBuf)
	if err != nil {
		return nil, fmt.Errorf("deserialize publickey error")
	}

	return pubKey, nil
}

func GetPublicKeyBuf(key keypair.PublicKey) []byte {
	return keypair.SerializePublicKey(key)
}
