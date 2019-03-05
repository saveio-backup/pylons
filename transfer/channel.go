package transfer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
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
	locksRoot         common.Locksroot
	nonce             common.Nonce
	transferredAmount common.TokenAmount
	lockedAmount      common.TokenAmount
}

func IsLockPending(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToLockedLocks[secretHash]; exist {
		return true
	} else if _, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
		return true
	} else if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
		return true
	}

	return false
}

func IsLockLocked(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToLockedLocks[secretHash]; exist {
		return true
	}
	return false
}

func IsLockExpired(endState *NettingChannelEndState, lock *HashTimeLockState,
	blockNumber common.BlockHeight, lockExpirationThreshold common.BlockHeight) (bool, string) {

	secretHash := common.SecretHash(lock.SecretHash)
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
		return false, "lock has been unlocked on-chain"
	}

	if blockNumber < lockExpirationThreshold {
		return false, "current block number is not larger than lock expiration + confirmation blocks"
	}

	return true, ""
}

func TransferExpired(transfer *LockedTransferSignedState, affectedChannel *NettingChannelState,
	blockNumber common.BlockHeight) bool {
	lockExpirationThreshold := transfer.Lock.Expiration + common.BlockHeight(DefaultNumberOfConfirmationsBlock*2)
	hasLockExpired, _ := IsLockExpired(affectedChannel.OurState, transfer.Lock, blockNumber, lockExpirationThreshold)
	return hasLockExpired
}

func IsSecretKnown(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
		return true
	} else if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
		return true
	}

	return false
}

func IsSecretKnownOffChain(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
		return true
	}
	return false
}

func IsSecretKnownOnChain(endState *NettingChannelEndState, secretHash common.SecretHash) bool {
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
		return true
	}
	return false
}

//[NOTE] may use *common.Secret as return value?
func GetSecret(endState *NettingChannelEndState, secretHash common.SecretHash) common.Secret {
	if IsSecretKnown(endState, secretHash) == true {
		var partialUnlockProof *UnlockPartialProofState

		if _, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
			partialUnlockProof = endState.SecretHashesToUnLockedLocks[secretHash]
		} else if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
			partialUnlockProof = endState.SecretHashesToOnChainUnLockedLocks[secretHash]
		}
		return partialUnlockProof.Secret
	}
	return common.Secret{}
}

func GetLock(endState *NettingChannelEndState, secretHash common.SecretHash) *HashTimeLockState {
	lock, exist := endState.SecretHashesToLockedLocks[secretHash]
	if !exist {
		partialUnlock, exist := endState.SecretHashesToUnLockedLocks[secretHash]
		if !exist {
			partialUnlock = endState.SecretHashesToOnChainUnLockedLocks[secretHash]
		}
		if exist {
			lock = partialUnlock.Lock
		}
	}
	return lock
}

func LockExistsInEitherChannelSide(channelState *NettingChannelState, secretHash common.SecretHash) bool {
	//"""Check if the lock with `secrethash` exists in either our state or the partner's state"""
	lock := GetLock(channelState.OurState, secretHash)
	if lock != nil {
		lock = GetLock(channelState.PartnerState, secretHash)
	}
	return lock != nil
}

func DelUnclaimedLock(endState *NettingChannelEndState, secretHash common.SecretHash) {
	if _, exist := endState.SecretHashesToLockedLocks[secretHash]; exist {
		delete(endState.SecretHashesToLockedLocks, secretHash)
	}

	if _, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
		delete(endState.SecretHashesToUnLockedLocks, secretHash)
	}
	return
}

func DelLock(endState *NettingChannelEndState, secretHash common.SecretHash) {
	if IsLockPending(endState, secretHash) == false {
		log.Debug("[DelLock] IsLockPending == false")
		return
	}

	DelUnclaimedLock(endState, secretHash)
	if _, exist := endState.SecretHashesToOnChainUnLockedLocks[secretHash]; exist {
		delete(endState.SecretHashesToOnChainUnLockedLocks, secretHash)
	}

	return
}

func RegisterSecretEndState(endState *NettingChannelEndState, secret common.Secret, secretHash common.SecretHash) {
	if IsLockLocked(endState, secretHash) == true {
		pendingLock := endState.SecretHashesToLockedLocks[secretHash]
		delete(endState.SecretHashesToLockedLocks, secretHash)

		endState.SecretHashesToUnLockedLocks[secretHash] = &UnlockPartialProofState{
			Lock: pendingLock, Secret: secret}
	}
	return
}

