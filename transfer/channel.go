package transfer

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/saveio/pylons/common"
	"github.com/saveio/themis-go-sdk/usdt"
	"github.com/saveio/themis/common/log"
	"math"
)

func Min(x, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

func Max(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

type BalanceProofData struct {
	locksRoot         common.LocksRoot
	nonce             common.Nonce
	transferredAmount common.TokenAmount
	lockedAmount      common.TokenAmount
}

func IsLockPending(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToLockedLocks.Load(secretHash); exist {
		return true
	} else if _, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		return true
	} else if _, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		return true
	}
	return false
}

func IsLockLocked(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	_, ok := endState.SecretHashesToLockedLocks.Load(secretHash)
	if ok {
		return true
	}
	return false
}

func IsLockExpired(endState *NettingChannelEndState, lock *HashTimeLockState,
	blockNumber common.BlockHeight, lockExpirationThreshold common.BlockHeight) (bool, error) {

	secretHash := common.SecretHash(lock.SecretHash)
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		return false, errors.New("lock has been unlocked on-chain")
	}

	if blockNumber < lockExpirationThreshold {
		return false, fmt.Errorf("current block number: %d is not larger than lockExpirationThreshold: %d",
			blockNumber, lockExpirationThreshold)
	}

	return true, nil
}

func TransferExpired(transfer *LockedTransferSignedState, affectedChannel *NettingChannelState,
	blockNumber common.BlockHeight) bool {
	lockExpirationThreshold := transfer.Lock.Expiration + common.BlockHeight(common.Config.ConfirmBlockCount*2)
	hasLockExpired, _ := IsLockExpired(affectedChannel.OurState, transfer.Lock, blockNumber, lockExpirationThreshold)
	return hasLockExpired
}

func IsSecretKnown(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		return true
	} else if _, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		return true
	}
	return false
}

func IsSecretKnownOffChain(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		return true
	}
	return false
}

func IsSecretKnownOnChain(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		return true
	}
	return false
}

//[NOTE] may use *common.Secret as return value?
func GetSecret(endState *NettingChannelEndState, secretHash common.SecretHash) common.Secret {
	var partialUnlockProof *UnlockPartialProofState
	if v, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		partialUnlockProof = v.(*UnlockPartialProofState)
		return partialUnlockProof.Secret
	}
	if v, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		partialUnlockProof = v.(*UnlockPartialProofState)
		return partialUnlockProof.Secret
	}
	return common.Secret{}
}

func GetLock(endState *NettingChannelEndState, secretHash common.SecretHash) *HashTimeLockState {
	lock, exist := endState.SecretHashesToLockedLocks.Load(secretHash)
	if exist {
		return lock.(*HashTimeLockState)
	}
	partialUnlock, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash)
	if exist {
		return partialUnlock.(*UnlockPartialProofState).Lock
	}
	partialUnlock, exist = endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash)
	if exist {
		return partialUnlock.(*UnlockPartialProofState).Lock
	}
	return nil
}

func LockExistsInEitherChannelSide(channelState *NettingChannelState, secretHash common.SecretHash) bool {
	//"""Check if the lock with `secrethash` exists in either our state or the partner's state"""
	lock := GetLock(channelState.OurState, secretHash)
	log.Debugf("state: %v, hash: %v, ourstate.lock: %v", channelState.OurState, secretHash, lock)
	if lock == nil {
		lock = GetLock(channelState.PartnerState, secretHash)
		log.Debugf("id: %d, Address: %v, partnerstate.lock: %v", channelState.Identifier, common.ToBase58(channelState.PartnerState.Address), lock)
	}
	return lock != nil
}

func DelUnclaimedLock(endState *NettingChannelEndState, secretHash common.SecretHash) {
	if _, exist := endState.SecretHashesToLockedLocks.Load(secretHash); exist {
		endState.SecretHashesToLockedLocks.Delete(secretHash)
	}
	if _, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		endState.SecretHashesToUnLockedLocks.Delete(secretHash)
	}
	return
}

func DelLock(endState *NettingChannelEndState, secretHash common.SecretHash) {
	if IsLockPending(endState, secretHash) == false {
		log.Debug("[DelLock] IsLockPending == false")
		return
	}
	DelUnclaimedLock(endState, secretHash)
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		endState.SecretHashesToOnChainUnLockedLocks.Delete(secretHash)
	}
	return
}

func RegisterSecretEndState(endState *NettingChannelEndState, secret common.Secret, secretHash common.SecretHash) {
	if IsLockLocked(endState, secretHash) {
		pendingLock, _ := endState.SecretHashesToLockedLocks.Load(secretHash)
		endState.SecretHashesToLockedLocks.Delete(secretHash)
		endState.SecretHashesToUnLockedLocks.Store(secretHash, &UnlockPartialProofState{
			Lock:   pendingLock.(*HashTimeLockState),
			Secret: secret,
		})
	}
	return
}

func RegisterOnChainSecretEndState(endState *NettingChannelEndState, secret common.Secret,
	secretHash common.SecretHash, secretRevealBlockNumber common.BlockHeight, deleteLock bool) {

	log.Debugf("[RegisterOnChainSecretEndState] called for : %v", secretHash)
	var pendingLock *HashTimeLockState

	if IsLockLocked(endState, secretHash) == true {
		load, _ := endState.SecretHashesToLockedLocks.Load(secretHash)
		pendingLock = load.(*HashTimeLockState)
	}

	if v, exist := endState.SecretHashesToUnLockedLocks.Load(secretHash); exist {
		pendingLock = v.(*UnlockPartialProofState).Lock
	}

	if pendingLock != nil {
		if pendingLock.Expiration < secretRevealBlockNumber {
			log.Debugf("[RegisterOnChainSecretEndState] secret has expired")
			return
		}

		if deleteLock == true {
			DelLock(endState, secretHash)
		}

		log.Debugf("[RegisterOnChainSecretEndState] register on chain unlock for : %v", secretHash)
		endState.SecretHashesToOnChainUnLockedLocks.Store(secretHash, &UnlockPartialProofState{
			Lock:   pendingLock,
			Secret: secret,
		})
	}
	return
}

func RegisterOffChainSecret(channelState *NettingChannelState, secret common.Secret, secretHash common.SecretHash) {
	ourState := channelState.OurState
	partnerState := channelState.PartnerState

	RegisterSecretEndState(ourState, secret, secretHash)
	RegisterSecretEndState(partnerState, secret, secretHash)
	return
}

func RegisterOnChainSecret(channelState *NettingChannelState, secret common.Secret,
	secretHash common.SecretHash, secretRevealBlockNumber common.BlockHeight, deleteLock bool /*= true*/) {

	ourState := channelState.OurState
	partnerState := channelState.PartnerState

	RegisterOnChainSecretEndState(ourState, secret, secretHash, secretRevealBlockNumber, deleteLock)
	RegisterOnChainSecretEndState(partnerState, secret, secretHash, secretRevealBlockNumber, deleteLock)

	return
}

func compareLocksroot(one common.LocksRoot, two common.LocksRoot) bool {
	length := len(one)
	for i := 0; i < length; i++ {
		if one[i] != two[i] {
			return false
		}
	}
	return true
}

func isDepositConfirmed(channelState *NettingChannelState, blockNumber common.BlockHeight) bool {
	if len(channelState.DepositTransactionQueue) == 0 {
		return false
	}

	result := IsTransactionConfirmed(channelState.DepositTransactionQueue[0].BlockHeight, blockNumber)
	return result
}

func IsTransactionConfirmed(transactionBlockHeight common.BlockHeight, blockChainBlockHeight common.BlockHeight) bool {
	confirmationBlock := transactionBlockHeight + (common.BlockHeight)(common.Config.ConfirmBlockCount)
	return blockChainBlockHeight > confirmationBlock
}

func IsBalanceProofSafeForOnchainOperations(balanceProof *BalanceProofSignedState) bool {
	totalAmount := balanceProof.TransferredAmount + balanceProof.LockedAmount
	return totalAmount <= math.MaxUint64
}

func IsValidRefund(refund *ReceiveTransferRefund, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState,
	receivedTransfer *LockedTransferUnsignedState) (bool, *MerkleTreeState, error) {

	merkleTree, err := ValidLockedTransferCheck(channelState,
		senderState, receiverState, "RefundTransfer", refund.Transfer.BalanceProof, refund.Transfer.Lock)

	if err != nil {
		return false, nil, err
	}
	if !RefundTransferMatchesReceived(refund.Transfer, receivedTransfer) {
		return false, nil, fmt.Errorf("Refund transfer did not match the received transfer")
	}
	return true, merkleTree, nil
}

func IsValidUnlock(unlock *ReceiveUnlock, channelState *NettingChannelState,
	senderState *NettingChannelEndState) (bool, *MerkleTreeState, error) {

	receivedBalanceProof := unlock.BalanceProof
	currentBalanceProof, err := getCurrentBalanceProof(senderState)
	if err != nil {
		log.Error("[IsValidUnlock] error: ", err.Error())
	}

	secretHash := common.GetHash(unlock.Secret)
	log.Debug("[IsValidUnlock] secretHash: ", secretHash)
	lock := GetLock(senderState, secretHash)

	if lock == nil {
		return false, nil, fmt.Errorf("invalid Unlock message. There is no corresponding lock for %s",
			hex.EncodeToString(secretHash[:]))
	}

	lock.LockHash = lock.CalcLockHash()
	merkleTree := computeMerkleTreeWithout(senderState.MerkleTree, lock.LockHash)
	locksRootWithoutLock := MerkleRoot(merkleTree.Layers)

	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount
	expectedTransferredAmount := currentTransferredAmount + lock.Amount
	expectedLockedAmount := currentLockedAmount - lock.Amount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(
		receivedBalanceProof, channelState, senderState)

	if !isBalanceProofUsable {
		return false, nil, fmt.Errorf("invalid Unlock message. %s", invalidBalanceProofMsg)
	} else if receivedBalanceProof.LocksRoot != common.LocksRoot(locksRootWithoutLock) {
		//# Secret messages remove a known lock, the new locksroot must have only
		//# that lock removed, otherwise the sender may be trying to remove
		//# additional locks.
		return false, nil, fmt.Errorf("invalid Unlock message. Balance proof's locksroot didn't match, expected: %s got: %s.",
			hex.EncodeToString(locksRootWithoutLock[:]), hex.EncodeToString(receivedBalanceProof.LocksRoot[:]))
	} else if receivedBalanceProof.TransferredAmount != expectedTransferredAmount {
		//# Secret messages must increase the transferred_amount by lock amount,
		//# otherwise the sender is trying to play the protocol and steal token.
		return false, nil, fmt.Errorf("invalid Unlock message. Balance proof's wrong transferred_amount, expected: %v got: %v.",
			expectedTransferredAmount, receivedBalanceProof.TransferredAmount)
	} else if receivedBalanceProof.LockedAmount != expectedLockedAmount {
		//# Secret messages must increase the transferred_amount by lock amount,
		//# otherwise the sender is trying to play the protocol and steal token.
		return false, nil, fmt.Errorf("invalid Unlock message. Balance proof's wrong locked_amount, expected: %v got: %v.",
			expectedLockedAmount, receivedBalanceProof.LockedAmount)
	} else {
		return true, merkleTree, nil
	}
}

