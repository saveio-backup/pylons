package secretcrypt

import (
	"crypto"
	"fmt"

	"github.com/saveio/pylons/common"
	chnSdk "github.com/saveio/themis-go-sdk/channel"
	"github.com/saveio/themis/account"
	comm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/themis/core/types"
	"github.com/saveio/themis/crypto/encrypt"
	"github.com/saveio/themis/crypto/keypair"
	"sync"
)

type SecretCrypt struct {
	account *account.Account
	channel *chnSdk.Channel
	pubKeyTmp *sync.Map
}

var SecretCryptService *SecretCrypt

func NewSecretCryptService(acc *account.Account, chn *chnSdk.Channel) *SecretCrypt {
	SecretCryptService = &SecretCrypt{
		account: acc,
		channel: chn,
	}
	SecretCryptService.pubKeyTmp = new(sync.Map)
	return SecretCryptService
}

func (this *SecretCrypt) EncryptSecret(target common.Address, secret common.Secret) (common.EncSecret, error) {
	//log.Infof("[EncryptSecret] Secret: %s", hex.EncodeToString(secret))
	var err error
	var exist bool
	var pubKey []byte
	var key interface{}

	if key, exist = this.pubKeyTmp.Load(target); !exist {
		log.Debugf("[EncryptSecret] GetNodePubKey target: %s", common.ToBase58(target))
		pubKey, err = this.channel.GetNodePubKey(comm.Address(target))
		if err != nil || len(pubKey) == 0 {
			log.Errorf("[EncryptSecret] GetNodePubKey error: %s", err.Error())
			return nil, fmt.Errorf("[EncryptSecret] GetNodePubKey error: %s", err.Error())
		}
		this.pubKeyTmp.Store(target, pubKey)
	} else {
		pubKey = key.([]byte)
	}

	//log.Infof("[EncryptSecret] Target PublicKey: %s", hex.EncodeToString(pubKey))
	nodePubKey, err := keypair.DeserializePublicKey(pubKey)
	if err != nil {
		log.Errorf("[EncryptSecret] DeserializePublicKey error: %s", err.Error())
		return nil, fmt.Errorf("[EncryptSecret] DeserializePublicKey error: %s", err.Error())
	}

	address := types.AddressFromPubKey(nodePubKey)
	if !common.AddressEqual(common.Address(address), target) {
		log.Errorf("[EncryptSecret] GetNodePubKey error: public key is not match")
		return nil, fmt.Errorf("[EncryptSecret] GetNodePubKey error: public key is not match")
	}

	encSecret, err := encrypt.Encrypt(encrypt.AES128withSHA256, crypto.PublicKey(nodePubKey), secret[:], nil, nil)
	if err != nil {
		log.Errorf("[EncryptSecret] Encrypt error: %s", err.Error())
		return nil, fmt.Errorf("[EncryptSecret] Encrypt error: %s", err.Error())
	}
	if len(encSecret) == 0 {
		log.Errorf("[EncryptSecret] Encrypt error: encSecret length is zero")
		return nil, fmt.Errorf("[EncryptSecret] Encrypt error: encSecret length is zero")
	}
	//log.Infof("[EncryptSecret] EncSecret: %s", hex.EncodeToString(encSecret))
	return encSecret, err
}

func (this *SecretCrypt) DecryptSecret(encSecret common.EncSecret) (common.Secret, error) {
	//log.Info("[DecryptSecret] EncSecret: ", hex.EncodeToString(encSecret))
	prvKey := crypto.PrivateKey(this.account.PrivateKey)
	secret, err := encrypt.Decrypt(prvKey, encSecret, nil, nil)
	//log.Info("[DecryptSecret] Secret: ", hex.EncodeToString(secret))
	return common.Secret(secret), err
}