func RegisterOnChainSecretEndState(endState *NettingChannelEndState, secret common.Secret,
	secretHash common.SecretHash, secretRevealBlockNumber common.BlockHeight, deleteLock bool) {

	var pendingLock *HashTimeLockState

	if IsLockLocked(endState, secretHash) == true {
		pendingLock = endState.SecretHashesToLockedLocks[secretHash]
	}

	if v, exist := endState.SecretHashesToUnLockedLocks[secretHash]; exist {
		pendingLock = v.Lock
	}

	//if pendingLock != nil {
	if pendingLock.Expiration < secretRevealBlockNumber {
		return
	}

	if deleteLock == true {
		DelLock(endState, secretHash)
	}

	endState.SecretHashesToOnChainUnLockedLocks[secretHash] = &UnlockPartialProofState{
		Lock:   pendingLock,
		Secret: secret,
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

func compareLocksroot(one common.Locksroot, two common.Locksroot) bool {
	result := true

	if len(one) != len(two) {
		result = false
	}

	length := len(one)
	for i := 0; i < length; i++ {
		if one[i] != two[i] {
			result = false
			break
		}
	}

	return result
}

func isDepositConfirmed(channelState *NettingChannelState, blockNumber common.BlockHeight) bool {
	if len(channelState.DepositTransactionQueue) == 0 {
		return false
	}

	result := IsTransactionConfirmed(channelState.DepositTransactionQueue[0].BlockHeight, blockNumber)
	return result
}

func IsTransactionConfirmed(transactionBlockHeight common.BlockHeight, blockchainBlockHeight common.BlockHeight) bool {
	confirmationBlock := transactionBlockHeight + (common.BlockHeight)(constants.COUNT_OF_CONFIRM_BLOCK)
	return blockchainBlockHeight > confirmationBlock
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
	senderState *NettingChannelEndState) (bool, string, *MerkleTreeState) {
	receivedBalanceProof := unlock.BalanceProof
	currentBalanceProof := getCurrentBalanceProof(senderState)

	secretHash := common.GetHash(unlock.Secret)
	log.Debug("[IsValidUnlock] secretHash: ", secretHash)
	lock := GetLock(senderState, secretHash)

	if lock == nil {
		msg := fmt.Sprintf("Invalid Unlock message. There is no corresponding lock for %s",
			hex.EncodeToString(secretHash[:]))

		return false, msg, nil
	}

	merkleTree := computeMerkleTreeWithout(senderState.MerkleTree, lock.LockHash)
	locksrootWithoutLock := MerkleRoot(merkleTree.Layers)

	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount
	expectedTransferredAmount := currentTransferredAmount + lock.Amount
	expectedLockedAmount := currentLockedAmount - lock.Amount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(
		receivedBalanceProof, channelState, senderState)

	if !isBalanceProofUsable {
		msg := fmt.Sprintf("Invalid Unlock message. %s", invalidBalanceProofMsg)
		return false, msg, nil
	} else if receivedBalanceProof.LocksRoot != common.Locksroot(locksrootWithoutLock) {
		//# Secret messages remove a known lock, the new locksroot must have only
		//# that lock removed, otherwise the sender may be trying to remove
		//# additional locks.
		msg := fmt.Sprintf("Invalid Unlock message. Balance proof's locksroot didn't match, expected: %s got: %s.",
			hex.EncodeToString(locksrootWithoutLock[:]), hex.EncodeToString(receivedBalanceProof.LocksRoot[:]))
		return false, msg, nil
	} else if receivedBalanceProof.TransferredAmount != expectedTransferredAmount {
		//# Secret messages must increase the transferred_amount by lock amount,
		//# otherwise the sender is trying to play the protocol and steal token.
		msg := fmt.Sprintf("Invalid Unlock message. Balance proof's wrong transferred_amount, expected: %s got: %s.",
			expectedTransferredAmount, receivedBalanceProof.TransferredAmount)

		return false, msg, nil
	} else if receivedBalanceProof.LockedAmount != expectedLockedAmount {
		//# Secret messages must increase the transferred_amount by lock amount,
		//# otherwise the sender is trying to play the protocol and steal token.
		msg := fmt.Sprintf("Invalid Unlock message. Balance proof's wrong locked_amount, expected: %s got: %s.",
			expectedLockedAmount, receivedBalanceProof.LockedAmount,
		)
		return false, msg, nil
	} else {
		return true, "", merkleTree
	}
}

func IsValidAmount(endState *NettingChannelEndState, amount common.TokenAmount) bool {
	balanceProofData := getCurrentBalanceProof(endState)
	currentTransferredAmount := balanceProofData.transferredAmount
	currentLockedAmount := balanceProofData.lockedAmount

	transferredAmountAfterUnlock := currentTransferredAmount + currentLockedAmount + amount

	return transferredAmountAfterUnlock <= math.MaxUint64
}

func IsValidSignature(balanceProof *BalanceProofSignedState, senderAddress common.Address) (bool, string) {

	//[TODO] find similar ONT function for
	// to_canonical_address(eth_recover(data=data_that_was_signed, signature=balance_proof.signature))
	var signerAddress common.Address

	balanceHash := HashBalanceData(balanceProof.TransferredAmount,
		balanceProof.LockedAmount, balanceProof.LocksRoot)

	dataThatWasSigned := PackBalanceProof(common.Nonce(balanceProof.Nonce), balanceHash, common.AdditionalHash(balanceProof.MessageHash[:]),
		balanceProof.ChannelIdentifier, common.TokenNetworkAddress(balanceProof.TokenNetworkIdentifier), balanceProof.ChainId, 1)

	if dataThatWasSigned == nil {
		return false, string("Signature invalid, could not be recovered dataThatWasSigned is nil")
	}

	pubKey, err := common.GetPublicKey(balanceProof.PublicKey)
	if err != nil {
		return false, string("Failed to get public key from balance proof")
	}

	//log.Debug("Verify [Data]: ", dataThatWasSigned)
	//log.Debug("Verify [PubKey]: ", balanceProof.PublicKey)
	//log.Debug("Verify [Signature]: ", balanceProof.Signature)
	err = common.VerifySignature(pubKey, dataThatWasSigned, balanceProof.Signature)
	if err != nil {
		return false, string("Signature invalid, could not be recovered")
	}

	signerAddress = common.GetAddressFromPubKey(pubKey)

	if common.AddressEqual(senderAddress, signerAddress) == true {
		return true, "success"
	} else {
		return false, string("Signature was valid but the expected address does not match.")
	}

}

func IsBalanceProofUsableOnChain(receivedBalanceProof *BalanceProofSignedState,
	channelState *NettingChannelState, senderState *NettingChannelEndState) (bool, string) {

	expectedNonce := getNextNonce(senderState)

	isValidSignature, signatureMsg := IsValidSignature(receivedBalanceProof,
		senderState.Address)
	if !isValidSignature {
		return isValidSignature, signatureMsg
	}
	if GetStatus(channelState) != ChannelStateOpened {
		msg := fmt.Sprintf("The channel is already closed.")
		return false, msg

	} else if receivedBalanceProof.ChannelIdentifier != channelState.Identifier {
		msg := fmt.Sprintf("channel_identifier does not match. expected:%d got:%d",
			channelState.Identifier, receivedBalanceProof.ChannelIdentifier)
		return false, msg
	} else if common.AddressEqual(common.Address(receivedBalanceProof.TokenNetworkIdentifier), common.Address(channelState.TokenNetworkIdentifier)) == false {
		msg := fmt.Sprintf("token_network_identifier does not match. expected:%d got:%d",
			channelState.TokenNetworkIdentifier, receivedBalanceProof.TokenNetworkIdentifier)
		return false, msg
	} else if receivedBalanceProof.ChainId != channelState.ChainId {
		msg := fmt.Sprintf("chain id does not match channel's chain id. expected:%d got:%d",
			channelState.ChainId, receivedBalanceProof.ChainId)
		return false, msg
	} else if IsBalanceProofSafeForOnchainOperations(receivedBalanceProof) == false {

		msg := fmt.Sprintf("Balance proof total transferred amount would overflow onchain.")
		return false, msg
	} else if receivedBalanceProof.Nonce != expectedNonce {
		msg := fmt.Sprintf("Nonce did not change sequentially, expected:%d got:%d",
			expectedNonce, receivedBalanceProof.Nonce)

		return false, msg
	} else if isValidSignature == false {
		return false, signatureMsg
	} else {
		return true, "success"
	}
}

func IsValidDirectTransfer(directTransfer *ReceiveTransferDirect, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState) (bool, string) {

	receivedBalanceProof := directTransfer.BalanceProof

	currentBalanceProof := getCurrentBalanceProof(senderState)
	currentLocksRoot := currentBalanceProof.locksRoot
	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	distributable := GetDistributable(senderState, receiverState)
	amount := receivedBalanceProof.TransferredAmount - currentTransferredAmount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(
		receivedBalanceProof, channelState, senderState)

	if isBalanceProofUsable == false {
		msg := fmt.Sprintf("Invalid DirectTransfer message. {%s}", invalidBalanceProofMsg)
		return false, msg
	} else if compareLocksroot(receivedBalanceProof.LocksRoot, currentLocksRoot) == false {
		var buf1, buf2 bytes.Buffer

		buf1.Write(currentBalanceProof.locksRoot[:])
		buf2.Write(receivedBalanceProof.LocksRoot[:])
		msg := fmt.Sprintf("Invalid DirectTransfer message. Balance proof's locksRoot changed, expected:%s got: %s",
			buf1.String(), buf2.String())

		return false, msg
	} else if receivedBalanceProof.TransferredAmount <= currentTransferredAmount {
		msg := fmt.Sprintf("Invalid DirectTransfer message. Balance proof's transferred_amount decreased, expected larger than: %d got: %d",
			currentTransferredAmount, receivedBalanceProof.TransferredAmount)

		return false, msg
	} else if receivedBalanceProof.LockedAmount != currentLockedAmount {
		msg := fmt.Sprintf("Invalid DirectTransfer message. Balance proof's locked_amount is invalid, expected: %d got: %d",
			currentLockedAmount, receivedBalanceProof.LockedAmount)

		return false, msg
	} else if amount > distributable {
		msg := fmt.Sprintf("Invalid DirectTransfer message. Transfer amount larger than the available distributable, transfer amount: %d maximum distributable: %d",
			amount, distributable)
		return false, msg
	} else {
		return true, "success"
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
	lock := channelState.PartnerState.SecretHashesToLockedLocks[secretHash]

	//# If the lock was not found in locked locks, this means that we've received
	//# the secret for the locked transfer but we haven't unlocked it yet. Lock
	//# expiry in this case could still happen which means that we have to make
	//# sure that we check for "unclaimed" locks in our check.
	if lock != nil {
		lock = channelState.PartnerState.SecretHashesToUnLockedLocks[secretHash].Lock
	}

	var lockRegisteredOnChain []common.SecretHash
	for secretHash := range channelState.OurState.SecretHashesToOnChainUnLockedLocks {
		lockRegisteredOnChain = append(lockRegisteredOnChain, secretHash)
	}

	if lock != nil {
		return nil, fmt.Errorf("Invalid LockExpired message. Lock with secrethash %s is not known. ",
			hex.EncodeToString(secretHash[:]))
	}

	currentBalanceProof := getCurrentBalanceProof(senderState)
	merkletree := computeMerkleTreeWithout(senderState.MerkleTree, lock.LockHash)

	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount
	expectedLockedAmount := currentLockedAmount - lock.Amount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnChain(
		receivedBalanceProof, channelState, senderState)

	//result: MerkleTreeOrError = (False, None, None)

	if !isBalanceProofUsable {
		return nil, fmt.Errorf("Invalid LockExpired message. %s ", invalidBalanceProofMsg)
	} else if merkletree == nil {
		return nil, fmt.Errorf("Invalid LockExpired message. Same lockhash handled twice. ")
	} else if lockRegisteredOnChain != nil {
		return nil, fmt.Errorf("Invalid LockExpired mesage. Lock was unlocked on-chain. ")
	} else {
		locksRootWithoutLock := MerkleRoot(merkletree.Layers)
		hasExpired, lockExpiredMessage := IsLockExpired(receiverState, lock, blockNumber,
			lock.Expiration+common.BlockHeight(DefaultNumberOfConfirmationsBlock))
		if !hasExpired {
			return nil, fmt.Errorf("Invalid LockExpired message. %s ", lockExpiredMessage)
		} else if receivedBalanceProof.LocksRoot != common.Locksroot(locksRootWithoutLock) {
			//The locksRoot must be updated, and the expired lock must be *removed*
			return nil, fmt.Errorf("Invalid LockExpired message. "+
				"Balance proof's locksroot didn't match, expected: %s got: %s. ",
				hex.EncodeToString(locksRootWithoutLock[:]),
				hex.EncodeToString(receivedBalanceProof.LocksRoot[:]))
		} else if receivedBalanceProof.TransferredAmount != currentTransferredAmount {
			//# Given an expired lock, transferred amount should stay the same
			return nil, fmt.Errorf("Invalid LockExpired message. "+
				"Balance proof's transferred_amount changed, expected: %d got: %d. ",
				currentTransferredAmount, receivedBalanceProof.TransferredAmount)
		} else if receivedBalanceProof.LockedAmount != expectedLockedAmount {
			//# locked amount should be the same found inside the balance proof
			return nil, fmt.Errorf("Invalid LockExpired message. "+
				"Balance proof's locked_amount is invalid, expected: %d got: %d. ",
				expectedLockedAmount, receivedBalanceProof.LockedAmount)
		} else {
			return merkletree, nil
		}
	}
	return nil, nil
}

func ValidLockedTransferCheck(channelState *NettingChannelState, senderState *NettingChannelEndState,
	receiverState *NettingChannelEndState, messageName string, receivedBalanceProof *BalanceProofSignedState,
	lock *HashTimeLockState) (*MerkleTreeState, error) {

	currentBalanceProof := getCurrentBalanceProof(senderState)
	lockHash := lock.CalcLockHash()
	merkleTree := computeMerkleTreeWith(senderState.MerkleTree, lockHash)

	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount
	//_, _, current_transferred_amount, current_locked_amount = current_balance_proof
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

		if receivedBalanceProof.LocksRoot != common.Locksroot(locksRootWithLock) {
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

	for _, lock := range endState.SecretHashesToLockedLocks {
		totalPending = totalPending + lock.Amount
	}
	log.Debug("[getAmountLocked] totalPending: ", totalPending)
	for _, unLock := range endState.SecretHashesToUnLockedLocks {
		totalUnclaimed = totalPending + unLock.Lock.Amount
	}
	log.Debug("[getAmountLocked] totalUnclaimed: ", totalUnclaimed)
	totalUnclaimedOnChain = getAmountUnClaimedOnChain(endState)
	log.Debug("[getAmountLocked] totalUnclaimedOnChain: ", totalUnclaimedOnChain)
	lockedAmount := (common.Balance)(totalPending + totalUnclaimed + totalUnclaimedOnChain)
	log.Debug("[getAmountLocked] lockedAmount: ", lockedAmount)

	return lockedAmount
}

func getAmountUnClaimedOnChain(endState *NettingChannelEndState) common.TokenAmount {
	var totalUnclaimedOnChain common.TokenAmount
	for _, unLock := range endState.SecretHashesToOnChainUnLockedLocks {
		totalUnclaimedOnChain = totalUnclaimedOnChain + unLock.Lock.Amount
	}
	return totalUnclaimedOnChain
}

func getBalance(sender *NettingChannelEndState, receiver *NettingChannelEndState) common.Balance {

	var senderTransferredAmount, receiverTransferredAmount common.TokenAmount

	if sender.BalanceProof != nil {
		senderTransferredAmount = sender.BalanceProof.TransferredAmount
	}

	if receiver.BalanceProof != nil {
		receiverTransferredAmount = receiver.BalanceProof.TransferredAmount
	}

	result := sender.ContractBalance - senderTransferredAmount + receiverTransferredAmount
	return (common.Balance)(result)

}

func getCurrentBalanceProof(endState *NettingChannelEndState) BalanceProofData {
	var locksRoot common.Locksroot
	var nonce common.Nonce
	var transferredAmount, lockedAmount common.TokenAmount

	balanceProof := endState.BalanceProof

	if balanceProof != nil {
		locksRoot = balanceProof.LocksRoot
		nonce = (common.Nonce)(balanceProof.Nonce)
		transferredAmount = balanceProof.TransferredAmount
		lockedAmount = (common.TokenAmount)(getAmountLocked(endState))
	} else {
		locksRoot = common.Locksroot{}
		nonce = 0
		transferredAmount = 0
		lockedAmount = 0
	}

	return BalanceProofData{locksRoot, nonce, transferredAmount, lockedAmount}
}

func GetDistributable(sender *NettingChannelEndState, receiver *NettingChannelEndState) common.TokenAmount {
	balanceProofData := getCurrentBalanceProof(sender)

	transferredAmount := balanceProofData.transferredAmount
	lockedAmount := balanceProofData.lockedAmount

	distributable := getBalance(sender, receiver) - getAmountLocked(sender)

	overflowLimit := Max(
		math.MaxUint64-(uint64)(transferredAmount)-(uint64)(lockedAmount),
		0,
	)

	result := Min(overflowLimit, (uint64)(distributable))
	return (common.TokenAmount)(result)
}

func getNextNonce(endState *NettingChannelEndState) common.Nonce {
	if endState.BalanceProof != nil {
		return endState.BalanceProof.Nonce + 1
	} else {
		log.Warn("[getNextNonce=] endState.BalanceProof == nil")
	}

	return 1
}

func MerkleTreeWidth(merkleTree *MerkleTreeState) int {
	return len(merkleTree.Layers[0])
}

func getNumberOfPendingTransfers(channelEndState *NettingChannelEndState) int {
	if channelEndState.MerkleTree == nil {
		log.Debug("[getNumberOfPendingTransfers] channelEndState.MerkleTree == nil")
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

func computeMerkleTreeWith(merkletree *MerkleTreeState, lockHash common.LockHash) *MerkleTreeState {
	var result *MerkleTreeState
	log.Debug("[computeMerkleTreeWith] lockHash:", lockHash)
	leaves := merkletree.Layers[0]
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
		newLeaves = append(newLeaves, common.Keccak256(lockHash))

		newLayers := computeLayers(newLeaves)
		result = &MerkleTreeState{}
		result.Layers = newLayers
	} else {
		log.Error("[computeMerkleTreeWith] lockHash is found")
	}

	return result
}

func computeMerkleTreeWithout(merkletree *MerkleTreeState, lockhash common.LockHash) *MerkleTreeState {
	var i int
	found := false
	leaves := merkletree.Layers[0]
	log.Debug("[computeMerkleTreeWithout] lockhash:", lockhash)
	for i = 0; i < len(leaves); i++ {
		temp := common.Keccak256(lockhash)
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
			return nil
		}
	}
	return &result
}

func createSendLockedTransfer(channelState *NettingChannelState, initiator common.InitiatorAddress,
	target common.Address, amount common.PaymentAmount, messageIdentifier common.MessageID,
	paymentIdentifier common.PaymentID, expiration common.BlockExpiration,
	secretHash common.SecretHash) (*SendLockedTransfer, *MerkleTreeState) {

	ourState := channelState.OurState
	partnerState := channelState.PartnerState
	ourBalanceProof := channelState.OurState.BalanceProof

	if common.TokenAmount(amount) > GetDistributable(ourState, partnerState) {
		return nil, nil
	}

	if GetStatus(channelState) != ChannelStateOpened {
		return nil, nil
	}

	lock := &HashTimeLockState{
		Amount:     common.TokenAmount(amount),
		Expiration: common.BlockHeight(expiration),
		SecretHash: common.Keccak256(secretHash),
	}
	lockHash := lock.CalcLockHash()
	log.Debug("[createSendLockedTransfer] lock.Lockhash", lockHash)
	merkleTree := computeMerkleTreeWith(channelState.OurState.MerkleTree, lockHash)
	//depth := len(merkleTree.Layers)
	//locksRoot := merkleTree.Layers[depth - 1][0]
	locksRoot := MerkleRoot(merkleTree.Layers)
	var transferAmount common.TokenAmount
	if ourBalanceProof != nil {
		transferAmount = ourBalanceProof.TransferredAmount
	} else {
		transferAmount = 0
	}

	token := channelState.TokenAddress
	nonce := getNextNonce(ourState)
	log.Debug("[createSendLockedTransfer] nonce: ", nonce)

	recipient := partnerState.Address
	lockedAmount := getAmountLocked(ourState) + common.Balance(amount)

	balanceProof := &BalanceProofUnsignedState{
		Nonce:                  nonce,
		TransferredAmount:      transferAmount,
		LockedAmount:           common.TokenAmount(lockedAmount),
		LocksRoot:              common.Locksroot(locksRoot),
		TokenNetworkIdentifier: channelState.TokenNetworkIdentifier,
		ChannelIdentifier:      channelState.Identifier,
		ChainId:                channelState.ChainId,
	}

	transfer := &LockedTransferUnsignedState{
		PaymentIdentifier: paymentIdentifier,
		Token:             token,
		BalanceProof:      balanceProof,
		Lock:              lock,
		Initiator:         common.Address(initiator),
		Target:            common.Address(target),
	}

	lockedTransfer := &SendLockedTransfer{
		SendMessageEvent: SendMessageEvent{
			Recipient:         common.Address(recipient),
			ChannelIdentifier: channelState.Identifier,
			MessageIdentifier: messageIdentifier,
		},
		Transfer: transfer,
	}

	return lockedTransfer, merkleTree
}

func createSendDirectTransfer(channelState *NettingChannelState, amount common.PaymentAmount,
	messageIdentifier common.MessageID, paymentIdentifier common.PaymentID) *SendDirectTransfer {

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
	var locksRoot common.Locksroot

	if ourBalanceProof != nil {
		transferAmount = common.TokenAmount(amount) + ourBalanceProof.TransferredAmount
		locksRoot = ourBalanceProof.LocksRoot
	} else {
		transferAmount = common.TokenAmount(amount)
		locksRoot = common.Locksroot{}
	}

	nonce := getNextNonce(ourState)
	recipient := partnerState.Address
	lockedAmount := getAmountLocked(ourState)

	balanceProof := &BalanceProofUnsignedState{
		Nonce:nonce,
		TransferredAmount:transferAmount,
		LockedAmount:common.TokenAmount(lockedAmount),
		LocksRoot:locksRoot,
		TokenNetworkIdentifier:channelState.TokenNetworkIdentifier,
		ChannelIdentifier:channelState.Identifier,
		ChainId:channelState.ChainId,
	}

	sendDirectTransfer := SendDirectTransfer{
		SendMessageEvent:SendMessageEvent{
			Recipient:common.Address(recipient),
			ChannelIdentifier:channelState.Identifier,
			MessageIdentifier:messageIdentifier,
		},
		PaymentIdentifier:paymentIdentifier,
		BalanceProof:balanceProof,
		TokenAddress:common.TokenAddress(channelState.TokenAddress),
	}

	return &sendDirectTransfer
}

func sendDirectTransfer(channelState *NettingChannelState, amount common.PaymentAmount,
	messageIdentifier common.MessageID, paymentIdentifier common.PaymentID) *SendDirectTransfer {

	directTransfer := createSendDirectTransfer(channelState, amount, messageIdentifier, paymentIdentifier)

	//Construct fake BalanceProofSignedState from BalanceProofUnsignedState!
	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:                  directTransfer.BalanceProof.Nonce,
		TransferredAmount:      directTransfer.BalanceProof.TransferredAmount,
		LockedAmount:           directTransfer.BalanceProof.LockedAmount,
		LocksRoot:              directTransfer.BalanceProof.LocksRoot,
		TokenNetworkIdentifier: directTransfer.BalanceProof.TokenNetworkIdentifier,
		ChannelIdentifier:      directTransfer.BalanceProof.ChannelIdentifier,
		ChainId:                directTransfer.BalanceProof.ChainId,
	}

	return directTransfer
}

func CreateUnlock(channelState *NettingChannelState, messageIdentifier common.MessageID,
	paymentIdentifier common.PaymentID, secret common.Secret,
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
		log.Debug("[CreateUnlock] lock.Amount: %v, transferredAmount: %v\n",
			lock.Amount, transferredAmount)
	}

	merkleTree := computeMerkleTreeWithout(ourState.MerkleTree, lock.LockHash)
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
		Nonce:                  nonce,
		TransferredAmount:      transferredAmount,
		LockedAmount:           lockedAmount,
		LocksRoot:              common.Locksroot(locksRoot),
		TokenNetworkIdentifier: channelState.TokenNetworkIdentifier,
		ChannelIdentifier:      channelState.Identifier,
		ChainId:                channelState.ChainId,
	}

	unlockLock := &SendBalanceProof{
		SendMessageEvent: SendMessageEvent{
			Recipient:         common.Address(recipient),
			ChannelIdentifier: channelState.Identifier,
			MessageIdentifier: messageIdentifier,
		},
		PaymentIdentifier: paymentIdentifier,
		TokenAddress:      common.TokenAddress(tokenAddress),
		Secret:            secret,
		BalanceProof:      balanceProof,
	}

	return unlockLock, merkleTree, nil
}

func sendLockedTransfer(channelState *NettingChannelState,
	initiator common.InitiatorAddress, target common.Address,
	amount common.PaymentAmount, messageIdentifier common.MessageID,
	paymentIdentifier common.PaymentID, expiration common.BlockExpiration,
	secretHash common.SecretHash) *SendLockedTransfer {

	sendLockedTransferEvent, merkleTree := createSendLockedTransfer(
		channelState, initiator, target, amount, messageIdentifier,
		paymentIdentifier, expiration, secretHash)

	transfer := sendLockedTransferEvent.Transfer
	lock := transfer.Lock

	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:                  transfer.BalanceProof.Nonce,
		TransferredAmount:      transfer.BalanceProof.TransferredAmount,
		LockedAmount:           transfer.BalanceProof.LockedAmount,
		LocksRoot:              transfer.BalanceProof.LocksRoot,
		TokenNetworkIdentifier: transfer.BalanceProof.TokenNetworkIdentifier,
		ChannelIdentifier:      transfer.BalanceProof.ChannelIdentifier,
		ChainId:                transfer.BalanceProof.ChainId,
	}
	channelState.OurState.MerkleTree = merkleTree

	log.Debug("[SendLockedTransfer] Add SecretHashesToLockedLocks")
	channelState.OurState.SecretHashesToLockedLocks[common.SecretHash(lock.SecretHash)] = lock

	return sendLockedTransferEvent
}