func IsValidAmount(endState *NettingChannelEndState, amount common.TokenAmount) bool {
	balanceProofData, err := getCurrentBalanceProof(endState)
	if err != nil {
		log.Error("[IsValidAmount] error: ", err.Error())
	}
	currentTransferredAmount := balanceProofData.transferredAmount
	currentLockedAmount := balanceProofData.lockedAmount

	transferredAmountAfterUnlock := currentTransferredAmount + currentLockedAmount + amount

	return transferredAmountAfterUnlock <= math.MaxUint64
}

func IsValidSignature(balanceProof *BalanceProofSignedState, senderAddress common.Address) (bool, error) {

	//[TODO] find similar ONT function for
	// to_canonical_address(eth_recover(data=data_that_was_signed, signature=balance_proof.signature))
	balanceHash := HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	dataThatWasSigned := PackBalanceProof(common.Nonce(balanceProof.Nonce), balanceHash, common.AdditionalHash(balanceProof.MessageHash[:]),
		balanceProof.ChannelId, common.TokenNetworkAddress(balanceProof.TokenNetworkId), balanceProof.ChainId, 1)

	if dataThatWasSigned == nil {
		return false, errors.New("Signature invalid, could not be recovered dataThatWasSigned is nil")
	}

	return isValidSignature(dataThatWasSigned, balanceProof.PublicKey, balanceProof.Signature, senderAddress)
}

func IsBalanceProofUsableOnChain(receivedBalanceProof *BalanceProofSignedState,
	channelState *NettingChannelState, senderState *NettingChannelEndState) (bool, error) {

	expectedNonce := getNextNonce(senderState)

	isValidSignature, err := IsValidSignature(receivedBalanceProof, senderState.Address)
	if !isValidSignature {
		return isValidSignature, err
	}
	if GetStatus(channelState) != ChannelStateOpened {
		return false, errors.New("The channel is already closed.")
	} else if receivedBalanceProof.ChannelId != channelState.Identifier {
		return false, fmt.Errorf("channel_identifier does not match. expected:%d got:%d",
			channelState.Identifier, receivedBalanceProof.ChannelId)
	} else if common.AddressEqual(common.Address(receivedBalanceProof.TokenNetworkId),
		common.Address(channelState.TokenNetworkId)) == false {
		return false, fmt.Errorf("token_network_identifier does not match. expected:%d got:%d",
			channelState.TokenNetworkId, receivedBalanceProof.TokenNetworkId)
	} else if receivedBalanceProof.ChainId != channelState.ChainId {
		return false, fmt.Errorf("chain id does not match channel's chain id. expected:%d got:%d",
			channelState.ChainId, receivedBalanceProof.ChainId)
	} else if IsBalanceProofSafeForOnchainOperations(receivedBalanceProof) == false {
		return false, errors.New("Balance proof total transferred amount would overflow onchain.")
	} else if receivedBalanceProof.Nonce != expectedNonce {
		return false, fmt.Errorf("nonce did not change sequentially, expected:%d got:%d",
			expectedNonce, receivedBalanceProof.Nonce)
	} else {
		return true, nil
	}
}

func IsValidDirectTransfer(directTransfer *ReceiveTransferDirect, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState) (bool, error) {

	receivedBalanceProof := directTransfer.BalanceProof
	currentBalanceProof, err := getCurrentBalanceProof(senderState)
	if err != nil {
		log.Error("[IsValidDirectTransfer] error: ", err.Error())
	}
	currentLocksRoot := currentBalanceProof.locksRoot
	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	distributable := GetDistributable(senderState, receiverState)
	amount := receivedBalanceProof.TransferredAmount - currentTransferredAmount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(receivedBalanceProof, channelState, senderState)

	if isBalanceProofUsable == false {
		return false, fmt.Errorf("invalid DirectTransfer message. {%s}", invalidBalanceProofMsg)
	} else if compareLocksroot(receivedBalanceProof.LocksRoot, currentLocksRoot) == false {
		var buf1, buf2 bytes.Buffer

		buf1.Write(currentBalanceProof.locksRoot[:])
		buf2.Write(receivedBalanceProof.LocksRoot[:])
		return false, fmt.Errorf("invalid DirectTransfer message. Balance proof's locksRoot changed, expected:%s got: %s",
			buf1.String(), buf2.String())
	} else if receivedBalanceProof.TransferredAmount <= currentTransferredAmount {
		return false, fmt.Errorf("invalid DirectTransfer message. Balance proof's transferred_amount decreased, expected larger than: %d got: %d",
			currentTransferredAmount, receivedBalanceProof.TransferredAmount)
	} else if receivedBalanceProof.LockedAmount != currentLockedAmount {
		return false, fmt.Errorf("invalid DirectTransfer message. Balance proof's locked_amount is invalid, expected: %d got: %d",
			currentLockedAmount, receivedBalanceProof.LockedAmount)
	} else if amount > distributable {
		return false, fmt.Errorf("invalid DirectTransfer message. Transfer amount larger than the available distributable, transfer amount: %d maximum distributable: %d",
			amount, distributable)
	} else {
		return true, nil
	}
}

func IsValidLockedTransfer(transferState *LockedTransferSignedState, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState) (*MerkleTreeState, error) {
	return ValidLockedTransferCheck(channelState, senderState, receiverState, "LockedTransfer",
		transferState.BalanceProof, transferState.Lock)
}

func IsValidLockExpired(stateChange *ReceiveLockExpired, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState,
	blockNumber common.BlockHeight) (*MerkleTreeState, error) {

	secretHash := stateChange.SecretHash
	receivedBalanceProof := stateChange.BalanceProof

	var lock *HashTimeLockState
	load, _ := channelState.PartnerState.SecretHashesToLockedLocks.Load(secretHash)
	if load == nil {
		return nil, fmt.Errorf("secretHash not found in channel partner's secretHashesToLockedLocks")
	}
	lock = load.(*HashTimeLockState)

	//# If the lock was not found in locked locks, this means that we've received
	//# the secret for the locked transfer but we haven't unlocked it yet. Lock
	//# expiry in this case could still happen which means that we have to make
	//# sure that we check for "unclaimed" locks in our check.
	if lock == nil {
		value, exist := channelState.PartnerState.SecretHashesToUnLockedLocks.Load(secretHash)
		if exist {
			lock = value.(*UnlockPartialProofState).Lock
		}
	}

	lockRegisteredOnChain := false
	if _, exist := channelState.OurState.SecretHashesToOnChainUnLockedLocks.Load(secretHash); exist {
		lockRegisteredOnChain = true
	}

	currentBalanceProof, err := getCurrentBalanceProof(senderState)
	if err != nil {
		log.Error("[IsValidLockExpired] error: ", err.Error())
	}
	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	var merkleTree *MerkleTreeState
	var expectedLockedAmount common.TokenAmount
	if lock != nil {
		lock.LockHash = lock.CalcLockHash()
		merkleTree = computeMerkleTreeWithout(senderState.MerkleTree, lock.LockHash)

		expectedLockedAmount = currentLockedAmount - lock.Amount
	}

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(receivedBalanceProof, channelState, senderState)

	//result: MerkleTreeOrError = (False, None, None)

	if lockRegisteredOnChain {
		return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired mesage. Lock was unlocked on-chain. ")
	} else if lock == nil {
		return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. Lock with secrethash %s is not known. ",
			hex.EncodeToString(secretHash[:]))
	} else if !isBalanceProofUsable {
		return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. %s ", invalidBalanceProofMsg)
	} else if merkleTree == nil {
		return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. Same lockhash handled twice. ")
	} else {
		locksRootWithoutLock := MerkleRoot(merkleTree.Layers)
		hasExpired, err := IsLockExpired(receiverState, lock, blockNumber, lock.Expiration+
			common.BlockHeight(common.Config.ConfirmBlockCount-common.Config.MaxBlockDelay))
		if !hasExpired {
			log.Info("[IsValidLockExpired] blockNumber: %d, lock.Expiration: %d MaxBlockDelay:%d",
				blockNumber, lock.Expiration, common.Config.MaxBlockDelay)
			return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. %s ", err.Error())
		} else if receivedBalanceProof.LocksRoot != common.LocksRoot(locksRootWithoutLock) {
			//The locksRoot must be updated, and the expired lock must be *removed*
			return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. "+
				"Balance proof's locksroot didn't match, expected: %s got: %s. ",
				hex.EncodeToString(locksRootWithoutLock[:]),
				hex.EncodeToString(receivedBalanceProof.LocksRoot[:]))
		} else if receivedBalanceProof.TransferredAmount != currentTransferredAmount {
			//# Given an expired lock, transferred amount should stay the same
			return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. "+
				"Balance proof's transferred_amount changed, expected: %d got: %d. ",
				currentTransferredAmount, receivedBalanceProof.TransferredAmount)
		} else if receivedBalanceProof.LockedAmount != expectedLockedAmount {
			//# locked amount should be the same found inside the balance proof
			return nil, fmt.Errorf("[IsValidLockExpired] Invalid LockExpired message. "+
				"Balance proof's locked_amount is invalid, expected: %d got: %d. ",
				expectedLockedAmount, receivedBalanceProof.LockedAmount)
		} else {
			return merkleTree, nil
		}
	}
}

func ValidLockedTransferCheck(channelState *NettingChannelState, senderState *NettingChannelEndState,
	receiverState *NettingChannelEndState, messageName string, receivedBalanceProof *BalanceProofSignedState,
	lock *HashTimeLockState) (*MerkleTreeState, error) {

	currentBalanceProof, err := getCurrentBalanceProof(senderState)
	if err != nil {
		log.Error("[ValidLockedTransferCheck] error: ", err.Error())
		return nil, fmt.Errorf("[ValidLockedTransferCheck] error: %s", err.Error())
	}
	lock.LockHash = lock.CalcLockHash()
	merkleTree := computeMerkleTreeWith(senderState.MerkleTree, lock.LockHash)

	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	distributable := GetDistributable(senderState, receiverState)
	expectedLockedAmount := currentLockedAmount + lock.Amount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(
		receivedBalanceProof, channelState, senderState)

	if !isBalanceProofUsable {
		return nil, fmt.Errorf("Invalid %s message %s ", messageName, invalidBalanceProofMsg)
	} else if merkleTree == nil {
		return nil, fmt.Errorf("Invalid %s message. Same lockhash handled twice ", messageName)
	} else if MerkleTreeWidth(merkleTree) > MAXIMUM_PENDING_TRANSFERS {
		return nil, fmt.Errorf("Invalid %s message. Adding the transfer would exceed the allowed, "+
			" limit of %d pending transfers per channel. ", messageName, MAXIMUM_PENDING_TRANSFERS)
	} else {
		locksRootWithLock := MerkleRoot(merkleTree.Layers)

		if receivedBalanceProof.LocksRoot != common.LocksRoot(locksRootWithLock) {
			//The locksRoot must be updated to include the new lock
			return nil, fmt.Errorf("Invalid %s message. Balance proof's locksroot didn't match, expected: %s got: %s ",
				messageName, hex.EncodeToString(locksRootWithLock[:]),
				hex.EncodeToString(receivedBalanceProof.LocksRoot[:]))
		} else if receivedBalanceProof.TransferredAmount != currentTransferredAmount {
			//Mediated transfers must not change transferred_amount
			return nil, fmt.Errorf("Invalid %s message. Balance proof's transferred_amount changed, expected: %d got: %d ",
				messageName, currentTransferredAmount, receivedBalanceProof.TransferredAmount)
		} else if receivedBalanceProof.LockedAmount != expectedLockedAmount {
			//Mediated transfers must increase the locked_amount by lock.amount
			return nil, fmt.Errorf("Invalid %s message. Balance proof's locked_amount is invalid, expected: %d got: %d ",
				messageName, expectedLockedAmount, receivedBalanceProof.LockedAmount)
		} else if lock.Amount > distributable {
			//the locked amount is limited to the current available balance, otherwise
			//the sender is attempting to game the protocol and do a double spend
			return nil, fmt.Errorf("Invalid %s message. Lock amount larger than the available distributable, "+
				"lock amount: %d maximum distributable: %d ",
				messageName, lock.Amount, distributable)
		} else if lock.SecretHash == common.EmptyHashKeccak {
			//if the message contains the keccak of the empty hash it will never be
			//usable OnChain https://github.com/raiden-network/raiden/issues/3091
			return nil, fmt.Errorf("Invalid %s message. The secrethash is the keccak of 0x0 and will not be usable OnChain ",
				messageName)
		} else {
			return merkleTree, nil
		}
	}
}

