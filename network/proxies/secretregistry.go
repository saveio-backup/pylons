package proxies

import (
	"crypto/sha256"
	"strings"
	"sync"
	"time"

	chainsdk "github.com/saveio/themis-go-sdk"
	chnsdk "github.com/saveio/themis-go-sdk/channel"
	"github.com/saveio/themis/common/log"
	"github.com/saveio/pylons/common"
)

type SecretRegistry struct {
	Address                    common.SecretRegistryAddress
	ChainClient                *chainsdk.Chain
	ChannelClient              *chnsdk.Channel
	nodeAddress                common.Address
	openSecretTransactions     map[common.SecretHash]*sync.RWMutex // use secret hash as key because secret is defined as slice
	openSecretTransactionsLock sync.Mutex
}

func NewSecretRegistry(chainClient *chainsdk.Chain, secretRegistryAddress common.SecretRegistryAddress) *SecretRegistry {
	self := new(SecretRegistry)

	self.Address = secretRegistryAddress
	self.ChainClient = chainClient
	self.ChannelClient = chainClient.Native.Channel
	self.nodeAddress = common.Address(self.ChannelClient.DefAcc.Address)

	self.openSecretTransactions = make(map[common.SecretHash]*sync.RWMutex)

	return self
}

func (self *SecretRegistry) RegisterSecret(secret common.Secret) {
	self.RegisterSecretBatch([]common.Secret{secret})
	return
}

func (self *SecretRegistry) RegisterSecretBatch(secrets []common.Secret) {
	var secretsToRegister []common.Secret
	var secretHashesToRegister []common.SecretHash
	var secretHashesNotSent []common.SecretHash
	var waitFor []*sync.RWMutex

	self.openSecretTransactionsLock.Lock()

	for _, secret := range secrets {
		secretHash := sha256.Sum256(secret)

		val, exist := self.openSecretTransactions[secretHash]
		if exist {
			// already a transaction for secret register, wait for result
			waitFor = append(waitFor, val)
			secretHashesNotSent = append(secretHashesNotSent, secretHash)
		} else {
			if ok := self.IsSecretRegistered(secretHash); !ok {
				secretsToRegister = append(secretsToRegister, secret)
				secretHashesToRegister = append(secretHashesToRegister, secretHash)

				self.openSecretTransactions[secretHash] = new(sync.RWMutex)

				self.openSecretTransactions[secretHash].Lock()
				defer self.openSecretTransactions[secretHash].Unlock()
			}
		}
	}

	self.openSecretTransactionsLock.Unlock()

	if len(secretsToRegister) == 0 {
		log.Infof("registerSecretBatch skipped, waiting for transactions")

		for _, lock := range waitFor {
			lock.RLock()
			defer lock.RUnlock()
		}

		log.Infof("registerSecretBatch successful")
		return
	}

	var secretsSlice []byte

	for _, secret := range secretsToRegister {
		secretsSlice = append(secretsSlice, secret[:]...)
	}

	txHash, err := self.ChannelClient.RegisterSecretBatch(secretsSlice)
	if err != nil {
		log.Errorf("register secret failed: %s", err)
	}

	confirmed, err := self.ChainClient.PollForTxConfirmed(time.Duration(60)*time.Second, txHash)
	if err != nil || !confirmed {
		log.Errorf("poll register secret tx failed", err.Error())
		return
	}

	self.openSecretTransactionsLock.Lock()

	// Clear openSecretTransactions regardless of the transaction being  successfully executed or not.
	self.openSecretTransactions = make(map[common.SecretHash]*sync.RWMutex)
	self.openSecretTransactionsLock.Unlock()

	if len(waitFor) != 0 {
		log.Infof("registerSecretBatch waiting for pending")
		for _, lock := range waitFor {
			lock.RLock()
			defer lock.RUnlock()
		}
	}

	return
}

func (self *SecretRegistry) GetSecretRegistrationBlockBySecretHash(secretHash common.SecretHash) (common.BlockHeight, error) {

	height, err := self.ChannelClient.GetSecretRevealBlockHeight(secretHash[:])
	if err != nil {
		if !strings.Contains(err.Error(), "GetStorageItem") {
			log.Errorf("GetChannelInfo err:%s", err)
			return 0, err
		} else {
			log.Debugf("secret for hash %v is not register", secretHash)
			return 0, nil
		}
	}

	return common.BlockHeight(height), nil
}

func (self *SecretRegistry) IsSecretRegistered(secretHash common.SecretHash) bool {
	height, err := self.GetSecretRegistrationBlockBySecretHash(secretHash)
	if err == nil && height != 0 {
		return true
	}

	return false
}