func sendRefundTransfer(channelState *NettingChannelState, initiator common.InitiatorAddress,
	target common.Address, amount common.PaymentAmount, messageIdentifier common.MessageID,
	paymentIdentifier common.PaymentID, expiration common.BlockExpiration,
	secretHash common.SecretHash) (*SendRefundTransfer, error) {

	if _, ok := channelState.PartnerState.SecretHashesToLockedLocks[secretHash]; !ok {
		return nil, fmt.Errorf("Refunds are only valid for *known and pending* transfers ")
	}

	sendMediatedTransfer, merkleTree := createSendLockedTransfer(
		channelState, initiator, target, amount, messageIdentifier, paymentIdentifier,
		expiration, secretHash)

	mediatedTransfer := sendMediatedTransfer.Transfer
	lock := mediatedTransfer.Lock

	//todo
	//channelState.OurState.BalanceProof = mediatedTransfer.BalanceProof
	channelState.OurState.MerkleTree = merkleTree
	channelState.OurState.SecretHashesToLockedLocks[common.SecretHash(lock.SecretHash)] = lock

	refundTransfer := RefundFromSendmediated(sendMediatedTransfer)
	return refundTransfer, nil
}

func SendUnlock(channelState *NettingChannelState, messageIdentifier common.MessageID,
	paymentIdentifier common.PaymentID, secret common.Secret, secretHash common.SecretHash) *SendBalanceProof {
	lock := GetLock(channelState.OurState, secretHash)
	if lock == nil {
		return nil
	}

	unlock, merkleTree, _ := CreateUnlock(channelState, messageIdentifier, paymentIdentifier, secret, lock)

	channelState.OurState.BalanceProof = &BalanceProofSignedState{
		Nonce:                  unlock.BalanceProof.Nonce,
		TransferredAmount:      unlock.BalanceProof.TransferredAmount,
		LockedAmount:           unlock.BalanceProof.LockedAmount,
		LocksRoot:              unlock.BalanceProof.LocksRoot,
		TokenNetworkIdentifier: unlock.BalanceProof.TokenNetworkIdentifier,
		ChannelIdentifier:      unlock.BalanceProof.ChannelIdentifier,
		ChainId:                unlock.BalanceProof.ChainId,
	}
	channelState.OurState.MerkleTree = merkleTree
	DelLock(channelState.OurState, common.SecretHash(lock.SecretHash))
	return unlock
}