func getAmountLocked(endState *NettingChannelEndState) common.Balance {
	var totalPending, totalUnclaimed, totalUnclaimedOnChain common.TokenAmount

	endState.SecretHashesToLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*HashTimeLockState)
		totalPending = totalPending + v.Amount
		return true
	})
	log.Debug("[getAmountLocked] totalPending: ", totalPending)
	endState.SecretHashesToUnLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*UnlockPartialProofState)
		totalUnclaimed = totalUnclaimed + v.Lock.Amount
		return true
	})
	log.Debug("[getAmountLocked] totalUnclaimed: ", totalUnclaimed)
	endState.SecretHashesToOnChainUnLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*UnlockPartialProofState)
		totalUnclaimedOnChain = totalUnclaimedOnChain + v.Lock.Amount
		return true
	})
	log.Debug("[getAmountLocked] totalUnclaimedOnChain: ", totalUnclaimedOnChain)

	lockedAmount := (common.Balance)(totalPending + totalUnclaimed + totalUnclaimedOnChain)
	log.Debug("[getAmountLocked] lockedAmount: ", lockedAmount)
	return lockedAmount
}

func getBalance(sender *NettingChannelEndState, receiver *NettingChannelEndState) common.Balance {

	var senderTransferredAmount, receiverTransferredAmount common.TokenAmount

	if sender.BalanceProof != nil {
		senderTransferredAmount = sender.BalanceProof.TransferredAmount
	}

	if receiver.BalanceProof != nil {
		receiverTransferredAmount = receiver.BalanceProof.TransferredAmount
	}

	log.Debugf("sender.ContractBalance: %v sender.TotalWithdraw: %v senderTransferredAmount: %v receiverTransferredAmount: %v",
		sender.ContractBalance, sender.TotalWithdraw, senderTransferredAmount, receiverTransferredAmount)
	result := sender.ContractBalance - sender.TotalWithdraw - senderTransferredAmount + receiverTransferredAmount
	return (common.Balance)(result)

}

func getCurrentBalanceProof(endState *NettingChannelEndState) (*BalanceProofData, error) {
	balanceProof := endState.BalanceProof

	if balanceProof != nil {
		return &BalanceProofData{
			locksRoot:         balanceProof.LocksRoot,
			nonce:             balanceProof.Nonce,
			transferredAmount: balanceProof.TransferredAmount,
			lockedAmount:      (common.TokenAmount)(getAmountLocked(endState)),
		}, nil
	} else {
		return &BalanceProofData{
			locksRoot:         common.LocksRoot{},
			nonce:             0,
			transferredAmount: 0,
			lockedAmount:      0,
		}, nil
	}
}

func GetDistributable(sender *NettingChannelEndState, receiver *NettingChannelEndState) common.TokenAmount {
	balanceProofData, err := getCurrentBalanceProof(sender)
	if err != nil || balanceProofData == nil {
		log.Error("[GetDistributable] error: ", err.Error())
		return 0
	}
	transferredAmount := balanceProofData.transferredAmount
	lockedAmount := balanceProofData.lockedAmount
	balance := getBalance(sender, receiver)
	amountLocked := getAmountLocked(sender)

	distributable := balance - amountLocked
	overflowLimit := Max(math.MaxUint64-(uint64)(transferredAmount)-(uint64)(lockedAmount), 0)

	log.Debugf("[GetDistributable] lockedAmount: %v, transferredAmount: %v, balance: %v  amountLocked:%v  distributable: %v overflowLimit: %v",
		lockedAmount, transferredAmount, balance, amountLocked, distributable, overflowLimit)

	result := Min(overflowLimit, (uint64)(distributable))
	return (common.TokenAmount)(result)
}

func getNextNonce(endState *NettingChannelEndState) common.Nonce {
	var nonce common.Nonce

	log.Debug("[getNextNonce]: ", common.ToBase58(endState.Address))

	if endState.BalanceProof != nil {
		nonce = endState.BalanceProof.Nonce + 1
	} else {
		nonce = 1
		log.Debug("[getNextNonce=] endState.BalanceProof == nil")
	}
	log.Debug("[getNextNonce] nonce = ", nonce)
	return nonce
}

func GetBatchUnlock(endState *NettingChannelEndState) []*HashTimeLockState {
	if len(endState.MerkleTree.Layers) == 0 {
		return nil
	}

	lockHashesToLocks := make(map[common.LockHash]*HashTimeLockState)
	endState.SecretHashesToLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*HashTimeLockState)
		lockHashesToLocks[v.LockHash] = v
		return true
	})
	endState.SecretHashesToUnLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*UnlockPartialProofState)
		lockHashesToLocks[v.Lock.LockHash] = v.Lock
		return true
	})
	endState.SecretHashesToOnChainUnLockedLocks.Range(func(key, value interface{}) bool {
		v := value.(*UnlockPartialProofState)
		lockHashesToLocks[v.Lock.LockHash] = v.Lock
		return true
	})

	var orderedLocks []*HashTimeLockState
	for _, lockHash := range endState.MerkleTree.Layers[0] {
		if lock, exist := lockHashesToLocks[common.LockHash(lockHash)]; exist {
			orderedLocks = append(orderedLocks, lock)
		}
	}
	return orderedLocks
}

func MerkleTreeWidth(merkleTree *MerkleTreeState) int {
	if len(merkleTree.Layers) == 0 {
		return 0
	}
	return len(merkleTree.Layers[0])
}

func getNumberOfPendingTransfers(channelEndState *NettingChannelEndState) int {
	if channelEndState.MerkleTree == nil {
		log.Warn("[getNumberOfPendingTransfers] channelEndState.MerkleTree == nil")
		return 0
	}
	return MerkleTreeWidth(channelEndState.MerkleTree)
}

func GetStatus(channelState *NettingChannelState) string {
	var result string
	var finishedSuccessfully, running bool

	if channelState.SettleTransaction != nil {
		finishedSuccessfully = channelState.SettleTransaction.Result == TxnExecSucc
		running = channelState.SettleTransaction.FinishedBlockHeight == 0
		if finishedSuccessfully {
			result = ChannelStateSettled
		} else if running {
			result = ChannelStateSettling
		} else {
			result = ChannelStateUnusable
		}
	} else if channelState.CloseTransaction != nil {
		finishedSuccessfully = channelState.CloseTransaction.Result == TxnExecSucc
		running = channelState.CloseTransaction.FinishedBlockHeight == 0
		if finishedSuccessfully {
			result = ChannelStateClosed
		} else if running {
			result = ChannelStateClosing
		} else {
			result = ChannelStateUnusable
		}
	} else {
		result = ChannelStateOpened
	}
	return result
}

func setClosed(channelState *NettingChannelState, blockNumber common.BlockHeight) {
	if channelState.CloseTransaction == nil {
		channelState.CloseTransaction = &TransactionExecutionStatus{
			StartedBlockHeight:  0,
			FinishedBlockHeight: blockNumber,
			Result:              TxnExecSucc,
		}

	} else if channelState.CloseTransaction.FinishedBlockHeight == 0 {
		channelState.CloseTransaction.FinishedBlockHeight = blockNumber
		channelState.CloseTransaction.Result = TxnExecSucc
	}
}

func setSettled(channelState *NettingChannelState, blockNumber common.BlockHeight) {
	if channelState.SettleTransaction == nil {
		channelState.SettleTransaction = &TransactionExecutionStatus{
			0,
			blockNumber,
			TxnExecSucc}
	} else if channelState.SettleTransaction.FinishedBlockHeight == 0 {
		channelState.SettleTransaction.FinishedBlockHeight = blockNumber
		channelState.SettleTransaction.Result = TxnExecSucc
	}
}

func updateContractBalance(endState *NettingChannelEndState, contractBalance common.Balance) {
	if contractBalance > (common.Balance)(endState.ContractBalance) {
		endState.ContractBalance = (common.TokenAmount)(contractBalance)
	}
}

func computeMerkleTreeWith(merkleTree *MerkleTreeState, lockHash common.LockHash) *MerkleTreeState {
	var result *MerkleTreeState
	log.Debug("[computeMerkleTreeWith] lockHash:", common.LockHashHex(lockHash))
	leaves := merkleTree.Layers[0]
	found := false
	for i := 0; i < len(leaves); i++ {
		temp := common.Keccak256(lockHash)
		if common.Keccak256Compare(&leaves[i], &temp) == 0 {
			found = true
			break
		}
	}

	if found == false {
		newLeaves := make([]common.Keccak256, len(leaves)+1)
		copy(newLeaves, leaves)
		newLeaves[len(leaves)] = common.Keccak256(lockHash)

		newLayers := computeLayers(newLeaves)
		result = &MerkleTreeState{}
		result.Layers = newLayers
	} else {
		log.Warn("[computeMerkleTreeWith] lockHash is found")
	}

	return result
}

func computeMerkleTreeWithout(merkleTree *MerkleTreeState, lockHash common.LockHash) *MerkleTreeState {
	var i int
	found := false
	leaves := merkleTree.Layers[0]
	log.Debug("[computeMerkleTreeWithout] lockHash:", common.LockHashHex(lockHash))
	for i = 0; i < len(leaves); i++ {
		temp := common.Keccak256(lockHash)
		log.Debug("[computeMerkleTreeWithout] leaves[i]:", leaves[i])
		if common.Keccak256Compare(&leaves[i], &temp) == 0 {
			log.Debug("[computeMerkleTreeWithout] found")
			found = true
			break
		}
	}

	var result MerkleTreeState
	if found == true {
		var newLeaves []common.Keccak256
		newLeaves = append(newLeaves, leaves[0:i]...)
		if i+1 < len(leaves) {
			newLeaves = append(newLeaves, leaves[i+1:]...)
		}
		if len(newLeaves) > 0 {
			newLayers := computeLayers(newLeaves)
			result.Layers = newLayers
		} else {
			return GetEmptyMerkleTree()
		}
	}
	return &result
}

