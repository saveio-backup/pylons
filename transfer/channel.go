package transfer

import (
	"bytes"
	"container/list"
	"fmt"
	"math"
	"math/rand"

	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniChannel/utils"
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
	locksroot         typing.Locksroot
	nonce             typing.Nonce
	transferredAmount typing.TokenAmount
	lockedAmount      typing.TokenAmount
}

func compareLocksroot(one typing.Locksroot, two typing.Locksroot) bool {
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

func isDepositConfirmed(channelState *NettingChannelState, blockNumber typing.BlockHeight) bool {
	if len(channelState.DepositTransactionQueue) == 0 {
		return false
	}

	result := IsTransactionConfirmed(channelState.DepositTransactionQueue[0].BlockHeight, blockNumber)
	return result
}

func IsTransactionConfirmed(transactionBlockHeight typing.BlockHeight, blockchainBlockHeight typing.BlockHeight) bool {
	confirmationBlock := transactionBlockHeight + (typing.BlockHeight)(utils.DefaultNumberOfConfirmationsBlock)
	return blockchainBlockHeight > confirmationBlock
}

func IsBalanceProofSafeForOnchainOperations(balanceProof *BalanceProofSignedState) bool {
	totalAmount := balanceProof.TransferredAmount + balanceProof.LockedAmount
	return totalAmount <= math.MaxUint64
}

func IsValidAmount(endState *NettingChannelEndState, amount typing.TokenAmount) bool {
	balanceProofData := getCurrentBalanceproof(endState)
	currentTransferredAmount := balanceProofData.transferredAmount
	currentLockedAmount := balanceProofData.lockedAmount

	transferredAmountAfterUnlock := currentTransferredAmount + currentLockedAmount + amount

	return transferredAmountAfterUnlock <= math.MaxUint64
}

func IsValidSignature(balanceProof *BalanceProofSignedState, senderAddress typing.Address) (bool, string) {

	//[TODO] find similar ONT function for
	// to_canonical_address(eth_recover(data=data_that_was_signed, signature=balance_proof.signature))
	var signerAddress typing.Address

	balanceHash := HashBalanceData(
		balanceProof.TransferredAmount,
		balanceProof.LockedAmount,
		balanceProof.LocksRoot)

	dataThatWasSigned := PackBalanceProof(typing.Nonce(balanceProof.Nonce), balanceHash, typing.AdditionalHash(balanceProof.MessageHash[:]),
		balanceProof.ChannelIdentifier, typing.TokenNetworkAddress(balanceProof.TokenNetworkIdentifier), balanceProof.ChainId, 1)

	if dataThatWasSigned == nil {
		return false, string("Signature invalid, could not be recovered")
	}

	pubKey, err := utils.GetPublicKey(balanceProof.PublicKey)
	if err != nil {
		return false, string("Failed to get public key from balance proof")
	}

	err = utils.VerifySignature(pubKey, dataThatWasSigned, balanceProof.Signature)
	if err != nil {
		return false, string("Signature invalid, could not be recovered")
	}

	signerAddress = utils.GetAddressFromPubKey(pubKey)

	if typing.AddressEqual(senderAddress, signerAddress) == true {
		return true, "success"
	} else {
		return false, string("Signature was valid but the expected address does not match.")
	}

}

func IsBalanceProofUsableOnchain(
	receivedBalanceProof *BalanceProofSignedState,
	channelState *NettingChannelState,
	senderState *NettingChannelEndState) (bool, string) {

	expectedNonce := getNextNonce(senderState)

	isValidSignature, signatureMsg := IsValidSignature(receivedBalanceProof,
		senderState.Address)

	if GetStatus(channelState) != ChannelStateOpened {
		msg := fmt.Sprintf("The channel is already closed.")
		return false, msg

	} else if receivedBalanceProof.ChannelIdentifier != channelState.Identifier {
		msg := fmt.Sprintf("channel_identifier does not match. expected:%d got:%d",
			channelState.Identifier, receivedBalanceProof.ChannelIdentifier)
		return false, msg
	} else if typing.AddressEqual(typing.Address(receivedBalanceProof.TokenNetworkIdentifier), typing.Address(channelState.TokenNetworkIdentifier)) == false {
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

func IsValidDirecttransfer(directTransfer *ReceiveTransferDirect, channelState *NettingChannelState,
	senderState *NettingChannelEndState, receiverState *NettingChannelEndState) (bool, string) {

	receivedBalanceProof := directTransfer.BalanceProof

	currentBalanceProof := getCurrentBalanceproof(senderState)
	currentLocksroot := currentBalanceProof.locksroot
	currentTransferredAmount := currentBalanceProof.transferredAmount
	currentLockedAmount := currentBalanceProof.lockedAmount

	distributable := getDistributable(senderState, receiverState)
	amount := receivedBalanceProof.TransferredAmount - currentTransferredAmount

	isBalanceProofUsable, invalidBalanceProofMsg := IsBalanceProofUsableOnchain(
		receivedBalanceProof, channelState, senderState)

	if isBalanceProofUsable == false {
		msg := fmt.Sprintf("Invalid DirectTransfer message. {%s}", invalidBalanceProofMsg)
		return false, msg
	} else if compareLocksroot(receivedBalanceProof.LocksRoot, currentLocksroot) == false {
		var buf1, buf2 bytes.Buffer

		buf1.Write(currentBalanceProof.locksroot[:])
		buf2.Write(receivedBalanceProof.LocksRoot[:])
		msg := fmt.Sprintf("Invalid DirectTransfer message. Balance proof's locksroot changed, expected:%s got: %s",
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

func getAmountLocked(end_state *NettingChannelEndState) typing.Balance {
	var totalPending, totalUnclaimed, totalUnclaimedOnchain typing.TokenAmount

	var result typing.Balance
	result = (typing.Balance)(totalPending + totalUnclaimed + totalUnclaimedOnchain)
	return result
}

func getBalance(sender *NettingChannelEndState, receiver *NettingChannelEndState) typing.Balance {

	var senderTransferredAmount, receiverTransferredAmount typing.TokenAmount

	if sender.BalanceProof != nil {
		senderTransferredAmount = sender.BalanceProof.TransferredAmount
	}

	if receiver.BalanceProof != nil {
		receiverTransferredAmount = receiver.BalanceProof.TransferredAmount
	}

	result := sender.ContractBalance - senderTransferredAmount + receiverTransferredAmount
	return (typing.Balance)(result)

}

func getCurrentBalanceproof(endState *NettingChannelEndState) BalanceProofData {
	var locksroot typing.Locksroot
	var nonce typing.Nonce
	var transferredAmount, lockedAmount typing.TokenAmount

	balanceProof := endState.BalanceProof

	if balanceProof != nil {
		locksroot = balanceProof.LocksRoot
		nonce = (typing.Nonce)(balanceProof.Nonce)
		transferredAmount = balanceProof.TransferredAmount
		lockedAmount = (typing.TokenAmount)(getAmountLocked(endState))
	} else {
		locksroot = typing.Locksroot{}
		nonce = 0
		transferredAmount = 0
		lockedAmount = 0
	}

	return BalanceProofData{locksroot, nonce, transferredAmount, lockedAmount}
}

func getDistributable(sender *NettingChannelEndState, receiver *NettingChannelEndState) typing.TokenAmount {
	balanceProofData := getCurrentBalanceproof(sender)

	transferredAmount := balanceProofData.transferredAmount
	lockedAmount := balanceProofData.lockedAmount

	distributable := getBalance(sender, receiver) - getAmountLocked(sender)

	overflowLimit := Max(
		math.MaxUint64-(uint64)(transferredAmount)-(uint64)(lockedAmount),
		0,
	)

	result := Min(overflowLimit, (uint64)(distributable))
	return (typing.TokenAmount)(result)
}

func get_next_nonce(endState *NettingChannelEndState) typing.Nonce {
	if endState.BalanceProof != nil {
		return (typing.Nonce)(endState.BalanceProof.Nonce + 1)
	}

	return 1
}

func getNextNonce(endState *NettingChannelEndState) typing.Nonce {
	if endState.BalanceProof != nil {
		return endState.BalanceProof.Nonce + 1
	}

	return 1
}

func MerkletreeWidth(merkletree *MerkleTreeState) int {
	return len(merkletree.Layers[0])
}

func getNumberOfPendingTransfers(channelEndState *NettingChannelEndState) int {
	return MerkletreeWidth(channelEndState.Merkletree)
}

func GetStatus(channelState *NettingChannelState) string {
	var result string
	var finishedSuccessfully, running bool

	if channelState.SettleTransaction != nil {
		finishedSuccessfully =
			channelState.SettleTransaction.Result == TxnExecSucc

		running = channelState.SettleTransaction.FinishedBlockHeight == 0

		if finishedSuccessfully {
			result = ChannelStateSettled
		} else if running {
			result = ChannelStateSettling
		} else {
			result = ChannelStateUnusable
		}
	} else if channelState.CloseTransaction != nil {
		finishedSuccessfully =
			channelState.CloseTransaction.Result == TxnExecSucc

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

func setClosed(channelState *NettingChannelState, blockNumber typing.BlockHeight) {
	if channelState.CloseTransaction == nil {
		channelState.CloseTransaction = &TransactionExecutionStatus{
			0,
			blockNumber,
			TxnExecSucc}
	} else if channelState.CloseTransaction.FinishedBlockHeight == 0 {
		channelState.CloseTransaction.FinishedBlockHeight = blockNumber
		channelState.CloseTransaction.Result = TxnExecSucc
	}
}

func setSettled(channelState *NettingChannelState, blockNumber typing.BlockHeight) {
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

func updateContractBalance(endState *NettingChannelEndState, contractBalance typing.Balance) {
	if contractBalance > (typing.Balance)(endState.ContractBalance) {
		endState.ContractBalance = (typing.TokenAmount)(contractBalance)
	}
}

func computeMerkletreeWith(merkletree *MerkleTreeState, lockhash typing.LockHash) *MerkleTreeState {
	var result *MerkleTreeState

	leaves := merkletree.Layers[0]
	found := false
	for i := 0; i < len(leaves); i++ {
		temp := typing.Keccak256(lockhash)
		if typing.Keccak256Compare(&leaves[i], &temp) == 0 {
			found = true
			break
		}
	}

	if found == false {
		newLeaves := make([]typing.Keccak256, len(leaves)+1)
		copy(newLeaves, newLeaves)
		newLeaves = append(newLeaves, typing.Keccak256(lockhash))

		newLayers := computeLayers(newLeaves)
		result = new(MerkleTreeState)
		result.Layers = newLayers
	}

	return result
}

func computeMerkletreeWithout(merkletree *MerkleTreeState, lockhash typing.LockHash) *MerkleTreeState {
	var result *MerkleTreeState
	var i int

	leaves := merkletree.Layers[0]
	found := false
	for i = 0; i < len(leaves); i++ {
		temp := typing.Keccak256(lockhash)
		if typing.Keccak256Compare(&leaves[i], &temp) == 0 {
			found = true
			break
		}
	}

	if found == false {
		newLeaves := []typing.Keccak256{}
		newLeaves = append(newLeaves, leaves[0:i]...)
		if i+1 < len(leaves) {
			newLeaves = append(newLeaves, leaves[i+1:]...)
		}

		result = new(MerkleTreeState)

		if len(newLeaves) > 0 {
			newLayers := computeLayers(newLeaves)
			result.Layers = newLayers
		} else {
			result.init()
		}

	}

	return result
}

func createSendDirectTransfer(channelState *NettingChannelState, amount typing.PaymentAmount,
	messageIdentifier typing.MessageID, paymentIdentifier typing.PaymentID) *SendDirectTransfer {

	ourState := channelState.OurState
	partnerState := channelState.PartnerState

	if typing.TokenAmount(amount) > getDistributable(ourState, partnerState) {
		return nil
	}

	if GetStatus(channelState) != ChannelStateOpened {
		return nil
	}

	ourBalanceProof := channelState.OurState.BalanceProof

	var transferAmount typing.TokenAmount
	var locksroot typing.Locksroot

	if ourBalanceProof != nil {
		transferAmount = typing.TokenAmount(amount) + ourBalanceProof.TransferredAmount
		locksroot = ourBalanceProof.LocksRoot
	} else {
		transferAmount = typing.TokenAmount(amount)
		locksroot = typing.Locksroot{}
	}

	nonce := getNextNonce(ourState)
	recipient := partnerState.Address
	lockedAmount := getAmountLocked(ourState)

	balanceProof := &BalanceProofUnsignedState{
		nonce,
		transferAmount,
		typing.TokenAmount(lockedAmount),
		locksroot,
		channelState.TokenNetworkIdentifier,
		channelState.Identifier,
		channelState.ChainId}

	sendDirectTransfer := SendDirectTransfer{
		SendMessageEvent{typing.Address(recipient), channelState.Identifier, messageIdentifier},
		paymentIdentifier, balanceProof, typing.TokenAddress(channelState.TokenAddress)}

	return &sendDirectTransfer
}

func sendDirectTransfer(channelState *NettingChannelState, amount typing.PaymentAmount,
	messageIdentifier typing.MessageID, paymentIdentifier typing.PaymentID) *SendDirectTransfer {

	directTransfer := createSendDirectTransfer(
		channelState,
		amount,
		messageIdentifier,
		paymentIdentifier)

	//Construct fake BalanceProofSignedState from BalanceProofUnsignedState!
	balanceProof := &BalanceProofSignedState{
		Nonce:                  directTransfer.BalanceProof.Nonce,
		TransferredAmount:      directTransfer.BalanceProof.TransferredAmount,
		LockedAmount:           directTransfer.BalanceProof.LockedAmount,
		LocksRoot:              directTransfer.BalanceProof.LocksRoot,
		TokenNetworkIdentifier: directTransfer.BalanceProof.TokenNetworkIdentifier,
		ChannelIdentifier:      directTransfer.BalanceProof.ChannelIdentifier,
		ChainId:                directTransfer.BalanceProof.ChainId}

	channelState.OurState.BalanceProof = balanceProof

	return directTransfer
}

func EventsForClose(channelState *NettingChannelState, blockNumber typing.BlockHeight) *list.List {
	events := list.New()

	status := GetStatus(channelState)
	if status == ChannelStateOpened || status == ChannelStateClosing {
		channelState.CloseTransaction = &TransactionExecutionStatus{
			blockNumber, 0, ""}

		closeEvent := &ContractSendChannelClose{ContractSendEvent{},
			channelState.Identifier, typing.TokenAddress(channelState.TokenAddress),
			channelState.TokenNetworkIdentifier, channelState.PartnerState.BalanceProof}

		events.PushBack(closeEvent)
	}

	return events
}

func handleSendDirectTransfer(channelState *NettingChannelState, stateChange *ActionTransferDirect,
	pseudoRandomGenerator *rand.Rand) TransitionResult {

	events := list.New()

	amount := typing.TokenAmount(stateChange.Amount)
	paymentIdentifier := stateChange.PaymentIdentifier
	targetAddress := stateChange.ReceiverAddress
	distributableAmount := getDistributable(channelState.OurState, channelState.PartnerState)

	currentBalanceProof := getCurrentBalanceproof(channelState.OurState)
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
		messageIdentifier := GetMsgID(pseudoRandomGenerator)
		directTransfer := sendDirectTransfer(channelState, typing.PaymentAmount(amount), messageIdentifier, paymentIdentifier)
		events.PushBack(directTransfer)
	} else {
		if isOpen == false {
			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				typing.Address(targetAddress), "Channel is not opened"}

			events.PushBack(failure)
		} else if isValid == false {
			msg := fmt.Sprintf("Payment amount is invalid. Transfer %d", amount)
			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				typing.Address(targetAddress), msg}

			events.PushBack(failure)
		} else if canPay == false {
			msg := fmt.Sprintf("Payment amount exceeds the available capacity. Capacity:%d, Transfer:%d",
				distributableAmount, amount)

			failure := &EventPaymentSentFailed{channelState.PaymentNetworkIdentifier,
				channelState.TokenNetworkIdentifier, paymentIdentifier,
				typing.Address(targetAddress), msg}

			events.PushBack(failure)
		}
	}

	return TransitionResult{channelState, events}
}

func handleActionClose(channelState *NettingChannelState, close *ActionChannelClose,
	blockNumber typing.BlockHeight) TransitionResult {

	events := EventsForClose(channelState, blockNumber)
	return TransitionResult{channelState, events}
}

func handleReceiveDirecttransfer(channelState *NettingChannelState,
	directTransfer *ReceiveTransferDirect) TransitionResult {

	events := list.New()

	isValid, msg := IsValidDirecttransfer(directTransfer, channelState,
		channelState.PartnerState, channelState.OurState)

	if isValid {
		currentBalanceproof := getCurrentBalanceproof(channelState.PartnerState)
		previousTransferredAmount := currentBalanceproof.transferredAmount

		newTransferredAmount := directTransfer.BalanceProof.TransferredAmount
		transferAmount := newTransferredAmount - previousTransferredAmount

		channelState.PartnerState.BalanceProof = directTransfer.BalanceProof
		paymentReceivedSuccess := &EventPaymentReceivedSuccess{
			channelState.PaymentNetworkIdentifier,
			channelState.TokenNetworkIdentifier,
			directTransfer.PaymentIdentifier,
			transferAmount,
			typing.InitiatorAddress(channelState.PartnerState.Address)}

		sendProcessed := new(SendProcessed)
		sendProcessed.Recipient = typing.Address(directTransfer.BalanceProof.Sender)
		sendProcessed.ChannelIdentifier = 0
		sendProcessed.MessageIdentifier = directTransfer.MessageIdentifier

		events.PushBack(paymentReceivedSuccess)
		events.PushBack(sendProcessed)
	} else {
		transferInvalidEvent := &EventTransferReceivedInvalidDirectTransfer{
			directTransfer.PaymentIdentifier,
			msg}

		events.PushBack(transferInvalidEvent)
	}

	return TransitionResult{channelState, events}
}

func handleBlock(channelState *NettingChannelState, stateChange *Block,
	blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	if GetStatus(channelState) == ChannelStateClosed {
		closedBlockHeight := channelState.CloseTransaction.FinishedBlockHeight
		settlementEnd := closedBlockHeight + channelState.SettleTimeout

		if stateChange.BlockHeight > settlementEnd && channelState.SettleTransaction == nil {
			channelState.SettleTransaction = &TransactionExecutionStatus{
				stateChange.BlockHeight, 0, ""}

			event := &ContractSendChannelSettle{ContractSendEvent{}, channelState.Identifier,
				typing.TokenNetworkAddress(channelState.TokenNetworkIdentifier)}

			events.PushBack(event)
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
	events := list.New()

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

		if typing.AddressEqual(stateChange.TransactionFrom, channelState.OurState.Address) == false &&
			balanceProof != nil && channelState.UpdateTransaction == nil {
			callUpdate = true
		}

		if callUpdate {
			expiration := stateChange.BlockHeight + channelState.SettleTimeout
			update := &ContractSendChannelUpdateTransfer{
				ContractSendExpirableEvent{ContractSendEvent{}, typing.BlockExpiration(expiration)},
				channelState.Identifier, channelState.TokenNetworkIdentifier, balanceProof}

			channelState.UpdateTransaction = &TransactionExecutionStatus{stateChange.BlockHeight,
				0, ""}

			events.PushBack(update)
		}
	}

	return TransitionResult{channelState, events}
}

func handleChannelUpdatedTransfer(channelState *NettingChannelState,
	stateChange *ContractReceiveUpdateTransfer,
	blockNumber typing.BlockHeight) TransitionResult {

	if stateChange.ChannelIdentifier == channelState.Identifier {
		channelState.UpdateTransaction = &TransactionExecutionStatus{
			0, 0, "success"}
	}

	return TransitionResult{channelState, list.New()}
}

func handleChannelSettled(channelState *NettingChannelState,
	stateChange *ContractReceiveChannelSettled,
	blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()

	if stateChange.ChannelIdentifier == channelState.Identifier {
		setSettled(channelState, stateChange.BlockHeight)
		isSettlePending := false
		if channelState.OurUnlockTransaction != nil {
			isSettlePending = true
		}

		merkleTreeLeaves := []typing.Keccak256{}

		//[TODO] support getBatchUnlock when support unlock
		if isSettlePending == false && merkleTreeLeaves != nil && len(merkleTreeLeaves) != 0 {
			onChainUnlock := &ContractSendChannelBatchUnlock{
				ContractSendEvent{}, typing.TokenAddress(channelState.TokenAddress),
				channelState.TokenNetworkIdentifier, channelState.Identifier,
				channelState.PartnerState.Address}

			events.PushBack(onChainUnlock)

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
	blockNumber typing.BlockHeight) TransitionResult {

	depositTransaction := stateChange.DepositTransaction

	if IsTransactionConfirmed(depositTransaction.DepositBlockHeight, blockNumber) {
		applyChannelNewbalance(channelState, &stateChange.DepositTransaction)
	} else {
		order := TransactionOrder{depositTransaction.DepositBlockHeight, depositTransaction}
		channelState.DepositTransactionQueue.Push(order)
	}

	return TransitionResult{channelState, list.New()}
}

func applyChannelNewbalance(channelState *NettingChannelState,
	depositTransaction *TransactionChannelNewBalance) {
	participantAddress := depositTransaction.ParticipantAddress
	contractBalance := depositTransaction.ContractBalance

	if typing.AddressEqual(participantAddress, channelState.OurState.Address) {
		updateContractBalance(channelState.OurState, typing.Balance(contractBalance))
	} else if typing.AddressEqual(participantAddress, channelState.PartnerState.Address) {
		updateContractBalance(channelState.PartnerState, typing.Balance(contractBalance))
	}

	return
}

func StateTransitionForChannel(channelState *NettingChannelState, stateChange StateChange,
	pseudoRandomGenerator *rand.Rand, blockNumber typing.BlockHeight) TransitionResult {

	events := list.New()
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
		iteration = handleSendDirectTransfer(channelState, actionTransferDirect,
			pseudoRandomGenerator)
	case *ContractReceiveChannelClosed:
		contractReceiveChannelClosed, _ := stateChange.(*ContractReceiveChannelClosed)
		iteration = handleChannelClosed(channelState, contractReceiveChannelClosed)
	case *ContractReceiveUpdateTransfer:
		contractReceiveUpdateTransfer, _ := stateChange.(*ContractReceiveUpdateTransfer)
		iteration = handleChannelUpdatedTransfer(
			channelState,
			contractReceiveUpdateTransfer,
			blockNumber)
	case *ContractReceiveChannelSettled:
		contractReceiveChannelSettled, _ := stateChange.(*ContractReceiveChannelSettled)
		iteration = handleChannelSettled(channelState,
			contractReceiveChannelSettled,
			blockNumber)
	case *ContractReceiveChannelNewBalance:
		contractReceiveChannelNewBalance, _ := stateChange.(*ContractReceiveChannelNewBalance)
		iteration = handleChannelNewbalance(
			channelState,
			contractReceiveChannelNewBalance,
			blockNumber)
	case *ReceiveTransferDirect:
		receiveTransferDirect, _ := stateChange.(*ReceiveTransferDirect)
		iteration = handleReceiveDirecttransfer(
			channelState,
			receiveTransferDirect)
	}

	return iteration
}