func EventsForClose(channelState *NettingChannelState, blockNumber common.BlockHeight) []Event {
	var events []Event

	status := GetStatus(channelState)
	if status == ChannelStateOpened || status == ChannelStateClosing {
		channelState.CloseTransaction = &TransactionExecutionStatus{
			blockNumber, 0, ""}

		closeEvent := &ContractSendChannelClose{ContractSendEvent{},
			channelState.Identifier, common.TokenAddress(channelState.TokenAddress),
			channelState.TokenNetworkIdentifier, channelState.PartnerState.BalanceProof}

		events = append(events, closeEvent)
	}

	return events
}

func createSendExpiredLock(senderEndState *NettingChannelEndState, lockedLock *HashTimeLockState,
	chainId common.ChainID, tokenNetworkIdentifier common.TokenNetworkID,
	channelIdentifier common.ChannelID, recipient common.Address) (*SendLockExpired, *MerkleTreeState) {

	nonce := getNextNonce(senderEndState)
	lockedAmount := getAmountLocked(senderEndState)
	balanceProof := senderEndState.BalanceProof
	updatedLockedAmount := common.TokenAmount(lockedAmount) - lockedLock.Amount

	if balanceProof == nil {
		//there should be a balance proof because a lock is expiring
		return nil, nil
	}
	transferredAmount := balanceProof.TransferredAmount

	merkleTree := computeMerkleTreeWithout(senderEndState.MerkleTree, lockedLock.LockHash)
	if merkleTree == nil {
		return nil, nil
	}

	locksRoot := MerkleRoot(merkleTree.Layers)

	//todo: check
	balanceProofEx := &BalanceProofUnsignedState{
		Nonce:                  nonce,
		TransferredAmount:      transferredAmount,
		LockedAmount:           updatedLockedAmount,
		LocksRoot:              common.Locksroot(locksRoot),
		TokenNetworkIdentifier: tokenNetworkIdentifier,
		ChannelIdentifier:      channelIdentifier,
		ChainId:                chainId,
	}

	sendLockExpired := &SendLockExpired{
		SendMessageEvent: SendMessageEvent{
			Recipient:         common.Address(recipient),
			MessageIdentifier: GetMsgID(),
		},
		BalanceProof: balanceProofEx,
		SecretHash:   common.SecretHash(lockedLock.SecretHash),
	}
	return sendLockExpired, merkleTree
}