func createSendLockedTransfer(channelState *NettingChannelState, initiator common.Address,
	target common.Address, amount common.PaymentAmount, messageId common.MessageID,
	paymentId common.PaymentID, expiration common.BlockExpiration, encSecret common.EncSecret,
	secretHash common.SecretHash, mediators []common.Address) (*SendLockedTransfer, *MerkleTreeState, error) {

	var err error
	ourState := channelState.OurState
	partnerState := channelState.PartnerState
	ourBalanceProof := channelState.OurState.BalanceProof

	if common.TokenAmount(amount) > GetDistributable(ourState, partnerState) {
		err = fmt.Errorf("[createSendLockedTransfer] error: transfer amount is less then distributable")
		log.Error("[createSendLockedTransfer] error: ", err.Error())
		return nil, nil, err
	}

	if GetStatus(channelState) != ChannelStateOpened {
		err = fmt.Errorf("[createSendLockedTransfer] error: channelState is not Opened")
		log.Error("[createSendLockedTransfer] error: ", err.Error())
		return nil, nil, err
	}

	lock := &HashTimeLockState{
		Amount:     common.TokenAmount(amount),
		Expiration: common.BlockHeight(expiration),
		SecretHash: common.Keccak256(secretHash),
	}
	lock.LockHash = lock.CalcLockHash()

	log.Debug("---------------------------------------------------------------------")
	log.Debug("[createSendLockedTransfer] lock.LockHash: ", lock.LockHash)
	log.Debug("[createSendLockedTransfer] computeMerkleTreeWith Before merkleTreeWidth: ", MerkleTreeWidth(ourState.MerkleTree))
	log.Debug(ourState.MerkleTree)

	merkleTree := computeMerkleTreeWith(channelState.OurState.MerkleTree, lock.LockHash)

	log.Debug("[createSendLockedTransfer] computeMerkleTreeWith After  merkleTreeWidth: ", MerkleTreeWidth(merkleTree))
	log.Debug(merkleTree)
	log.Debug("======================================================================")

	locksRoot := MerkleRoot(merkleTree.Layers)
	var transferAmount common.TokenAmount
	if ourBalanceProof != nil {
		transferAmount = ourBalanceProof.TransferredAmount
	} else {
		transferAmount = 0
	}

	token := channelState.TokenAddress
	nonce := getNextNonce(ourState)

	recipient := partnerState.Address
	lockedAmount := getAmountLocked(ourState) + common.Balance(amount)

	balanceProof := &BalanceProofUnsignedState{
		Nonce:             nonce,
		TransferredAmount: transferAmount,
		LockedAmount:      common.TokenAmount(lockedAmount),
		LocksRoot:         common.LocksRoot(locksRoot),
		TokenNetworkId:    channelState.TokenNetworkId,
		ChannelId:         channelState.Identifier,
		ChainId:           channelState.ChainId,
	}

	balanceProof.BalanceHash = HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	transfer := &LockedTransferUnsignedState{
		PaymentId:    paymentId,
		Token:        token,
		BalanceProof: balanceProof,
		Lock:         lock,
		Initiator:    initiator,
		Target:       target,
		EncSecret:    encSecret,
		Mediators:    mediators,
	}

	lockedTransfer := &SendLockedTransfer{
		SendMessageEvent: SendMessageEvent{
			Recipient: recipient,
			ChannelId: channelState.Identifier,
			MessageId: messageId,
		},
		Transfer: transfer,
	}

	return lockedTransfer, merkleTree, nil
}

func createSendDirectTransfer(channelState *NettingChannelState, amount common.PaymentAmount,
	messageId common.MessageID, paymentId common.PaymentID) *SendDirectTransfer {

	ourState := channelState.OurState
	partnerState := channelState.PartnerState

	if common.TokenAmount(amount) > GetDistributable(ourState, partnerState) {
		return nil
	}

	if GetStatus(channelState) != ChannelStateOpened {
		return nil
	}

	ourBalanceProof := channelState.OurState.BalanceProof

	var transferAmount common.TokenAmount
	var locksRoot common.LocksRoot

	if ourBalanceProof != nil {
		transferAmount = common.TokenAmount(amount) + ourBalanceProof.TransferredAmount
		locksRoot = ourBalanceProof.LocksRoot
	} else {
		transferAmount = common.TokenAmount(amount)
		locksRoot = common.LocksRoot{}
	}

	nonce := getNextNonce(ourState)
	recipient := partnerState.Address
	lockedAmount := getAmountLocked(ourState)

	balanceProof := &BalanceProofUnsignedState{
		Nonce:             nonce,
		TransferredAmount: transferAmount,
		LockedAmount:      common.TokenAmount(lockedAmount),
		LocksRoot:         locksRoot,
		TokenNetworkId:    channelState.TokenNetworkId,
		ChannelId:         channelState.Identifier,
		ChainId:           channelState.ChainId,
	}

	balanceProof.BalanceHash = HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	sendDirectTransfer := SendDirectTransfer{
		SendMessageEvent: SendMessageEvent{
			Recipient: recipient,
			ChannelId: channelState.Identifier,
			MessageId: messageId,
		},
		PaymentId:    paymentId,
		BalanceProof: balanceProof,
		TokenAddress: common.TokenAddress(channelState.TokenAddress),
	}

	return &sendDirectTransfer
}

func sendDirectTransfer(channelState *NettingChannelState, amount common.PaymentAmount,
	messageId common.MessageID, paymentId common.PaymentID) *SendDirectTransfer {

	directTransfer := createSendDirectTransfer(channelState, amount, messageId, paymentId)

	//Construct fake BalanceProofSignedState from BalanceProofUnsignedState!
	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:             directTransfer.BalanceProof.Nonce,
		TransferredAmount: directTransfer.BalanceProof.TransferredAmount,
		LockedAmount:      directTransfer.BalanceProof.LockedAmount,
		LocksRoot:         directTransfer.BalanceProof.LocksRoot,
		TokenNetworkId:    directTransfer.BalanceProof.TokenNetworkId,
		ChannelId:         directTransfer.BalanceProof.ChannelId,
		ChainId:           directTransfer.BalanceProof.ChainId,
		BalanceHash:       directTransfer.BalanceProof.BalanceHash,
	}

	return directTransfer
}

func CreateUnlock(channelState *NettingChannelState, messageId common.MessageID,
	paymentId common.PaymentID, secret common.Secret,
	lock *HashTimeLockState) (*SendBalanceProof, *MerkleTreeState, error) {
	ourState := channelState.OurState
	if !IsLockPending(ourState, common.SecretHash(lock.SecretHash)) {
		return nil, nil, fmt.Errorf("caller must make sure the lock is known")
	}
	ourBalanceProof := ourState.BalanceProof

	var transferredAmount common.TokenAmount
	if ourBalanceProof != nil {
		transferredAmount = lock.Amount + ourBalanceProof.TransferredAmount
		log.Debug("[CreateUnlock] lock.Amount: %v, ourBalanceProof.TransferredAmount: %v, transferredAmount: %v\n",
			lock.Amount, ourBalanceProof.TransferredAmount, transferredAmount)
	} else {
		transferredAmount = lock.Amount
		log.Debug("[CreateUnlock] lock.Amount: %v, transferredAmount: %v\n", lock.Amount, transferredAmount)
	}

	lock.LockHash = lock.CalcLockHash()

	log.Debug("---------------------------------------------------------------------")
	log.Debug("[CreateUnlock], lock.LockHash: ", common.LockHashHex(lock.LockHash))
	log.Debug("[CreateUnlock] computeMerkleTreeWithout Before merkleTreeWidth: ", MerkleTreeWidth(ourState.MerkleTree))
	log.Debug(ourState.MerkleTree)
	merkleTree := computeMerkleTreeWithout(ourState.MerkleTree, lock.LockHash)

	log.Debug("[CreateUnlock] computeMerkleTreeWithout After merkleTreeWidth: ", MerkleTreeWidth(merkleTree))
	log.Debug(merkleTree)
	log.Debug("======================================================================")

	locksRoot := MerkleRoot(merkleTree.Layers)

	tokenAddress := channelState.TokenAddress
	nonce := getNextNonce(ourState)
	recipient := channelState.PartnerState.Address

	// the lock is still registered
	amountLocked := common.TokenAmount(getAmountLocked(ourState))
	lockedAmount := amountLocked - lock.Amount
	log.Debug("[CreateUnlock] lockedAmount: %v, amountLocked: %v, lock.Amount: %v",
		lockedAmount, amountLocked, lock.Amount)

	balanceProof := &BalanceProofUnsignedState{
		Nonce:             nonce,
		TransferredAmount: transferredAmount,
		LockedAmount:      lockedAmount,
		LocksRoot:         common.LocksRoot(locksRoot),
		TokenNetworkId:    channelState.TokenNetworkId,
		ChannelId:         channelState.Identifier,
		ChainId:           channelState.ChainId,
	}

	balanceProof.BalanceHash = HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	unlockLock := &SendBalanceProof{
		SendMessageEvent: SendMessageEvent{
			Recipient: recipient,
			ChannelId: channelState.Identifier,
			MessageId: messageId,
		},
		PaymentId:    paymentId,
		TokenAddress: common.TokenAddress(tokenAddress),
		Secret:       secret,
		BalanceProof: balanceProof,
	}

	return unlockLock, merkleTree, nil
}

func sendLockedTransfer(channelState *NettingChannelState, initiator common.Address, target common.Address,
	amount common.PaymentAmount, messageId common.MessageID, paymentId common.PaymentID,
	expiration common.BlockExpiration, encSecret common.EncSecret, secretHash common.SecretHash,
	mediators []common.Address) *SendLockedTransfer {

	sendLockedTransferEvent, merkleTree, err := createSendLockedTransfer(channelState, initiator, target, amount,
		messageId, paymentId, expiration, encSecret, secretHash, mediators)
	if err != nil {
		log.Error("[sendLockedTransfer] createSendLockedTransfer error: %s", err.Error())
		return nil
	}

	transfer := sendLockedTransferEvent.Transfer
	lock := transfer.Lock

	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:             transfer.BalanceProof.Nonce,
		TransferredAmount: transfer.BalanceProof.TransferredAmount,
		LockedAmount:      transfer.BalanceProof.LockedAmount,
		LocksRoot:         transfer.BalanceProof.LocksRoot,
		TokenNetworkId:    transfer.BalanceProof.TokenNetworkId,
		ChannelId:         transfer.BalanceProof.ChannelId,
		ChainId:           transfer.BalanceProof.ChainId,
		BalanceHash:       transfer.BalanceProof.BalanceHash,
	}
	channelState.OurState.MerkleTree = merkleTree

	hashHex := common.SecretHashHex(common.SecretHash(lock.SecretHash))
	log.Debugf("[SendLockedTransfer] Add SecretHash %s To LockedLocks", hashHex)
	channelState.OurState.SecretHashesToLockedLocks.Store(common.SecretHash(lock.SecretHash), lock)

	return sendLockedTransferEvent
}

func sendRefundTransfer(channelState *NettingChannelState, initiator common.Address,
	target common.Address, amount common.PaymentAmount, messageId common.MessageID,
	paymentId common.PaymentID, expiration common.BlockExpiration,
	encSecret common.EncSecret, secretHash common.SecretHash) (*SendRefundTransfer, error) {

	_, ok := channelState.PartnerState.SecretHashesToLockedLocks.Load(secretHash)
	if !ok {
		return nil, fmt.Errorf("Refunds are only valid for *known and pending* transfers ")
	}

	sendMediatedTransfer, merkleTree, err := createSendLockedTransfer(
		channelState, initiator, target, amount, messageId, paymentId,
		expiration, encSecret, secretHash, nil)
	if err != nil {
		log.Error("[sendRefundTransfer] createSendLockedTransfer error: %s", err.Error())
		return nil, err
	}
	mediatedTransfer := sendMediatedTransfer.Transfer
	lock := mediatedTransfer.Lock

	//todo (do what?
	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:             mediatedTransfer.BalanceProof.Nonce,
		TransferredAmount: mediatedTransfer.BalanceProof.TransferredAmount,
		LockedAmount:      mediatedTransfer.BalanceProof.LockedAmount,
		LocksRoot:         mediatedTransfer.BalanceProof.LocksRoot,
		TokenNetworkId:    mediatedTransfer.BalanceProof.TokenNetworkId,
		ChannelId:         mediatedTransfer.BalanceProof.ChannelId,
		ChainId:           mediatedTransfer.BalanceProof.ChainId,
		BalanceHash:       mediatedTransfer.BalanceProof.BalanceHash,
	}
	channelState.OurState.MerkleTree = merkleTree
	channelState.OurState.SecretHashesToLockedLocks.Store(common.SecretHash(lock.SecretHash), lock)

	refundTransfer := RefundFromSendmediated(sendMediatedTransfer)
	return refundTransfer, nil
}

func SendUnlock(channelState *NettingChannelState, messageId common.MessageID,
	paymentId common.PaymentID, secret common.Secret, secretHash common.SecretHash) *SendBalanceProof {

	lock := GetLock(channelState.OurState, secretHash)
	if lock == nil {
		return nil
	}

	unlock, merkleTree, _ := CreateUnlock(channelState, messageId, paymentId, secret, lock)

	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:             unlock.BalanceProof.Nonce,
		TransferredAmount: unlock.BalanceProof.TransferredAmount,
		LockedAmount:      unlock.BalanceProof.LockedAmount,
		LocksRoot:         unlock.BalanceProof.LocksRoot,
		TokenNetworkId:    unlock.BalanceProof.TokenNetworkId,
		ChannelId:         unlock.BalanceProof.ChannelId,
		ChainId:           unlock.BalanceProof.ChainId,
		BalanceHash:       unlock.BalanceProof.BalanceHash,
	}
	channelState.OurState.MerkleTree = merkleTree
	DelLock(channelState.OurState, common.SecretHash(lock.SecretHash))
	return unlock
}

func EventsForClose(channelState *NettingChannelState, blockNumber common.BlockHeight) []Event {
	var events []Event

	status := GetStatus(channelState)
	log.Debugf("channel status while channel close event: %s", status)
	if status == ChannelStateOpened || status == ChannelStateClosing {
		channelState.CloseTransaction = &TransactionExecutionStatus{
			blockNumber, 0, ""}

		closeEvent := &ContractSendChannelClose{ContractSendEvent{},
			channelState.Identifier, common.TokenAddress(channelState.TokenAddress),
			channelState.TokenNetworkId, channelState.PartnerState.BalanceProof}

		events = append(events, closeEvent)
	}

	return events
}

func createSendExpiredLock(senderEndState *NettingChannelEndState, lockedLock *HashTimeLockState,
	chainId common.ChainID, TokenNetworkId common.TokenNetworkID,
	channelId common.ChannelID, recipient common.Address) (*SendLockExpired, *MerkleTreeState) {

	lockedAmount := getAmountLocked(senderEndState)
	balanceProof := senderEndState.BalanceProof
	updatedLockedAmount := common.TokenAmount(lockedAmount) - lockedLock.Amount

	if balanceProof == nil {
		log.Errorf("createSendExpiredLock error: balanceProof is nil!")
		//there should be a balance proof because a lock is expiring
		return nil, nil
	}
	transferredAmount := balanceProof.TransferredAmount

	merkleTree := computeMerkleTreeWithout(senderEndState.MerkleTree, lockedLock.LockHash)
	if merkleTree == nil {
		log.Errorf("computeMerkleTreeWithout error: merkleTree is nil!")
		return nil, nil
	}

	nonce := getNextNonce(senderEndState)
	locksRoot := MerkleRoot(merkleTree.Layers)

	//todo: check
	balanceProofEx := &BalanceProofUnsignedState{
		Nonce:             nonce,
		TransferredAmount: transferredAmount,
		LockedAmount:      updatedLockedAmount,
		LocksRoot:         common.LocksRoot(locksRoot),
		TokenNetworkId:    TokenNetworkId,
		ChannelId:         channelId,
		ChainId:           chainId,
	}

	balanceProof.BalanceHash = HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	// calculate ex balance proof balance hash with updated locked amount and locks root
	balanceProofEx.BalanceHash = HashBalanceData(transferredAmount, updatedLockedAmount, common.LocksRoot(locksRoot))

	sendLockExpired := &SendLockExpired{
		SendMessageEvent: SendMessageEvent{
			Recipient: recipient,
			ChannelId: channelId,
			MessageId: common.GetMsgID(),
		},
		BalanceProof: balanceProofEx,
		SecretHash:   common.SecretHash(lockedLock.SecretHash),
	}
	return sendLockExpired, merkleTree
}

func EventsForExpiredLock(channelState *NettingChannelState, lockedLock *HashTimeLockState) []Event {
	var lockExpired []Event
	sendLockExpired, merkleTree := createSendExpiredLock(channelState.OurState, lockedLock, channelState.ChainId,
		channelState.TokenNetworkId, channelState.Identifier, channelState.PartnerState.Address,
	)

	if sendLockExpired != nil {
		channelState.OurState.MerkleTree = merkleTree
		channelState.OurState.BalanceProof = &BalanceProofSignedState{
			Nonce:             sendLockExpired.BalanceProof.Nonce,
			TransferredAmount: sendLockExpired.BalanceProof.TransferredAmount,
			LockedAmount:      sendLockExpired.BalanceProof.LockedAmount,
			LocksRoot:         sendLockExpired.BalanceProof.LocksRoot,
			BalanceHash:       sendLockExpired.BalanceProof.BalanceHash,
			TokenNetworkId:    sendLockExpired.BalanceProof.TokenNetworkId,
			ChannelId:         sendLockExpired.BalanceProof.ChannelId,
			ChainId:           sendLockExpired.BalanceProof.ChainId,
		}

		DelUnclaimedLock(channelState.OurState, common.SecretHash(lockedLock.SecretHash))

		lockExpired = append(lockExpired, sendLockExpired)
		return lockExpired
	}
	return nil
}

func handleSendDirectTransfer(channelState *NettingChannelState, stateChange *ActionTransferDirect) TransitionResult {

	var events []Event

	amount := stateChange.Amount
	paymentId := stateChange.PaymentId
	targetAddress := stateChange.ReceiverAddress
	distributableAmount := GetDistributable(channelState.OurState, channelState.PartnerState)

	currentBalanceProof, err := getCurrentBalanceProof(channelState.OurState)
	if err != nil {
		log.Error("[handleSendDirectTransfer] error: ", err.Error())
	}
	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	transferedAmountAfterUnlock := currentTransferredAmount + amount + currentLockedAmount

	isOpen := false
	if GetStatus(channelState) == ChannelStateOpened {
		isOpen = true
	}

	isValid := false
	if amount > 0 && transferedAmountAfterUnlock < math.MaxUint64 {
		isValid = true
	}

	canPay := false
	if amount <= distributableAmount {
		canPay = true
	}
	if isOpen && isValid && canPay {
		messageId := common.GetMsgID()
		directTransfer := sendDirectTransfer(channelState, common.PaymentAmount(amount), messageId, paymentId)
		events = append(events, directTransfer)
	} else {
		if isOpen == false {
			msg := fmt.Sprintf("Channel is not opened")
			failure := &EventPaymentSentFailed{
				PaymentNetworkId: channelState.PaymentNetworkId,
				TokenNetworkId:   channelState.TokenNetworkId,
				Identifier:       paymentId,
				Target:           common.Address(targetAddress),
				Reason:           msg,
			}
			log.Warn("[handleSendDirectTransfer] failure: ", msg)
			events = append(events, failure)
		} else if isValid == false {
			msg := fmt.Sprintf("Payment amount is invalid. Transfer %d", amount)
			failure := &EventPaymentSentFailed{
				PaymentNetworkId: channelState.PaymentNetworkId,
				TokenNetworkId:   channelState.TokenNetworkId,
				Identifier:       paymentId,
				Target:           targetAddress,
				Reason:           msg,
			}
			log.Warn("[handleSendDirectTransfer] failure: ", msg)
			events = append(events, failure)
		} else if canPay == false {
			msg := fmt.Sprintf("Payment amount exceeds the available capacity. Capacity:%d, Transfer:%d",
				distributableAmount, amount)
			failure := &EventPaymentSentFailed{
				PaymentNetworkId: channelState.PaymentNetworkId,
				TokenNetworkId:   channelState.TokenNetworkId,
				Identifier:       paymentId,
				Target:           targetAddress,
				Reason:           msg,
			}
			log.Warn("[handleSendDirectTransfer] failure: ", msg)
			events = append(events, failure)
		}
	}
	return TransitionResult{channelState, events}
}

func handleSendWithdrawRequest(channelState *NettingChannelState, stateChange *ActionWithdraw, blockNumber common.BlockHeight) TransitionResult {
	var events []Event
	var err error

	isOpen := false
	isValid := false

	if GetStatus(channelState) == ChannelStateOpened {
		isOpen = true
		isValid, err = isValidWithdrawAmount(channelState.GetChannelEndState(0), channelState.GetChannelEndState(1), stateChange.TotalWithdraw)
	} else {
		err = errors.New("channel is not opened")
	}

	if isOpen && isValid {
		messageId := common.GetMsgID()
		sendWithdrawRequest := &SendWithdrawRequest{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.Partner,
				ChannelId: stateChange.ChannelId,
				MessageId: messageId,
			},

			Participant:    stateChange.Participant,
			TokenNetworkId: channelState.TokenNetworkId,
			WithdrawAmount: stateChange.TotalWithdraw,
		}

		events = append(events, sendWithdrawRequest)
		RecordWithdrawTransaction(channelState, blockNumber)
	} else {
		failure := &EventWithdrawRequestSentFailed{
			ChannelId:      stateChange.ChannelId,
			TokenNetworkId: stateChange.TokenNetworkId,
			WithdrawAmount: stateChange.TotalWithdraw,
			Reason:         err.Error(),
		}
		log.Warn("[handleSendWithdrawRequest] failure: ", err.Error())
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}

func isValidWithdrawAmount(participant *NettingChannelEndState, partner *NettingChannelEndState, totalWithdraw common.TokenAmount) (bool, error) {
	currentWithdraw := participant.GetTotalWithdraw()
	amountToWithdraw := totalWithdraw - currentWithdraw
	log.Debugf("withdraw total: %d, current: %d, amount: %d", totalWithdraw, currentWithdraw, amountToWithdraw)

	if totalWithdraw < currentWithdraw {
		return false, fmt.Errorf("total withdraw smaller than current, totalWithdraw: %d, currentWithdraw: %d",
			totalWithdraw, currentWithdraw)
	}

	if amountToWithdraw <= 0 {
		return false, errors.New("amount to withdraw no larger than 0")
	}

	bothDeposit := participant.GetContractBalance() + partner.GetContractBalance()
	bothWithdraw := totalWithdraw + partner.GetTotalWithdraw()
	if bothWithdraw > bothDeposit {
		return false, fmt.Errorf("we total withdraw %d, partner total withdraw %d, "+
			"both side total withdraw %d large than both side total deposit %d",
			totalWithdraw, partner.GetTotalWithdraw(), bothWithdraw, bothDeposit)
	}

	// NOTE: GetDistributable already take current withdraw into account
	distributable := GetDistributable(participant, partner)
	if amountToWithdraw > distributable {
		return false, fmt.Errorf("total withdraw %d is larger than distributable %d", totalWithdraw, distributable)
	}

	return true, nil
}