func EventsForExpiredLock(channelState *NettingChannelState, lockedLock *HashTimeLockState) []Event {

	var lockExpired []Event
	sendLockExpired, merkleTree := createSendExpiredLock(
		channelState.OurState, lockedLock, channelState.ChainId,
		channelState.TokenNetworkIdentifier,
		channelState.Identifier, channelState.PartnerState.Address,
	)

	if sendLockExpired != nil {
		channelState.OurState.MerkleTree = merkleTree
		//todo
		//channelState.OurState.BalanceProof = sendLockExpired.BalanceProof

		DelUnclaimedLock(channelState.OurState, common.SecretHash(lockedLock.SecretHash))

		lockExpired = append(lockExpired, sendLockExpired)
		return lockExpired
	}
	return nil
}

func handleSendDirectTransfer(channelState *NettingChannelState, stateChange *ActionTransferDirect) TransitionResult {

	var events []Event

	amount := common.TokenAmount(stateChange.Amount)
	paymentIdentifier := stateChange.PaymentIdentifier
	targetAddress := stateChange.ReceiverAddress
	distributableAmount := GetDistributable(channelState.OurState, channelState.PartnerState)

	currentBalanceProof := getCurrentBalanceProof(channelState.OurState)
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
		messageIdentifier := GetMsgID()
		directTransfer := sendDirectTransfer(channelState, common.PaymentAmount(amount), messageIdentifier, paymentIdentifier)
		events = append(events, directTransfer)
	} else {
		if isOpen == false {
			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				common.Address(targetAddress), "Channel is not opened"}

			events = append(events, failure)
		} else if isValid == false {
			msg := fmt.Sprintf("Payment amount is invalid. Transfer %d", amount)
			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				common.Address(targetAddress), msg}

			events = append(events, failure)
		} else if canPay == false {
			msg := fmt.Sprintf("Payment amount exceeds the available capacity. Capacity:%d, Transfer:%d",
				distributableAmount, amount)

			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				common.Address(targetAddress), msg}

			events = append(events, failure)
		}
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
		channelState.PartnerState.SecretHashesToLockedLocks[common.SecretHash(lock.SecretHash)] = lock

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient:         common.Address(refund.Transfer.BalanceProof.Sender),
				ChannelIdentifier: ChannelIdentifierGlobalQueue,
				MessageIdentifier: refund.Transfer.MessageIdentifier,
			},
		}
		events = append(events, sendProcessed)
	} else {
		invalidRefund := &EventInvalidReceivedTransferRefund{
			PaymentIdentifier: receivedTransfer.PaymentIdentifier,
			Reason:            err.Error(),
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
				Recipient:         common.Address(stateChange.BalanceProof.Sender),
				ChannelIdentifier: ChannelIdentifierGlobalQueue,
				MessageIdentifier: stateChange.MessageIdentifier,
			},
		}
		events = append(events, sendProcessed)
	} else {
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
	log.Debug("[HandleReceiveLockedTransfer] LocksRoot:", mediatedTransfer.BalanceProof.LocksRoot)
	merkleTree, err := IsValidLockedTransfer(mediatedTransfer, channelState,
		channelState.PartnerState, channelState.OurState)

	var events []Event
	if err == nil {
		channelState.PartnerState.BalanceProof = mediatedTransfer.BalanceProof
		channelState.PartnerState.MerkleTree = merkleTree

		lock := mediatedTransfer.Lock
		channelState.PartnerState.SecretHashesToLockedLocks[common.SecretHash(lock.SecretHash)] = lock
		//addr2 := common2.Address(mediatedTransfer.BalanceProof.Sender)
		//fmt.Println("[HandleReceiveLockedTransfer] SendProcessed to: ", addr2.ToBase58())

		sendProcessed := &SendProcessed{SendMessageEvent: SendMessageEvent{
			Recipient:         common.Address(mediatedTransfer.BalanceProof.Sender),
			ChannelIdentifier: ChannelIdentifierGlobalQueue,
			MessageIdentifier: mediatedTransfer.MessageIdentifier,
		}}
		events = append(events, sendProcessed)
	} else {
		invalidLocked := &EventInvalidReceivedLockedTransfer{
			PaymentIdentifier: mediatedTransfer.PaymentIdentifier,
			Reason:            err.Error(),
		}
		events = append(events, invalidLocked)
	}
	return events, err
}

func handleReceiveDirectTransfer(channelState *NettingChannelState,
	directTransfer *ReceiveTransferDirect) TransitionResult {

	var events []Event

	isValid, msg := IsValidDirectTransfer(directTransfer, channelState,
		channelState.PartnerState, channelState.OurState)

	if isValid {
		currentBalanceProof := getCurrentBalanceProof(channelState.PartnerState)
		previousTransferredAmount := currentBalanceProof.transferredAmount

		newTransferredAmount := directTransfer.BalanceProof.TransferredAmount
		transferAmount := newTransferredAmount - previousTransferredAmount

		channelState.PartnerState.BalanceProof = directTransfer.BalanceProof
		paymentReceivedSuccess := &EventPaymentReceivedSuccess{
			channelState.PaymentNetworkIdentifier,
			channelState.TokenNetworkIdentifier,
			directTransfer.PaymentIdentifier,
			transferAmount,
			common.InitiatorAddress(channelState.PartnerState.Address)}

		sendProcessed := new(SendProcessed)
		sendProcessed.Recipient = common.Address(directTransfer.BalanceProof.Sender)
		sendProcessed.ChannelIdentifier = 0
		sendProcessed.MessageIdentifier = directTransfer.MessageIdentifier

		events = append(events, paymentReceivedSuccess)
		events = append(events, sendProcessed)
	} else {
		transferInvalidEvent := &EventTransferReceivedInvalidDirectTransfer{
			directTransfer.PaymentIdentifier,
			msg}

		events = append(events, transferInvalidEvent)
		log.Info("[handleReceiveDirectTransfer] EventTransferReceivedInvalidDirectTransfer: %s", msg)
	}

	return TransitionResult{channelState, events}
}