func RecordWithdrawTransaction(channelState *NettingChannelState, height common.BlockHeight) {
	log.Debugf("[RecordWithdrawTransaction] for channel %d at height %d", uint32(channelState.Identifier), height)
	channelState.WithdrawTransaction = &TransactionExecutionStatus{height, 0, ""}
}

func DeleteWithdrawTransaction(channelState *NettingChannelState) {
	log.Debugf("[DeleteWithdrawTransaction] for channel %d", uint32(channelState.Identifier))
	channelState.WithdrawTransaction = nil
}

func GetWithdrawTransaction(channelState *NettingChannelState) *TransactionExecutionStatus {
	return channelState.WithdrawTransaction
}

func handleWithdrawRequestReceived(channelState *NettingChannelState, stateChange *ReceiveWithdrawRequest) TransitionResult {
	var events []Event

	valid, err := isValidWithdrawRequest(channelState, stateChange)
	if valid {
		messageId := common.GetMsgID()
		sendWithdraw := &SendWithdraw{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.Participant,
				ChannelId: stateChange.ChannelId,
				MessageId: messageId,
			},
			Participant:          stateChange.Participant,
			TokenNetworkId:       channelState.TokenNetworkId,
			WithdrawAmount:       stateChange.TotalWithdraw,
			ParticipantSignature: stateChange.ParticipantSignature,
			ParticipantAddress:   stateChange.ParticipantAddress,
			ParticipantPublicKey: stateChange.ParticipantPublicKey,
		}

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.Participant,
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}

		events = append(events, sendWithdraw)
		events = append(events, sendProcessed)
	} else {
		failure := &EventInvalidReceivedWithdrawRequest{
			ChannelId:     stateChange.ChannelId,
			Participant:   stateChange.Participant,
			TotalWithdraw: stateChange.TotalWithdraw,
			Reason:        err.Error(),
		}
		log.Warn("[handleWithdrawRequestReceived] failure: ", err.Error())
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}

func isValidWithdrawRequest(channelState *NettingChannelState, stateChange *ReceiveWithdrawRequest) (bool, error) {
	if GetStatus(channelState) != ChannelStateOpened {
		return false, errors.New("channel is not opened")
	}
	// check if participant is valid,
	partnerState := channelState.PartnerState
	if !common.AddressEqual(partnerState.Address, stateChange.Participant) {
		return false, errors.New("participant address invalid")
	}

	if !common.AddressEqual(stateChange.Participant, stateChange.ParticipantAddress) {
		return false, errors.New("participant address is same as the signer")
	}

	isValid, err := isValidWithdrawAmount(channelState.GetChannelEndState(1), channelState.GetChannelEndState(0), stateChange.TotalWithdraw)
	if !isValid {
		return false, err
	}

	// verify if signature is valid
	dataToSign := PackWithdraw(stateChange.ChannelId, stateChange.Participant, stateChange.TotalWithdraw)
	return isValidSignature(dataToSign, stateChange.ParticipantPublicKey, stateChange.ParticipantSignature, stateChange.ParticipantAddress)
}

func isValidSignature(dataToSign []byte, publicKey common.PubKey, signature common.Signature, senderAddress common.Address) (bool, error) {
	if len(dataToSign) == 0 {
		return false, errors.New("Signature invalid, dataToSign is nil")
	}

	pubKey, err := common.GetPublicKey(publicKey)
	if err != nil {
		return false, errors.New("Failed to get public key")
	}

	err = common.VerifySignature(pubKey, dataToSign, signature)
	if err != nil {
		return false, errors.New("Signature invalid, could not be recovered")
	}

	signerAddress := common.GetAddressFromPubKey(pubKey)
	if common.AddressEqual(senderAddress, signerAddress) == true {
		return true, nil
	} else {
		log.Debugf("PubKey:%v, senderAddr: %s", pubKey, common.ToBase58(senderAddress))
		return false, errors.New("Signature was valid but the expected address does not match.")
	}
}

func handleWithdrawReceived(channelState *NettingChannelState, stateChange *ReceiveWithdraw) TransitionResult {
	var events []Event

	valid, err := isValidWithdraw(channelState, stateChange)
	if valid {
		contractSendChannelWithdraw := &ContractSendChannelWithdraw{
			ChannelId:            stateChange.ChannelId,
			TokenNetworkId:       stateChange.TokenNetworkId,
			Participant:          stateChange.Participant,
			TotalWithdraw:        stateChange.TotalWithdraw,
			ParticipantSignature: stateChange.ParticipantSignature,
			ParticipantAddress:   stateChange.ParticipantAddress,
			ParticipantPublicKey: stateChange.ParticipantPublicKey,
			PartnerSignature:     stateChange.PartnerSignature,
			PartnerAddress:       stateChange.PartnerAddress,
			PartnerPublicKey:     stateChange.PartnerPublicKey,
		}

		events = append(events, contractSendChannelWithdraw)
	} else {
		failure := &EventInvalidReceivedWithdraw{
			TokenNetworkId: stateChange.TokenNetworkId,
			ChannelId:      stateChange.ChannelId,
			Participant:    stateChange.Participant,
			TotalWithdraw:  stateChange.TotalWithdraw,
			Reason:         err.Error(),
		}
		log.Warn("[handleWithdrawReceived] failure: ", err.Error())
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}
func isValidWithdraw(channelState *NettingChannelState, stateChange *ReceiveWithdraw) (bool, error) {
	if GetStatus(channelState) != ChannelStateOpened {
		return false, errors.New("channel is not opened")
	}
	// check if participant is valid,
	ourState := channelState.OurState
	if !common.AddressEqual(ourState.Address, stateChange.Participant) {
		return false, errors.New("[isValidWithdraw] participant address invalid")
	}

	// verify if signature is valid
	dataToSign := PackWithdraw(stateChange.ChannelId, stateChange.Participant, stateChange.TotalWithdraw)
	return isValidSignature(dataToSign, stateChange.PartnerPublicKey, stateChange.PartnerSignature, stateChange.PartnerAddress)
}

func handleChannelWithdraw(channelState *NettingChannelState, stateChange *ContractReceiveChannelWithdraw) TransitionResult {
	var events []Event

	ourState := channelState.GetChannelEndState(0)
	partnerState := channelState.GetChannelEndState(1)

	log.Infof("handleChannelWithdraw withdraw amount is %d", stateChange.TotalWithdraw)

	if common.AddressEqual(stateChange.Participant, ourState.GetAddress()) {
		ourState.TotalWithdraw = stateChange.TotalWithdraw
	} else {
		partnerState.TotalWithdraw = stateChange.TotalWithdraw
	}

	DeleteWithdrawTransaction(channelState)

	return TransitionResult{channelState, events}
}

func handleSendCooperativeSettleRequest(channelState *NettingChannelState, stateChange *ActionCooperativeSettle) TransitionResult {
	var events []Event

	if GetStatus(channelState) == ChannelStateOpened {
		ourBalance, partnerBalance := GetCooprativeSettleBalances(channelState)

		messageId := common.GetMsgID()
		ourAddress := channelState.OurState.GetAddress()
		partnerAddress := channelState.PartnerState.GetAddress()

		sendCooperativeSettleRequest := &SendCooperativeSettleRequest{
			SendMessageEvent: SendMessageEvent{
				Recipient: partnerAddress,
				ChannelId: stateChange.ChannelId,
				MessageId: messageId,
			},
			TokenNetworkId:      stateChange.TokenNetworkId,
			Participant1:        ourAddress,
			Participant1Balance: ourBalance,
			Participant2:        partnerAddress,
			Participant2Balance: partnerBalance,
		}

		//record there is an cooperative settlement ongoing
		channelState.SettleTransaction = &TransactionExecutionStatus{0, 0, ""}

		events = append(events, sendCooperativeSettleRequest)
	} else {
		msg := fmt.Sprintf("Channel is not opened")
		failure := &EventCooperativeSettleRequestSentFailed{
			TokenNetworkId: stateChange.TokenNetworkId,
			ChannelId:      stateChange.ChannelId,
			Reason:         msg,
		}
		log.Warn("[handleSendCooperativeSettleRequest] failure: ", msg)
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}

func GetCooprativeSettleBalances(channelState *NettingChannelState) (ourBalance common.TokenAmount, partnerBalance common.TokenAmount) {
	// B1 : balance of P1
	// D1 : total deposit of P1
	// W1 : total withdraw of P1
	// T1 : transferred amount from P1 to P2
	// Lc1: Locked amount that will be transferred to P2 (details to be confirmed)
	// TAD : Total available deposit
	//	(1) B1 = D1 - W1 + T2 - T1 + Lc2 - Lc1
	//	(2) B2 = D2 - W2 + T1 - T2 + Lc1 - Lc2
	//	(3) B1 + B2 = TAD

	ourState := channelState.GetChannelEndState(0)
	partnerState := channelState.GetChannelEndState(1)

	ourBalance = calculateCooperativeSettleBalance(ourState, partnerState)
	partnerBalance = calculateCooperativeSettleBalance(partnerState, ourState)

	log.Debugf("GetCooprativeSettleBalances, ourBalance %d, partnerBalance %d", ourBalance, partnerBalance)

	//fwtodo : add checking (3)

	return ourBalance, partnerBalance
}

func calculateCooperativeSettleBalance(participant1 *NettingChannelEndState, participant2 *NettingChannelEndState) common.TokenAmount {
	bpData1, _ := getCurrentBalanceProof(participant1)
	bpData2, _ := getCurrentBalanceProof(participant2)

	balance := participant1.ContractBalance - participant1.TotalWithdraw + bpData2.transferredAmount - bpData1.transferredAmount + bpData2.lockedAmount - bpData1.lockedAmount
	return balance
}

func handleCooperativeSettleRequestReceived(channelState *NettingChannelState, stateChange *ReceiveCooperativeSettleRequest) TransitionResult {
	var events []Event

	valid, err := isValidCooperativeSettleRequest(channelState, stateChange)
	if valid {
		messageId := common.GetMsgID()
		sendCooperativeSettle := &SendCooperativeSettle{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.Participant1,
				ChannelId: stateChange.ChannelId,
				MessageId: messageId,
			},
			TokenNetworkId:        stateChange.TokenNetworkId,
			Participant1:          stateChange.Participant1,
			Participant1Balance:   stateChange.Participant1Balance,
			Participant2:          stateChange.Participant2,
			Participant2Balance:   stateChange.Participant2Balance,
			Participant1Signature: stateChange.Participant1Signature,
			Participant1Address:   stateChange.Participant1Address,
			Participant1PublicKey: stateChange.Participant1PublicKey,
		}

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.Participant1,
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}

		//record there is an cooperative settlement ongoing
		channelState.SettleTransaction = &TransactionExecutionStatus{0, 0, ""}

		events = append(events, sendCooperativeSettle)
		events = append(events, sendProcessed)
	} else {
		failure := &EventInvalidReceivedCooperativeSettleRequest{
			TokenNetworkId: stateChange.TokenNetworkId,
			ChannelId:      stateChange.ChannelId,
			Reason:         err.Error(),
		}
		log.Warn("[handleCooperativeSettleRequestReceived] failure: ", err.Error())
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}

func isValidCooperativeSettleRequest(channelState *NettingChannelState, stateChange *ReceiveCooperativeSettleRequest) (bool, error) {
	if GetStatus(channelState) != ChannelStateOpened {
		return false, errors.New("channel is not opened")
	}

	ourState := channelState.GetChannelEndState(0)
	partnerState := channelState.GetChannelEndState(1)

	if !common.AddressEqual(partnerState.Address, stateChange.Participant1) {
		return false, errors.New("participant1 address invalid")
	} else if !common.AddressEqual(ourState.Address, stateChange.Participant2) {
		return false, errors.New("participant2 address invalid")
	} else if !common.AddressEqual(stateChange.Participant1, stateChange.Participant1Address) {
		return false, errors.New("participant address is same as the signer")
	}

	// calculate the final balance and compare with incoming balance values
	ourBalance := calculateCooperativeSettleBalance(ourState, partnerState)
	partnerBalance := calculateCooperativeSettleBalance(partnerState, ourState)

	if ourBalance != stateChange.Participant2Balance || partnerBalance != stateChange.Participant1Balance {
		return false, fmt.Errorf("invliad balance : ourBalance %d, partnerBalance %d, participant1 balance %d, participant2 balance %d",
			ourBalance, partnerBalance, stateChange.Participant1Balance, stateChange.Participant2Balance)
	}
	// verify if signature is valid
	dataToSign := PackCooperativeSettle(stateChange.ChannelId, stateChange.Participant1, stateChange.Participant1Balance,
		stateChange.Participant2, stateChange.Participant2Balance)
	return isValidSignature(dataToSign, stateChange.Participant1PublicKey, stateChange.Participant1Signature, stateChange.Participant1Address)
}

func handleCooperativeSettleReceived(channelState *NettingChannelState, stateChange *ReceiveCooperativeSettle) TransitionResult {
	var events []Event

	valid, err := isValidCooperativeSettle(channelState, stateChange)
	if valid {
		contractSendChannelCooperativeSettle := &ContractSendChannelCooperativeSettle{
			ChannelId:             stateChange.ChannelId,
			TokenNetworkId:        stateChange.TokenNetworkId,
			Participant1:          stateChange.Participant1,
			Participant1Balance:   stateChange.Participant1Balance,
			Participant2:          stateChange.Participant2,
			Participant2Balance:   stateChange.Participant2Balance,
			Participant1Signature: stateChange.Participant1Signature,
			Participant1Address:   stateChange.Participant1Address,
			Participant1PublicKey: stateChange.Participant1PublicKey,
			Participant2Signature: stateChange.Participant2Signature,
			Participant2Address:   stateChange.Participant2Address,
			Participant2PublicKey: stateChange.Participant2PublicKey,
		}

		events = append(events, contractSendChannelCooperativeSettle)
	} else {
		channelState.SettleTransaction = nil
		failure := &EventInvalidReceivedCooperativeSettle{
			TokenNetworkId: stateChange.TokenNetworkId,
			ChannelId:      stateChange.ChannelId,
			Reason:         err.Error(),
		}
		log.Warn("[handleCooperativeSettleReceived] failure: ", err.Error())
		events = append(events, failure)
	}
	return TransitionResult{channelState, events}
}
func isValidCooperativeSettle(channelState *NettingChannelState, stateChange *ReceiveCooperativeSettle) (bool, error) {
	if GetStatus(channelState) != ChannelStateSettling {

		return false, errors.New("channel is not opened")
	}

	ourState := channelState.GetChannelEndState(0)
	partnerState := channelState.GetChannelEndState(1)

	if !common.AddressEqual(ourState.Address, stateChange.Participant1) {
		return false, errors.New("participant1 address invalid")
	} else if !common.AddressEqual(partnerState.Address, stateChange.Participant2) {
		return false, errors.New("participant2 address invalid")
	} else if !common.AddressEqual(stateChange.Participant1, stateChange.Participant1Address) {
		return false, errors.New("participant1 address is same as the signer")
	} else if !common.AddressEqual(stateChange.Participant2, stateChange.Participant2Address) {
		return false, errors.New("participant1 address is same as the signer")
	}

	// calculate again the final balance and compare with incoming balance values
	ourBalance := calculateCooperativeSettleBalance(ourState, partnerState)
	partnerBalance := calculateCooperativeSettleBalance(partnerState, ourState)

	if ourBalance != stateChange.Participant1Balance || partnerBalance != stateChange.Participant2Balance {
		return false, fmt.Errorf("invliad balance : ourBalance %d, partnerBalance %d, participant1 balance %d, participant2 balance %d",
			ourBalance, partnerBalance, stateChange.Participant1Balance, stateChange.Participant2Balance)
	}

	dataToSign := PackCooperativeSettle(stateChange.ChannelId, stateChange.Participant1, stateChange.Participant1Balance,
		stateChange.Participant2, stateChange.Participant2Balance)
	ok, err := isValidSignature(dataToSign, stateChange.Participant1PublicKey, stateChange.Participant1Signature, stateChange.Participant1Address)
	if !ok {
		return false, fmt.Errorf("participant1 siganature invalid : %s", err.Error())
	}
	return isValidSignature(dataToSign, stateChange.Participant2PublicKey, stateChange.Participant2Signature, stateChange.Participant2Address)
}

func handleChannelCooperativeSettled(channelState *NettingChannelState, stateChange *ContractReceiveChannelCooperativeSettled) TransitionResult {
	var events []Event
	log.Debugf("[handleChannelCooperativeSettled]")

	if stateChange.ChannelId == channelState.Identifier {
		setSettled(channelState, stateChange.BlockHeight)
		// set to nil to delete the channel
		channelState = nil
	}

	return TransitionResult{channelState, events}
}

func handleActionClose(channelState *NettingChannelState, close *ActionChannelClose,
	blockNumber common.BlockHeight) TransitionResult {

	events := EventsForClose(channelState, blockNumber)
	return TransitionResult{channelState, events}
}

func handleRefundTransfer(receivedTransfer *LockedTransferUnsignedState, channelState *NettingChannelState,
	refund *ReceiveTransferRefund) ([]Event, error) {
	var events []Event
	isValid, merkleTree, err := IsValidRefund(refund, channelState, channelState.PartnerState,
		channelState.OurState, receivedTransfer)
	if isValid && err == nil {
		channelState.PartnerState.BalanceProof = refund.Transfer.BalanceProof
		channelState.PartnerState.MerkleTree = merkleTree

		lock := refund.Transfer.Lock
		channelState.PartnerState.SecretHashesToLockedLocks.Store(common.SecretHash(lock.SecretHash), lock)

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(refund.Transfer.BalanceProof.Sender),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: refund.Transfer.MessageId,
			},
		}
		events = append(events, sendProcessed)
	} else {
		invalidRefund := &EventInvalidReceivedTransferRefund{
			PaymentId: receivedTransfer.PaymentId,
			Reason:    err.Error(),
		}
		events = append(events, invalidRefund)
	}

	return events, err
}

func handleReceiveLockExpired(channelState *NettingChannelState, stateChange *ReceiveLockExpired,
	blockNumber common.BlockHeight) *TransitionResult {

	//"""Remove expired locks from channel states."""
	merkleTree, err := IsValidLockExpired(stateChange, channelState,
		channelState.PartnerState, channelState.OurState, blockNumber)

	var events []Event
	if err == nil {
		channelState.PartnerState.BalanceProof = stateChange.BalanceProof
		channelState.PartnerState.MerkleTree = merkleTree

		DelUnclaimedLock(channelState.PartnerState, stateChange.SecretHash)

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: stateChange.BalanceProof.Sender,
				ChannelId: ChannelIdGlobalQueue,
				MessageId: stateChange.MessageId,
			},
		}
		events = append(events, sendProcessed)
	} else {
		log.Debugf("handle invalid LockExpired : %s", err)
		invalidLockExpired := &EventInvalidReceivedLockExpired{
			SecretHash: stateChange.SecretHash,
			Reason:     err.Error(),
		}
		events = append(events, invalidLockExpired) //[invalid_lock_expired]
	}
	return &TransitionResult{NewState: channelState, Events: events}
}

func HandleReceiveLockedTransfer(channelState *NettingChannelState,
	mediatedTransfer *LockedTransferSignedState) ([]Event, error) {
	//Register the latest known transfer.
	//The receiver needs to use this method to update the container with a
	//_valid_ transfer, otherwise the locksroot will not contain the pending
	//transfer. The receiver needs to ensure that the merkle root has the
	//secrethash included, otherwise it won't be able to claim it.
	log.Debug("[HandleReceiveLockedTransfer] LocksRoot:", common.LocksRootHex(mediatedTransfer.BalanceProof.LocksRoot))

	merkleTree, err := IsValidLockedTransfer(mediatedTransfer, channelState, channelState.PartnerState, channelState.OurState)

	var events []Event
	if err == nil {
		channelState.PartnerState.BalanceProof = mediatedTransfer.BalanceProof
		channelState.PartnerState.MerkleTree = merkleTree

		lock := mediatedTransfer.Lock
		channelState.PartnerState.SecretHashesToLockedLocks.Store(common.SecretHash(lock.SecretHash), lock)

		sendProcessed := &SendProcessed{SendMessageEvent: SendMessageEvent{
			Recipient: mediatedTransfer.BalanceProof.Sender,
			ChannelId: ChannelIdGlobalQueue,
			MessageId: mediatedTransfer.MessageId,
		}}
		events = append(events, sendProcessed)
	} else {
		log.Errorf("[HandleReceiveLockedTransfer] IsValidLockedTransfer, error: %s", err.Error())
		invalidLocked := &EventInvalidReceivedLockedTransfer{
			PaymentId: mediatedTransfer.PaymentId,
			Reason:    err.Error(),
		}
		events = append(events, invalidLocked)
	}
	return events, err
}

func handleReceiveDirectTransfer(channelState *NettingChannelState,
	directTransfer *ReceiveTransferDirect) TransitionResult {

	var events []Event
	log.Debug("[handleReceiveDirectTransfer] %v \n", channelState.PartnerState)
	isValid, err := IsValidDirectTransfer(directTransfer, channelState,
		channelState.PartnerState, channelState.OurState)

	if isValid {
		currentBalanceProof, err := getCurrentBalanceProof(channelState.PartnerState)
		if err != nil {
			log.Error("[handleReceiveDirectTransfer] error: ", err.Error())
		}
		previousTransferredAmount := currentBalanceProof.transferredAmount

		newTransferredAmount := directTransfer.BalanceProof.TransferredAmount
		transferAmount := newTransferredAmount - previousTransferredAmount

		channelState.PartnerState.BalanceProof = directTransfer.BalanceProof

		paymentReceivedSuccess := &EventPaymentReceivedSuccess{
			PaymentNetworkId: channelState.PaymentNetworkId,
			TokenNetworkId:   channelState.TokenNetworkId,
			Identifier:       directTransfer.PaymentId,
			Amount:           transferAmount,
			Initiator:        channelState.PartnerState.Address,
		}

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: directTransfer.BalanceProof.Sender,
				ChannelId: ChannelIdGlobalQueue,
				MessageId: directTransfer.MessageId,
			},
		}
		events = append(events, paymentReceivedSuccess)
		events = append(events, sendProcessed)
	} else {
		transferInvalidEvent := &EventTransferReceivedInvalidDirectTransfer{
			Identifier: directTransfer.PaymentId,
			Reason:     err.Error(),
		}

		events = append(events, transferInvalidEvent)
		log.Error("[handleReceiveDirectTransfer] EventTransferReceivedInvalidDirectTransfer:", err.Error())
	}

	return TransitionResult{channelState, events}
}