func HandleUnlock(channelState *NettingChannelState, unlock *ReceiveUnlock) (bool, []Event, string) {
	isValid, msg, unlockedMerkleTree := IsValidUnlock(unlock, channelState, channelState.PartnerState)
	var events []Event
	if isValid {
		channelState.PartnerState.BalanceProof = unlock.BalanceProof
		channelState.PartnerState.MerkleTree = unlockedMerkleTree

		secretHash := common.GetHash(unlock.Secret)
		DelLock(channelState.PartnerState, secretHash)

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient:         common.Address(unlock.BalanceProof.Sender),
				ChannelIdentifier: ChannelIdentifierGlobalQueue,
				MessageIdentifier: unlock.MessageIdentifier,
			},
		}
		events = append(events, sendProcessed)
	} else {
		log.Error("[HandleUnlock] ErrorMsg: ", msg)
		secretHash := common.GetHash(unlock.Secret)
		invalidUnlock := &EventInvalidReceivedUnlock{
			SecretHash: secretHash,
			Reason:     msg,
		}
		events = append(events, invalidUnlock)
	}
	return isValid, events, msg
}

func handleBlock(channelState *NettingChannelState, stateChange *Block,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event
	if GetStatus(channelState) == ChannelStateClosed {
		closedBlockHeight := channelState.CloseTransaction.FinishedBlockHeight
		settlementEnd := closedBlockHeight + common.BlockHeight(channelState.SettleTimeout)

		if stateChange.BlockHeight > settlementEnd && channelState.SettleTransaction == nil {
			channelState.SettleTransaction = &TransactionExecutionStatus{
				stateChange.BlockHeight, 0, ""}

			event := &ContractSendChannelSettle{ContractSendEvent{}, channelState.Identifier,
				common.TokenNetworkAddress(channelState.TokenNetworkIdentifier)}
			events = append(events, event)
		}
	}

	for isDepositConfirmed(channelState, blockNumber) {
		orderDepositTransaction := channelState.DepositTransactionQueue.Pop()
		applyChannelNewbalance(channelState,
			&orderDepositTransaction.Transaction)
	}

	return TransitionResult{channelState, events}
}

func handleChannelClosed(channelState *NettingChannelState, stateChange *ContractReceiveChannelClosed) TransitionResult {
	var events []Event

	justClosed := false

	status := GetStatus(channelState)
	if stateChange.ChannelIdentifier == channelState.Identifier &&
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
				ChannelIdentifier:      channelState.Identifier,
				TokenNetworkIdentifier: channelState.TokenNetworkIdentifier,
				BalanceProof:           balanceProof,
			}

			channelState.UpdateTransaction = &TransactionExecutionStatus{stateChange.BlockHeight,
				0, ""}

			events = append(events, update)
		}
	}

	return TransitionResult{channelState, events}
}

func handleChannelUpdatedTransfer(channelState *NettingChannelState,
	stateChange *ContractReceiveUpdateTransfer,
	blockNumber common.BlockHeight) TransitionResult {

	if stateChange.ChannelIdentifier == channelState.Identifier {
		channelState.UpdateTransaction = &TransactionExecutionStatus{
			0, 0, "success"}
	}

	return TransitionResult{channelState, nil}
}

func handleChannelSettled(channelState *NettingChannelState,
	stateChange *ContractReceiveChannelSettled,
	blockNumber common.BlockHeight) TransitionResult {

	var events []Event

	if stateChange.ChannelIdentifier == channelState.Identifier {
		setSettled(channelState, stateChange.BlockHeight)
		isSettlePending := false
		if channelState.OurUnlockTransaction != nil {
			isSettlePending = true
		}

		merkleTreeLeaves := []common.Keccak256{}

		//[TODO] support getBatchUnlock when support unlock
		if isSettlePending == false && merkleTreeLeaves != nil && len(merkleTreeLeaves) != 0 {
			onChainUnlock := &ContractSendChannelBatchUnlock{
				ContractSendEvent{}, common.TokenAddress(channelState.TokenAddress),
				channelState.TokenNetworkIdentifier, channelState.Identifier,
				channelState.PartnerState.Address}

			events = append(events, onChainUnlock)

			channelState.OurUnlockTransaction = &TransactionExecutionStatus{
				blockNumber, 0, ""}
		} else {
			channelState = nil
		}

	}

	return TransitionResult{channelState, events}
}

func handleChannelNewbalance(channelState *NettingChannelState,
	stateChange *ContractReceiveChannelNewBalance,
	blockNumber common.BlockHeight) TransitionResult {

	depositTransaction := stateChange.DepositTransaction

	if IsTransactionConfirmed(depositTransaction.DepositBlockHeight, blockNumber) {
		applyChannelNewbalance(channelState, &stateChange.DepositTransaction)
	} else {
		order := TransactionOrder{depositTransaction.DepositBlockHeight, depositTransaction}
		channelState.DepositTransactionQueue.Push(order)
	}

	return TransitionResult{channelState, nil}
}

func applyChannelNewbalance(channelState *NettingChannelState,
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
		iteration = handleChannelNewbalance(channelState, contractReceiveChannelNewBalance, blockNumber)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleReceiveDirectTransfer(channelState, receiveTransferDirect)
	}

	return iteration
}