func HandleUnlock(channelState *NettingChannelState, unlock *ReceiveUnlock) (bool, []Event, error) {
	isValid, unlockedMerkleTree, err := IsValidUnlock(unlock, channelState, channelState.PartnerState)
	var events []Event
	if isValid {
		channelState.PartnerState.BalanceProof = unlock.BalanceProof
		channelState.PartnerState.MerkleTree = unlockedMerkleTree

		secretHash := common.GetHash(unlock.Secret)
		DelLock(channelState.PartnerState, secretHash)

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: unlock.BalanceProof.Sender,
				ChannelId: ChannelIdGlobalQueue,
				MessageId: unlock.MessageId,
			},
		}
		events = append(events, sendProcessed)
	} else {
		log.Error("[HandleUnlock] ErrorMsg: ", err)
		secretHash := common.GetHash(unlock.Secret)
		invalidUnlock := &EventInvalidReceivedUnlock{
			SecretHash: secretHash,
			Reason:     err.Error(),
		}
		events = append(events, invalidUnlock)
	}
	return isValid, events, err
}

func handleBlock(channelState *NettingChannelState, stateChange *Block,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	if GetStatus(channelState) == ChannelStateClosed {
		closedBlockHeight := channelState.CloseTransaction.FinishedBlockHeight
		settlementEnd := closedBlockHeight + common.BlockHeight(channelState.SettleTimeout)

		if stateChange.BlockHeight > settlementEnd && channelState.SettleTransaction == nil {
			channelState.SettleTransaction = &TransactionExecutionStatus{
				StartedBlockHeight:  stateChange.BlockHeight,
				FinishedBlockHeight: 0,
				Result:              "",
			}

			event := &ContractSendChannelSettle{
				ContractSendEvent: ContractSendEvent{},
				ChannelId:         channelState.Identifier,
				TokenNetworkId:    common.TokenNetworkAddress(channelState.TokenNetworkId),
			}
			events = append(events, event)
		}
	}

	for isDepositConfirmed(channelState, blockNumber) {
		orderDepositTransaction := channelState.DepositTransactionQueue.Pop()
		log.Debug("[handleBlock] isDepositConfirmed applyChannelNewBalance")
		applyChannelNewBalance(channelState, &orderDepositTransaction.Transaction)
	}

	if GetStatus(channelState) == ChannelStateOpened {
		if withdraw := GetWithdrawTransaction(channelState); withdraw != nil {
			if withdraw.StartedBlockHeight+common.BlockHeight(common.Config.WithdrawTimeout) < blockNumber {
				event := &EventWithdrawRequestTimeout{
					ChannelId:      channelState.Identifier,
					TokenNetworkId: channelState.TokenNetworkId,
				}
				events = append(events, event)
			}
		}
	}

	return TransitionResult{channelState, events}
}

func handleChannelClosed(channelState *NettingChannelState, stateChange *ContractReceiveChannelClosed) TransitionResult {
	var events []Event

	justClosed := false

	status := GetStatus(channelState)
	if stateChange.ChannelId == channelState.Identifier &&
		(status == ChannelStateOpened || status == ChannelStateClosing) {
		justClosed = true
	}

	if justClosed {
		setClosed(channelState, stateChange.BlockHeight)

		balanceProof := channelState.PartnerState.BalanceProof
		callUpdate := false

		if common.AddressEqual(stateChange.TransactionFrom, channelState.OurState.Address) == false &&
			balanceProof != nil && channelState.UpdateTransaction == nil {
			callUpdate = true
		}

		if callUpdate {
			expiration := stateChange.BlockHeight + common.BlockHeight(channelState.SettleTimeout)
			update := &ContractSendChannelUpdateTransfer{
				ContractSendExpireAbleEvent: ContractSendExpireAbleEvent{
					ContractSendEvent: ContractSendEvent{},
					Expiration:        common.BlockExpiration(expiration),
				},
				ChannelId:      channelState.Identifier,
				TokenNetworkId: channelState.TokenNetworkId,
				BalanceProof:   balanceProof,
			}

			channelState.UpdateTransaction = &TransactionExecutionStatus{stateChange.BlockHeight,
				0, ""}

			events = append(events, update)
		}
	}

	return TransitionResult{channelState, events}
}

func handleChannelUpdatedTransfer(channelState *NettingChannelState,
	stateChange *ContractReceiveUpdateTransfer, blockNumber common.BlockHeight) TransitionResult {

	if stateChange.ChannelId == channelState.Identifier {
		channelState.UpdateTransaction = &TransactionExecutionStatus{
			0, 0, "success"}
	}

	return TransitionResult{channelState, nil}
}

func handleChannelSettled(channelState *NettingChannelState,
	stateChange *ContractReceiveChannelSettled, blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	if stateChange.ChannelId == channelState.Identifier {
		setSettled(channelState, stateChange.BlockHeight)
		isSettlePending := false
		if channelState.OurUnlockTransaction != nil {
			isSettlePending = true
		}

		merkleTreeLeaves := GetBatchUnlock(channelState.PartnerState)

		if isSettlePending == false && merkleTreeLeaves != nil && len(merkleTreeLeaves) != 0 {
			onChainUnlock := &ContractSendChannelBatchUnlock{
				ContractSendEvent: ContractSendEvent{},
				TokenAddress:      common.TokenAddress(channelState.TokenAddress),
				TokenNetworkId:    channelState.TokenNetworkId,
				ChannelId:         channelState.Identifier,
				Participant:       channelState.PartnerState.Address,
			}

			events = append(events, onChainUnlock)

			channelState.OurUnlockTransaction = &TransactionExecutionStatus{
				blockNumber, 0, ""}
		} else {
			channelState = nil
		}

	}

	return TransitionResult{channelState, events}
}

func handleChannelBatchUnlock(channelState *NettingChannelState, stateChange *ContractReceiveChannelBatchUnlock) TransitionResult {
	var events []Event

	// unlock is allowed by the smart contract only on a settled channel.
	// ignore the unlock if the channel is not closed yet
	if GetStatus(channelState) == ChannelStateSettled {
		channelState = nil
	}

	return TransitionResult{channelState, events}
}

func handleChannelNewBalance(channelState *NettingChannelState,
	stateChange *ContractReceiveChannelNewBalance,
	blockNumber common.BlockHeight) TransitionResult {

	depositTransaction := stateChange.DepositTransaction

	if IsTransactionConfirmed(depositTransaction.DepositBlockHeight, blockNumber) {
		log.Debug("[handleChannelNewBalance] IsTransactionConfirmed applyChannelNewBalance")
		applyChannelNewBalance(channelState, &stateChange.DepositTransaction)
	} else {
		order := TransactionOrder{depositTransaction.DepositBlockHeight, depositTransaction}
		log.Debug("[handleChannelNewBalance] DepositTransactionQueue.Push")
		channelState.DepositTransactionQueue.Push(order)
	}

	UpdateFeeScheduleAfterBalanceChange(channelState, stateChange.FeeConfig)
	return TransitionResult{channelState, nil}
}

func UpdateFeeScheduleAfterBalanceChange(channelState *NettingChannelState, feeConfig common.MediationFeeConfig) {
	flat := feeConfig.TokenToFlatFee[common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)]
	pro := feeConfig.TokenToProportionalFee[common.TokenAddress(usdt.USDT_CONTRACT_ADDRESS)]
	channelState.SetFeeSchedule(flat, pro)
}

func applyChannelNewBalance(channelState *NettingChannelState,
	depositTransaction *TransactionChannelNewBalance) {
	participantAddress := depositTransaction.ParticipantAddress
	contractBalance := depositTransaction.ContractBalance

	if common.AddressEqual(participantAddress, channelState.OurState.Address) {
		updateContractBalance(channelState.OurState, common.Balance(contractBalance))
	} else if common.AddressEqual(participantAddress, channelState.PartnerState.Address) {
		updateContractBalance(channelState.PartnerState, common.Balance(contractBalance))
	}

	return
}

func StateTransitionForChannel(channelState *NettingChannelState, stateChange StateChange,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event
	iteration := TransitionResult{channelState, events}

	switch stateChange.(type) {
	case *Block:
		block, _ := stateChange.(*Block)
		iteration = handleBlock(channelState, block, blockNumber)
	case *ActionChannelClose:
		actionChannelClose, _ := stateChange.(*ActionChannelClose)
		iteration = handleActionClose(channelState, actionChannelClose, blockNumber)
	case *ActionTransferDirect:
		actionTransferDirect, _ := stateChange.(*ActionTransferDirect)
		iteration = handleSendDirectTransfer(channelState, actionTransferDirect)
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		iteration = handleChannelClosed(channelState, contractReceiveChannelClosed)
	case *ContractReceiveUpdateTransfer:
		contractReceiveUpdateTransfer, _ := stateChange.(*ContractReceiveUpdateTransfer)
		iteration = handleChannelUpdatedTransfer(channelState, contractReceiveUpdateTransfer, blockNumber)
	case *ContractReceiveChannelSettled:
		contractReceiveChannelSettled, _ := stateChange.(*ContractReceiveChannelSettled)
		iteration = handleChannelSettled(channelState, contractReceiveChannelSettled, blockNumber)
	case *ContractReceiveChannelNewBalance:
		contractReceiveChannelNewBalance, _ := stateChange.(*ContractReceiveChannelNewBalance)
		iteration = handleChannelNewBalance(channelState, contractReceiveChannelNewBalance, blockNumber)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleReceiveDirectTransfer(channelState, receiveTransferDirect)
	case *ActionWithdraw:
		actionWithdraw, _ := stateChange.(*ActionWithdraw)
		iteration = handleSendWithdrawRequest(channelState, actionWithdraw, blockNumber)
	case *ReceiveWithdrawRequest:
		receiveWithdrawRequest, _ := stateChange.(*ReceiveWithdrawRequest)
		iteration = handleWithdrawRequestReceived(channelState, receiveWithdrawRequest)
	case *ReceiveWithdraw:
		receiveWithdraw, _ := stateChange.(*ReceiveWithdraw)
		iteration = handleWithdrawReceived(channelState, receiveWithdraw)
	case *ContractReceiveChannelWithdraw:
		contractReceiveChannelWithdraw, _ := stateChange.(*ContractReceiveChannelWithdraw)
		iteration = handleChannelWithdraw(channelState, contractReceiveChannelWithdraw)
	case *ActionCooperativeSettle:
		actionCooperativeSettle, _ := stateChange.(*ActionCooperativeSettle)
		iteration = handleSendCooperativeSettleRequest(channelState, actionCooperativeSettle)
	case *ReceiveCooperativeSettleRequest:
		receiveCooperativeSettleRequest, _ := stateChange.(*ReceiveCooperativeSettleRequest)
		iteration = handleCooperativeSettleRequestReceived(channelState, receiveCooperativeSettleRequest)
	case *ReceiveCooperativeSettle:
		receiveCooperativeSettle, _ := stateChange.(*ReceiveCooperativeSettle)
		iteration = handleCooperativeSettleReceived(channelState, receiveCooperativeSettle)
	case *ContractReceiveChannelCooperativeSettled:
		contractReceiveChannelCooperativeSettled, _ := stateChange.(*ContractReceiveChannelCooperativeSettled)
		iteration = handleChannelCooperativeSettled(channelState, contractReceiveChannelCooperativeSettled)
	case *ContractReceiveChannelBatchUnlock:
		contractReceiveChannelBatchUnlock, _ := stateChange.(*ContractReceiveChannelBatchUnlock)
		iteration = handleChannelBatchUnlock(channelState, contractReceiveChannelBatchUnlock)
	}

	return iteration
}
