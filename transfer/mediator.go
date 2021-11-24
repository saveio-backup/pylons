package transfer

import (
	"crypto/sha256"
	"fmt"
	"github.com/saveio/pylons/common/constants"
	"math"
	"reflect"

	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
)

const MAXIMUM_PENDING_TRANSFERS = 160

var STATE_TRANSFER_PAID = []string{
	"payee_contract_unlock",
	"payee_balance_proof",
	"payer_balance_proof",
}

// TODO: fix expired state, it is not final
var STATE_TRANSFER_FINAL = []string{
	"payee_contract_unlock",
	"payee_balance_proof",
	"payee_expired",
	"payer_balance_proof",
	"payer_expired",
}

var VALID_PAYEE_STATES = []string{
	"payee_pending",
	"payee_secret_revealed",
	"payee_contract_unlock",
	"payee_balance_proof",
	"payee_expired",
}

var VALID_PAYER_STATES = []string{
	"payer_pending",
	"payer_secret_revealed",       //# SendSecretReveal was sent
	"payer_waiting_unlock",        //# ContractSendChannelBatchUnlock was sent
	"payer_waiting_secret_reveal", //# ContractSendSecretReveal was sent
	"payer_balance_proof",         //# ReceiveUnlock was received
	"payer_expired",               //# None of the above happened and the lock expired
}

var STATE_SECRET_KNOWN = []string{
	"payee_secret_revealed",
	"payee_contract_unlock",
	"payee_balance_proof",
	"payer_secret_revealed",
	"payer_waiting_unlock",
	"payer_balance_proof",
}

func MdIsLockValid(expiration common.BlockExpiration, blockNumber common.BlockHeight) bool {
	return blockNumber <= common.BlockHeight(expiration)
}

func MdIsSafeToWait(lockExpiration common.BlockExpiration, revealTimeout common.BlockTimeout,
	blockNumber common.BlockHeight) error {
	var err error
	//NOTE, need ensure lock_expiration > reveal_timeout
	if common.BlockHeight(lockExpiration) < blockNumber {
		err = fmt.Errorf("lock has expired, lockExpiration %d, blockNumber %d", lockExpiration, blockNumber)
		log.Warnf(err.Error())
		return err
	}

	lockTimeout := common.BlockHeight(lockExpiration) - blockNumber
	if common.BlockTimeout(lockTimeout) > revealTimeout {
		log.Debugf("lock timeout is safe. LockExpiration %d, blockNumber %d, revealTimeout %d",
			lockExpiration, blockNumber, revealTimeout)
		return nil
	}
	err = fmt.Errorf(`lock timeout is unsafe. timeout must be larger than %d,
		but it is %d. expiration:%d block number: %d`, revealTimeout, lockTimeout, lockExpiration, blockNumber)
	log.Debugf(err.Error())
	return err

}

func MdIsChannelUsable(candidateChannelState *NettingChannelState, transferAmount common.PaymentAmount,
	lockTimeout common.BlockTimeout) bool {
	pendingTransfersCount := getNumberOfPendingTransfers(candidateChannelState.OurState)
	distributable := GetDistributable(candidateChannelState.OurState,
		candidateChannelState.PartnerState)

	log.Debug(lockTimeout, candidateChannelState.SettleTimeout, candidateChannelState.RevealTimeout,
		pendingTransfersCount, transferAmount, distributable)

	channelState := GetStatus(candidateChannelState)
	log.Debug("channelState: ", channelState)

	if lockTimeout <= 0 {
		log.Error("[MdIsChannelUsable] lockTimeout is not valid")
		return false
	}
	if channelState != ChannelStateOpened {
		log.Error("[MdIsChannelUsable] ChannelState is not Opened")
		return false
	}
	if candidateChannelState.SettleTimeout < lockTimeout {
		log.Error("[MdIsChannelUsable] ChannelState.SettleTimeout < lockTimeout")
		return false
	}
	if candidateChannelState.RevealTimeout >= lockTimeout {
		log.Error("[MdIsChannelUsable] ChannelState.RevealTimeout >= lockTimeout")
		return false
	}
	if pendingTransfersCount >= MAXIMUM_PENDING_TRANSFERS {
		log.Error("[MdIsChannelUsable] pendingTransfersCount >= MAXIMUM_PENDING_TRANSFERS")
		return false
	}
	if transferAmount > common.PaymentAmount(distributable) {
		log.Error("[MdIsChannelUsable] transferAmount > distributable")
		return false
	}
	if !IsValidAmount(candidateChannelState.OurState, common.TokenAmount(transferAmount)) {
		log.Error("[MdIsChannelUsable] transferAmount is not valid")
		return false
	}

	return true
}

func MdIsSendTransferAlmostEqual(send *LockedTransferUnsignedState, received *LockedTransferSignedState) bool {
	//""" True if both transfers are for the same mediated transfer. """
	//# The only thing that may change is the direction of the transfer
	ret := send.PaymentId == received.PaymentId &&
		send.Token == received.Token &&
		send.Lock.Amount == received.Lock.Amount &&
		send.Lock.Expiration == received.Lock.Expiration &&
		send.Lock.SecretHash == received.Lock.SecretHash &&
		send.Initiator == received.Initiator &&
		common.Address(send.Target) == received.Target
	return ret
}

func MdHasSecretRegistrationStarted(channelStates []*NettingChannelState,
	transfersPair []*MediationPairState, secretHash common.SecretHash) bool {
	//# If it's known the secret is registered on-chain, the node should not send
	//# a new transaction. Note there is a race condition:
	//#
	//# - Node B learns the secret on-chain, sends a secret reveal to A
	//# - Node A receives the secret reveal off-chain prior to the event for the
	//#   secret registration, if the lock is in the danger zone A will try to
	//#   register the secret on-chain, because from its perspective the secret
	//#   is not there yet.

	isSecretRegisteredOnChain := false
	for i := 0; i < len(channelStates); i++ {
		payerChannel := channelStates[i]
		if IsSecretKnownOnChain(payerChannel.PartnerState, secretHash) {
			isSecretRegisteredOnChain = true
		}
	}
	hasPendingTransaction := false
	for i := 0; i < len(transfersPair); i++ {
		pair := transfersPair[i]
		if pair.PayerState == "payer_waiting_secret_reveal" {
			hasPendingTransaction = true
		}
	}
	return isSecretRegisteredOnChain || hasPendingTransaction
}

func MdFilterUsedRoutes(transfersPair []*MediationPairState, routes []RouteState) []RouteState {
	/*
			   This function makes sure we filter routes that have already been used.
		So in a setup like this, we want to make sure that node 2, having tried to
		route the transfer through 3 will also try 5 before sending it backwards to 1

		1 -> 2 -> 3 -> 4
			 v         ^
			 5 -> 6 -> 7
	*/
	channelIdToRoute := make(map[common.ChannelID]RouteState)
	for i := 0; i < len(routes); i++ {
		r := routes[i]
		channelIdToRoute[r.ChannelId] = r
	}

	for i := 0; i < len(transfersPair); i++ {
		pair := transfersPair[i]
		payerChannelId := pair.PayerTransfer.BalanceProof.ChannelId
		if _, exist := channelIdToRoute[payerChannelId]; exist {
			delete(channelIdToRoute, payerChannelId)
		}
		payeeChannelId := pair.PayeeTransfer.BalanceProof.ChannelId
		if _, exist := channelIdToRoute[payeeChannelId]; exist {
			delete(channelIdToRoute, payeeChannelId)
		}
	}
	var routeState []RouteState
	for _, rs := range channelIdToRoute {
		routeState = append(routeState, rs)
	}
	return routeState
}

func GetPayeeChannel(channelsMap map[common.ChannelID]*NettingChannelState,
	transferPair *MediationPairState) *NettingChannelState {

	payeeChannelId := transferPair.PayeeTransfer.BalanceProof.ChannelId
	return channelsMap[payeeChannelId]
}

func GetPayerChannel(channelsMap map[common.ChannelID]*NettingChannelState,
	transferPair *MediationPairState) *NettingChannelState {

	payerChannelId := transferPair.PayerTransfer.BalanceProof.ChannelId
	return channelsMap[payerChannelId]
}

func GetPendingTransferPairs(transfersPair []*MediationPairState) []*MediationPairState {
	var pendingPairs []*MediationPairState
	for i := 0; i < len(transfersPair); i++ {
		payeeFlag := false
		payeeState := transfersPair[i].PayeeState
		for _, v := range STATE_TRANSFER_FINAL {
			if v == payeeState {
				payeeFlag = true
				break
			}
		}
		payerFlag := false
		payerState := transfersPair[i].PayerState
		for _, v := range STATE_TRANSFER_FINAL {
			if v == payerState {
				payerFlag = true
				break
			}
		}
		if !payeeFlag || !payerFlag {
			pendingPairs = append(pendingPairs, transfersPair[i])
		}
	}
	return pendingPairs
}

func GetAmountWithoutFees(amountWithFees common.TokenAmount, channelIn *NettingChannelState) common.PaymentWithFeeAmount {
	amountWithoutFees := common.FeeAmount(amountWithFees)
	log.Debug("amount before fee:", amountWithoutFees)
	fee := channelIn.GetFeeSchedule().Flat
	proFee := channelIn.GetFeeSchedule().Proportional
	rate := float64(proFee) / math.Pow10(9)
	log.Debugf("flat fee: %d, rate: %f", fee, rate)

	fee += common.FeeAmount(float64(amountWithFees) * rate)
	if amountWithoutFees < fee {
		return 0
	}
	if float64(fee) <= float64(amountWithFees) * constants.DEFAULT_MEDIATION_FEE_LIMIT {
		amountWithoutFees -= fee
	} else {
		amountWithoutFees -= common.FeeAmount(float64(amountWithFees) * constants.DEFAULT_MEDIATION_FEE_LIMIT)
	}
	log.Debug("amount after fee:", amountWithoutFees)
	return common.PaymentWithFeeAmount(amountWithoutFees)
}

func SanityCheck(state *MediatorTransferState) error {

	//# if a transfer is paid we must know the secret
	var allTransfersStates []string
	for _, pair := range state.TransfersPair {
		allTransfersStates = append(allTransfersStates, pair.PayeeState)
	}
	for _, pair := range state.TransfersPair {
		allTransfersStates = append(allTransfersStates, pair.PayerState)
	}
	for _, stateStr := range allTransfersStates {
		if stateStr == STATE_TRANSFER_PAID[0] || stateStr == STATE_TRANSFER_PAID[1] ||
			stateStr == STATE_TRANSFER_PAID[2] {
			if state.SecretHash == common.EmptySecretHash {
				return fmt.Errorf("state.SecretHash is nil")
			}
		}
	}

	//# the "transitivity" for these values is checked below as part of
	//# almost_equal check
	if state.TransfersPair != nil {
		firstPair := state.TransfersPair[0]
		if state.SecretHash != common.SecretHash(firstPair.PayerTransfer.Lock.SecretHash) {
			return fmt.Errorf("[SanityCheck] state.SecretHash != firstPair.PayerTransfer.Lock.SecretHash")
		}
	}
	for i := 0; i < len(state.TransfersPair); i++ {
		pair := state.TransfersPair[i]
		if !MdIsSendTransferAlmostEqual(pair.PayeeTransfer, pair.PayerTransfer) {
			return fmt.Errorf("[SanityCheck] MdIsSendTransferAlmostEqual error")
		}
		payerFlag := false
		for j := 0; j < len(VALID_PAYER_STATES); j++ {
			if pair.PayerState == VALID_PAYER_STATES[j] {
				payerFlag = true
			}
		}
		if !payerFlag {
			return fmt.Errorf("[SanityCheck] PayerState not in VALID_PAYER_STATES error")
		}

		payeeFlag := false
		for j := 0; j < len(VALID_PAYEE_STATES); j++ {
			if pair.PayeeState == VALID_PAYEE_STATES[j] {
				payeeFlag = true
			}
		}
		if !payeeFlag {
			return fmt.Errorf("[SanityCheck] PayeeState not in VALID_PAYEE_STATES error")
		}
	}
	transferPairLen := len(state.TransfersPair)
	for i := 0; i < transferPairLen; i++ {
		original := state.TransfersPair[transferPairLen-i-1]
		refund := state.TransfersPair[i]
		if original.PayeeAddress != common.Address(refund.PayerTransfer.Initiator) {
			return fmt.Errorf("[SanityCheck] original.payee_address != refund.payer_address")
		}
		transferSent := original.PayeeTransfer
		transferReceived := refund.PayerTransfer
		if !MdIsSendTransferAlmostEqual(transferSent, transferReceived) {
			return fmt.Errorf("[SanityCheck] MdIsSendTransferAlmostEqual error")
		}
	}

	if state.WaitingTransfer != nil && state.TransfersPair != nil {
		transferSent := state.TransfersPair[transferPairLen-1].PayeeTransfer
		transferReceived := state.WaitingTransfer.Transfer
		if !MdIsSendTransferAlmostEqual(transferSent, transferReceived) {
			return fmt.Errorf("[SanityCheck] MdIsSendTransferAlmostEqual error")
		}
	}
	return nil
}

func clearIfFinalized(iteration *TransitionResult,
	channelsMap map[common.ChannelID]*NettingChannelState) *TransitionResult {

	//"""Clear the mediator task if all the locks have been finalized.
	//A lock is considered finalized if it has been removed from the merkle tree
	//offchain, either because the transfer was unlocked or expired, or because the
	//channel was settled on chain and therefore the channel is removed."""
	if IsStateNil(iteration.NewState) {
		log.Debug("[clearIfFinalized] iteration is nil")
		return iteration
	}
	state := iteration.NewState.(*MediatorTransferState)
	if state == nil {
		return iteration
	}

	//# Only clear the task if all channels have the lock cleared.
	secretHash := state.SecretHash
	for i := 0; i < len(state.TransfersPair); i++ {
		pair := state.TransfersPair[i]
		payerChannel := GetPayerChannel(channelsMap, pair)
		if payerChannel != nil && IsLockPending(payerChannel.PartnerState, secretHash) {
			return iteration
		}
		payeeChannel := GetPayeeChannel(channelsMap, pair)
		if payeeChannel != nil && IsLockPending(payeeChannel.OurState, secretHash) {
			return iteration
		}
	}

	if state.WaitingTransfer != nil {
		waitingTransfer := state.WaitingTransfer.Transfer
		waitingChannelId := waitingTransfer.BalanceProof.ChannelId
		waitingChannel := channelsMap[waitingChannelId]

		if waitingChannel != nil && IsLockPending(waitingChannel.PartnerState, secretHash) {
			return iteration
		}
	}
	return &TransitionResult{NewState: nil, Events: iteration.Events}
}

func nextChannelFromRoutes(availableRoutes []RouteState,
	channelsMap map[common.ChannelID]*NettingChannelState,
	transferAmount common.PaymentAmount, lockTimeout common.BlockTimeout) *NettingChannelState {
	/*
		Returns the first route that may be used to mediated the transfer.

		The routing service can race with local changes, so the recommended routes
		must be validated.
		Args:
			availableRoutes: Current available routes that may be used, it's
			assumed that the available_routes list is ordered from best to
			worst.
			channelsMap: Mapping from channel identifier
		to NettingChannelState.
			transferAmount: The amount of tokens that will be transferred
		through the given route.
			lockTimeout: Number of blocks until the lock expires, used to filter
		out channels that have a smaller settlement window.
			Returns: The next route.
	*/
	for _, route := range availableRoutes {
		channelState := channelsMap[route.ChannelId]

		addr := common.ToBase58(route.NodeAddress)
		if channelState == nil {
			log.Debug("nextChannelFromRoutes channelsMap == nil", addr)
			continue
		}
		log.Debug("nextChannelFromRoutes channelsMap != nil", addr)
		if MdIsChannelUsable(channelState, transferAmount, lockTimeout) {
			return channelState
		}
	}
	return nil
}

func forwardTransferPair(payerTransfer *LockedTransferSignedState, availableRoutes []RouteState,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) (*MediationPairState, []Event, error) {
	/*
		Given a payer transfer tries a new route to proceed with the mediation.

		Args:
		payer_transfer: The transfer received from the payer_channel.
		available_routes: Current available routes that may be used, it's
		assumed that the routes list is ordered from best to worst.
		channelidentifiers_to_channels: All the channels available for this
		transfer.
		pseudo_random_generator: Number generator to generate a message id.
		block_number: The current block number.
	*/
	var transferPair *MediationPairState
	var mediatedEvents []Event
	var lockTimeout common.BlockTimeout

	log.Debugf("[forwardTransferPair] lock expiration %d, blockNumber %d", payerTransfer.Lock.Expiration, blockNumber)
	// check overflow
	if payerTransfer.Lock.Expiration > blockNumber {
		lockTimeout = common.BlockTimeout(payerTransfer.Lock.Expiration - blockNumber)
	} else {
		lockTimeout = 0
	}

	payerChannel := channelsMap[payerTransfer.BalanceProof.ChannelId]
	payeeChannel := nextChannelFromRoutes(availableRoutes, channelsMap, common.PaymentAmount(payerTransfer.Lock.Amount), lockTimeout)

	amountAfterFee := GetAmountWithoutFees(payerTransfer.Lock.Amount, payerChannel)
	if amountAfterFee <= 0 {
		log.Warn("[forwardTransferPair] amount zero after fee")
	}

	if payeeChannel != nil {
		if payeeChannel.SettleTimeout < lockTimeout {
			return nil, nil, fmt.Errorf("[forwardTransferPair] payeeChannel.SettleTimeout < lockTimeout")
		}
		if payeeChannel.TokenAddress != payerTransfer.Token {
			return nil, nil, fmt.Errorf("[forwardTransferPair] payeeChannel.TokenAddress != payerTransfer.Token")
		}

		messageId := common.GetMsgID()
		lockedTransferEvent := sendLockedTransfer(payeeChannel, payerTransfer.Initiator, payerTransfer.Target,
			common.PaymentAmount(amountAfterFee), messageId, payerTransfer.PaymentId,
			common.BlockExpiration(payerTransfer.Lock.Expiration), payerTransfer.EncSecret,
			common.SecretHash(payerTransfer.Lock.SecretHash), payerTransfer.Mediators)
		if lockedTransferEvent == nil {
			log.Debug("[forwardTransferPair] lockedTransferEvent is nil")
			return nil, nil, fmt.Errorf("[forwardTransferPair] lockedTransferEvent is nil")
		}

		transferPair = &MediationPairState{
			PayerTransfer: payerTransfer,
			PayeeAddress:  payeeChannel.PartnerState.Address,
			PayeeTransfer: lockedTransferEvent.Transfer,
			PayerState:    "payer_pending",
			PayeeState:    "payee_pending",
		}
		mediatedEvents = append(mediatedEvents, lockedTransferEvent)
	} else {
		log.Debug("[forwardTransferPair] payeeChannel is nil")
	}
	return transferPair, mediatedEvents, nil
}

func backwardTransferPair(backwardChannel *NettingChannelState, payerTransfer *LockedTransferSignedState,
	blockNumber common.BlockHeight) (*MediationPairState, []Event, error) {
	//	""" Sends a transfer backwards, allowing the previous hop to try a new
	//	route.
	//
	//		When all the routes available for this node failed, send a transfer
	//	backwards with the same amount and secrethash, allowing the previous hop to
	//	do a retry.
	//
	//		Args:
	//backward_channel: The original channel which sent the mediated transfer
	//	to this node.
	//		payer_transfer: The *latest* payer transfer which is backing the
	//	mediation.
	//		block_number: The current block number.
	//
	//		Returns:
	//	The mediator pair and the correspoding refund event.
	//	"""
	var transferPair *MediationPairState
	var events []Event
	var lockTimeout common.BlockTimeout

	lock := payerTransfer.Lock
	log.Debugf("lock.Expiration: %d, blockNumber: %d", lock.Expiration, blockNumber)
	if lock.Expiration > blockNumber {
		lockTimeout = common.BlockTimeout(lock.Expiration - blockNumber)
	} else {
		lockTimeout = 0
	}

	//# Ensure the refund transfer's lock has a safe expiration, otherwise don't
	//# do anything and wait for the received lock to expire.
	if MdIsChannelUsable(backwardChannel, common.PaymentAmount(lock.Amount), lockTimeout) {
		messageId := common.GetMsgID()
		refundTransfer, err := sendRefundTransfer(backwardChannel, payerTransfer.Initiator, payerTransfer.Target,
			common.PaymentAmount(lock.Amount), messageId, payerTransfer.PaymentId,
			common.BlockExpiration(lock.Expiration), payerTransfer.EncSecret, common.SecretHash(lock.SecretHash))
		if err != nil {
			log.Error("[backwardTransferPair] sendRefundTransfer error: %s", err.Error())
			return transferPair, events, fmt.Errorf("backwardTransferPair sendRefundTransfer error: %s", err.Error())
		}

		transferPair = &MediationPairState{
			PayerTransfer: payerTransfer,
			PayeeAddress:  backwardChannel.PartnerState.Address,
			PayeeTransfer: refundTransfer.Transfer,
			PayerState:    "payer_pending",
			PayeeState:    "payee_pending",
		}
		events = append(events, refundTransfer)
	}
	return transferPair, events, nil
}

func setOffChainSecret(state *MediatorTransferState,
	channelsMap map[common.ChannelID]*NettingChannelState,
	secret common.Secret, secretHash common.SecretHash) []Event {

	var events []Event
	//""" Set the secret to all mediated transfers. """
	state.Secret = secret

	for _, pair := range state.TransfersPair {
		payerChannel := channelsMap[pair.PayerTransfer.BalanceProof.ChannelId]

		if payerChannel != nil {
			RegisterOffChainSecret(payerChannel, secret, secretHash)
		}

		payeeChannel, exist := channelsMap[pair.PayeeTransfer.BalanceProof.ChannelId]
		if exist {
			RegisterOffChainSecret(payeeChannel, secret, secretHash)
		}
	}

	//# The secret should never be revealed if `waiting_transfer` is not None.
	//# For this to happen this node must have received a transfer, which it did
	//# *not* mediate, and neverthless the secret was revealed.
	//#
	//# This can only be possible if the initiator reveals the secret without the
	//# target's secret request, or if the node which sent the `waiting_transfer`
	//# has sent another transfer which reached the target (meaning someone along
	//# the path will lose tokens).
	if state.WaitingTransfer != nil {
		payerChannel := channelsMap[state.WaitingTransfer.Transfer.BalanceProof.ChannelId]

		if payerChannel != nil {
			RegisterOffChainSecret(payerChannel, secret, secretHash)
		}
		unexpectedReveal := &EventUnexpectedSecretReveal{
			SecretHash: secretHash,
			Reason:     "The mediator has a waiting transfer.",
		}
		events = append(events, unexpectedReveal)
		return events
	}
	return nil
}

func setOnChainSecret(state *MediatorTransferState,
	channelsMap map[common.ChannelID]*NettingChannelState,
	secret common.Secret, secretHash common.SecretHash, blockNumber common.BlockHeight) []Event {
	var events []Event
	/*
		Set the secret to all mediated transfers.
		The secret should have been learned from the secret registry.
	*/
	state.Secret = secret

	for _, pair := range state.TransfersPair {
		payerChannel := channelsMap[pair.PayerTransfer.BalanceProof.ChannelId]
		if payerChannel != nil {
			RegisterOnChainSecret(payerChannel, secret, secretHash, blockNumber, true)
		}

		payeeChannel := channelsMap[pair.PayeeTransfer.BalanceProof.ChannelId]
		if payeeChannel != nil {
			RegisterOnChainSecret(payeeChannel, secret, secretHash, blockNumber, true)
		}
	}

	//# Like the off-chain secret reveal, the secret should never be revealed
	//# on-chain if there is a waiting transfer.

	if state.WaitingTransfer != nil {
		payerChannel := channelsMap[state.WaitingTransfer.Transfer.BalanceProof.ChannelId]
		if payerChannel != nil {
			RegisterOnChainSecret(payerChannel, secret, secretHash, blockNumber, true)
		}
		unexpectedReveal := &EventUnexpectedSecretReveal{
			SecretHash: secretHash,
			Reason:     "The mediator has a waiting transfer.",
		}

		events = append(events, unexpectedReveal)
		return events
	}
	return nil
}

func setOffchainRevealState(transfersPair []*MediationPairState, payeeAddress common.Address) {
	//""" Set the state of a transfer *sent* to a payee. """
	for _, pair := range transfersPair {
		if pair.PayeeAddress == payeeAddress {
			pair.PayeeState = "payee_secret_revealed"
		}
	}
}

func eventsForExpiredPairs(channelsMap map[common.ChannelID]*NettingChannelState,
	transfersPair []*MediationPairState, waitingTransfer *WaitingTransferState,
	blockNumber common.BlockHeight) []Event {

	//Informational events for expired locks.
	pendingTransfersPairs := GetPendingTransferPairs(transfersPair)

	var events []Event
	for _, pair := range pendingTransfersPairs {
		payerBalanceProof := pair.PayerTransfer.BalanceProof
		payerChannel := channelsMap[payerBalanceProof.ChannelId]
		if payerChannel == nil {
			continue
		}

		hasPayerTransferExpired := TransferExpired(pair.PayerTransfer, payerChannel, blockNumber)
		if hasPayerTransferExpired {
			//# For safety, the correct behavior is:
			//#
			//# - If the payee has been paid, then the payer must pay too.
			//#
			//#   And the corollary:
			//#
			//# - If the payer transfer has expired, then the payee transfer must
			//#   have expired too.
			//#
			//# The problem is that this corollary cannot be asserted. If a user
			//# is running Raiden without a monitoring service, then it may go
			//# offline after having paid a transfer to a payee, but without
			//# getting a balance proof of the payer, and once it comes back
			//# online the transfer may have expired.
			//#
			//# assert pair.payee_state == 'payee_expired'

			pair.PayerState = "payer_expired"
			unlockClaimFailed := &EventUnlockClaimFailed{
				Identifier: pair.PayerTransfer.PaymentId,
				SecretHash: common.SecretHash(pair.PayerTransfer.Lock.SecretHash),
				Reason:     "lock expired",
			}
			events = append(events, unlockClaimFailed)
		}
	}

	if waitingTransfer != nil && waitingTransfer.State != "expired" {
		waitingTransfer.State = "expired"
		unlockClaimFailed := &EventUnlockClaimFailed{
			Identifier: waitingTransfer.Transfer.PaymentId,
			SecretHash: common.SecretHash(waitingTransfer.Transfer.Lock.SecretHash),
			Reason:     "lock expired",
		}
		events = append(events, unlockClaimFailed)
	}
	return events
}

func eventsForSecretReveal(transfersPair []*MediationPairState, payerChannelId common.ChannelID,
	secret common.Secret) []Event {

	/*
		Reveal the secret off-chain.

		The secret is revealed off-chain even if there is a pending transaction to
		reveal it on-chain, this allows the unlock to happen off-chain, which is
		faster.

			This node is named N, suppose there is a mediated transfer with two refund
		transfers, one from B and one from C:

		A-N-B...B-N-C..C-N-D

		Under normal operation N will first learn the secret from D, then reveal to
		C, wait for C to inform the secret is known before revealing it to B, and
		again wait for B before revealing the secret to A.

			If B somehow sent a reveal secret before C and D, then the secret will be
		revealed to A, but not C and D, meaning the secret won't be propagated
		forward. Even if D sent a reveal secret at about the same time, the secret
		will only be revealed to B upon confirmation from C.

			If the proof doesn't arrive in time and the lock's expiration is at risk, N
		won't lose tokens since it knows the secret can go on-chain at any time.
	*/
	var events []Event
	for i, j := 0, len(transfersPair)-1; i < j; i, j = i+1, j-1 {
		transfersPair[i], transfersPair[j] = transfersPair[j], transfersPair[i]
	}

	for _, pair := range transfersPair {
		payeeKnowsSecret := false
		for _, payeeState := range STATE_SECRET_KNOWN {
			if pair.PayeeState == payeeState {
				payeeKnowsSecret = true
			}
		}

		payerKnowsSecret := false
		for _, payerState := range STATE_SECRET_KNOWN {
			if pair.PayerState == payerState {
				payerKnowsSecret = true
			}
		}

		isTransferPending := pair.PayerState == "payer_pending"

		if payeeKnowsSecret {
			log.Debug("[eventsForSecretReveal] payeeKnowsSecret true")
		}
		if !payerKnowsSecret {
			log.Debug("[eventsForSecretReveal] payerKnowsSecret")
		}
		if isTransferPending {
			log.Debug("[eventsForSecretReveal] isTransferPending true")
		} else {
			log.Debug("pair.PayerState: ", pair.PayerState)
		}

		shouldSendSecret := payeeKnowsSecret && !payerKnowsSecret && isTransferPending

		if shouldSendSecret {
			log.Debug("[eventsForSecretReveal] shouldSendSecret")
			messageId := common.GetMsgID()
			pair.PayerState = "payer_secret_revealed"
			payerTransfer := pair.PayerTransfer
			revealSecret := &SendSecretReveal{
				SendMessageEvent: SendMessageEvent{
					Recipient: common.Address(payerTransfer.BalanceProof.Sender),
					ChannelId: ChannelIdGlobalQueue,
					MessageId: messageId,
				},
				Secret: secret,
			}
			events = append(events, revealSecret)
		}
	}
	return events
}

func eventsForBalanceProof(channelsMap map[common.ChannelID]*NettingChannelState,
	transfersPair []*MediationPairState, blockNumber common.BlockHeight, secret common.Secret,
	secrethash common.SecretHash) ([]Event, error) {
	//""" While it's safe do the off-chain unlock. """

	var events []Event
	for i, j := 0, len(transfersPair)-1; i < j; i, j = i+1, j-1 {
		transfersPair[i], transfersPair[j] = transfersPair[j], transfersPair[i]
	}

	for _, pair := range transfersPair {
		payeeKnowsSecret := false
		for _, payeeState := range STATE_SECRET_KNOWN {
			if pair.PayeeState == payeeState {
				payeeKnowsSecret = true
			}
		}
		payeePayed := false
		for _, transferPaid := range STATE_TRANSFER_PAID {
			if pair.PayeeState == transferPaid {
				payeePayed = true
			}
		}

		payeeChannel := GetPayeeChannel(channelsMap, pair)
		payeeChannelOpen := payeeChannel != nil && GetStatus(payeeChannel) == ChannelStateOpened
		payerChannel := GetPayerChannel(channelsMap, pair)

		//# The mediator must not send to the payee a balance proof if the lock
		//# is in the danger zone, because the payer may not do the same and the
		//# on-chain unlock may fail. If the lock is nearing it's expiration
		//# block, then on-chain unlock should be done, and if successful it can
		//# be unlocked off-chain.

		if payerChannel == nil {
			continue
		}

		err := MdIsSafeToWait(common.BlockExpiration(pair.PayerTransfer.Lock.Expiration),
			payerChannel.RevealTimeout, blockNumber)
		if err != nil {
			continue
		}

		if payeeChannelOpen && payeeKnowsSecret && !payeePayed {
			//# At this point we are sure that payee_channel exists due to the
			//# payee_channel_open check above. So let mypy know about this
			if payeeChannel == nil {
				return nil, fmt.Errorf("[eventsForBalanceProof] payeeChannel is nil")
			}
			//payeeChannel = payeeChannel.(*NettingChannelState)
			pair.PayeeState = "payee_balance_proof"

			messageId := common.GetMsgID()
			unlockLock := SendUnlock(payeeChannel, messageId,
				pair.PayeeTransfer.PaymentId, secret, secrethash)

			unlockSuccess := &EventUnlockSuccess{
				Identifier: pair.PayerTransfer.PaymentId,
				SecretHash: common.SecretHash(pair.PayerTransfer.Lock.SecretHash),
			}
			events = append(events, unlockLock)
			events = append(events, unlockSuccess)
		}
	}
	return events, nil
}

func EventsForOnChainSecretReveal(channelState *NettingChannelState, secret common.Secret,
	expiration common.BlockExpiration) []Event {
	var events []Event
	cs := GetStatus(channelState)
	if cs == ChannelStateClosed || cs == ChannelStateOpened ||
		cs == ChannelStateClosing {
		revealEvent := &ContractSendSecretReveal{
			ContractSendExpireAbleEvent: ContractSendExpireAbleEvent{
				Expiration: expiration,
			},
			Secret: secret,
		}
		events = append(events, revealEvent)
	}
	return events
}

func eventsForOnChainSecretRevealIfDangerzone(channelsMap map[common.ChannelID]*NettingChannelState,
	secretHash common.SecretHash, transfersPair []*MediationPairState,
	blockNumber common.BlockHeight) []Event {
	//""" Reveal the secret on-chain if the lock enters the unsafe region and the
	//secret is not yet on-chain.
	//"""
	var events []Event

	var allPayerChannels []*NettingChannelState
	for _, pair := range transfersPair {
		channelState := GetPayerChannel(channelsMap, pair)
		if channelState != nil {
			allPayerChannels = append(allPayerChannels, channelState)
		}
	}

	transactionSent := MdHasSecretRegistrationStarted(allPayerChannels, transfersPair, secretHash)

	//# Only consider the transfers which have a pair. This means if we have a
	//# waiting transfer and for some reason the node knows the secret, it will
	//# not try to register it. Otherwise it would be possible for an attacker to
	//# reveal the secret late, just to force the node to send an unecessary
	//# transaction.

	for _, pair := range GetPendingTransferPairs(transfersPair) {
		payerChannel := GetPayerChannel(channelsMap, pair)
		if payerChannel == nil {
			continue
		}
		lock := pair.PayerTransfer.Lock

		err := MdIsSafeToWait(common.BlockExpiration(lock.Expiration), payerChannel.RevealTimeout, blockNumber)
		secretKnown := IsSecretKnown(payerChannel.PartnerState, common.SecretHash(pair.PayerTransfer.Lock.SecretHash))

		if err != nil && secretKnown {
			pair.PayerState = "payer_waiting_secret_reveal"

			if !transactionSent {
				secret := GetSecret(payerChannel.PartnerState, common.SecretHash(lock.SecretHash))
				revealEvents := EventsForOnChainSecretReveal(
					payerChannel, secret, common.BlockExpiration(lock.Expiration))

				events = append(events, revealEvents...)
				transactionSent = true
			}
		}
	}
	return events
}

func eventsForOnChainSecretRevealIfClosed(channelsMap map[common.ChannelID]*NettingChannelState,
	transfersPair []*MediationPairState, secret common.Secret,
	secretHash common.SecretHash) []Event {

	//""" Register the secret on-chain if the payer channel is already closed and
	//the mediator learned the secret off-chain.
	//
	//	Balance proofs are not exchanged for closed channels, so there is no reason
	//to wait for the unsafe region to register secret.
	//
	//	Note:
	//
	//If the secret is learned before the channel is closed, then the channel
	//will register the secrets in bulk, not the transfer.
	//"""
	var events []Event
	var transactionSent bool
	var allPayerChannels []*NettingChannelState
	for _, pair := range transfersPair {
		channelState := GetPayerChannel(channelsMap, pair)
		if channelState != nil {
			allPayerChannels = append(allPayerChannels, channelState)
		}
		transactionSent = MdHasSecretRegistrationStarted(
			allPayerChannels, transfersPair, secretHash)
	}

	//# Just like the case for entering the danger zone, this will only consider
	//# the transfers which have a pair.

	for _, pendingPair := range GetPendingTransferPairs(transfersPair) {
		payerChannel := GetPayerChannel(channelsMap, pendingPair)
		//# Don't register the secret on-chain if the channel is open or settled
		if payerChannel != nil && GetStatus(payerChannel) == ChannelStateClosed {
			pendingPair.PayerState = "payer_waiting_secret_reveal"

			if !transactionSent {
				partnerState := payerChannel.PartnerState
				lock := GetLock(partnerState, secretHash)
				revealEvents := EventsForOnChainSecretReveal(payerChannel, secret, common.BlockExpiration(lock.Expiration))
				events = append(events, revealEvents...)
				transactionSent = true
			}
		}
	}
	return events
}

func eventsToRemoveExpiredLocks(mediatorState *MediatorTransferState,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) ([]Event, error) {
	/*
		Clear the channels which have expired locks.

		This only considers the *sent* transfers, received transfers can only be
		updated by the partner.
	*/
	var events []Event
	for _, transferPair := range mediatorState.TransfersPair {
		balanceProof := transferPair.PayeeTransfer.BalanceProof
		channelId := balanceProof.ChannelId
		channelState := channelsMap[channelId]
		if channelState == nil {
			continue
		}

		secretHash := mediatorState.SecretHash
		var lock *HashTimeLockState

		flag1 := false
		flag2 := false
		for _, secretHashTmp := range channelState.OurState.SecretHashesToLockedLocks {
			if secretHash == common.SecretHash(secretHashTmp.SecretHash) {
				flag1 = true
			}
		}
		for secretHashTmp := range channelState.OurState.SecretHashesToUnLockedLocks {
			if secretHash == secretHashTmp {
				flag2 = true
			}
		}

		if flag1 {
			if flag2 {
				return nil, fmt.Errorf("secrethash not in OurState SecretHashesToUnLockedLocks")
			}
			lock = channelState.OurState.SecretHashesToLockedLocks[secretHash]
		} else if flag2 {
			lock = channelState.OurState.SecretHashesToUnLockedLocks[secretHash].Lock
		}
		if lock != nil {
			lockExpirationThreshold := lock.Expiration + common.BlockHeight(common.Config.ConfirmBlockCount)*2
			hasLockExpired, _ := IsLockExpired(channelState.OurState, lock, blockNumber, lockExpirationThreshold)
			if hasLockExpired {
				transferPair.PayeeState = "payee_expired"
				expiredLockEvents := EventsForExpiredLock(channelState, lock)
				events = append(events, expiredLockEvents...)

				unlockFailed := &EventUnlockFailed{
					Identifier: transferPair.PayeeTransfer.PaymentId,
					SecretHash: common.SecretHash(transferPair.PayeeTransfer.Lock.SecretHash),
					Reason:     "lock expired",
				}
				events = append(events, unlockFailed)
			}
		}
	}
	return events, nil
}

func secretLearned(state *MediatorTransferState, channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight, secret common.Secret, secretHash common.SecretHash, payerChannelId common.ChannelID,
	payeeAddress common.Address) (*TransitionResult, error) {
	/*
		Unlock the payee lock, reveal the lock to the payer, and if necessary
		register the secret on-chain.
	*/
	var events []Event
	secretRevealEvents := setOffChainSecret(state, channelsMap, secret, secretHash)

	setOffchainRevealState(state.TransfersPair, payeeAddress)

	OnChainSecretReveal := eventsForOnChainSecretRevealIfClosed(channelsMap,
		state.TransfersPair, secret, secretHash)

	offChainSecretReveal := eventsForSecretReveal(state.TransfersPair, payerChannelId, secret)

	balanceProof, err := eventsForBalanceProof(channelsMap, state.TransfersPair,
		blockNumber, secret, secretHash)
	if err != nil {
		return nil, err
	}

	events = append(events, secretRevealEvents...)
	events = append(events, offChainSecretReveal...)
	events = append(events, balanceProof...)
	events = append(events, OnChainSecretReveal...)

	iteration := &TransitionResult{NewState: state, Events: events}
	return iteration, nil
}

func mediateTransfer(state *MediatorTransferState, possibleRoutes []RouteState,
	payerChannel *NettingChannelState, channelsMap map[common.ChannelID]*NettingChannelState,
	payerTransfer *LockedTransferSignedState, blockNumber common.BlockHeight) (*TransitionResult, error) {
	/*
		""" Try a new route or fail back to a refund.

		The mediator can safely try a new route knowing that the tokens from
		payer_transfer will cover the expenses of the mediation. If there is no
		route available that may be used at the moment of the call the mediator may
		send a refund back to the payer, allowing the payer to try a different
		route.
		"""
	*/
	for _, tp := range state.TransfersPair {
		log.Debug("PayeeAddress: ", tp.PayeeAddress)
		log.Debug("PayerTransfer.Initiator", tp.PayerTransfer.Initiator)
		log.Debug("PayerTransfer.Target", tp.PayerTransfer.Target)
	}
	availableRoutes := MdFilterUsedRoutes(state.TransfersPair, possibleRoutes)
	if payerChannel.PartnerState.Address != payerTransfer.BalanceProof.Sender {
		return nil, fmt.Errorf("payerChannel.PartnerState.Address != payerTransfer.BalanceProof.Sender")
	}

	//fmt.Println("[mediateTransfer] availableRoutes: ")
	//for _, r := range availableRoutes {
	//	addr := common2.Address(r.NodeAddress)
	//	fmt.Println(addr.ToBase58())
	//}
	var events []Event
	transferPair, forwardEvents, err := forwardTransferPair(payerTransfer, availableRoutes, channelsMap, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("mediateTransfer forwardTransferPair error: %s", err.Error())
	}

	if transferPair != nil && forwardEvents != nil {
		events = append(events, forwardEvents...)
		state.TransfersPair = append(state.TransfersPair, transferPair)
	} else {
		var originalChannel *NettingChannelState
		if state.TransfersPair != nil {
			originalChannel = GetPayerChannel(channelsMap, state.TransfersPair[0])
		} else {
			originalChannel = payerChannel
		}

		var backwardEvents []Event
		if originalChannel != nil {
			transferPair, backwardEvents, err = backwardTransferPair(originalChannel, payerTransfer, blockNumber)
		} else {
			transferPair = nil
			backwardEvents = nil
		}

		if transferPair != nil && backwardEvents != nil {
			events = append(events, backwardEvents...)
			state.TransfersPair = append(state.TransfersPair, transferPair)
		} else {
			state.WaitingTransfer = &WaitingTransferState{Transfer: payerTransfer}
		}
	}
	return &TransitionResult{NewState: state, Events: events}, nil
}

func HandleInitMediator(stateChange *ActionInitMediator, channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	routes := stateChange.Routes
	fromRoute := stateChange.FromRoute
	fromTransfer := stateChange.FromTransfer
	payerChannel := channelsMap[fromRoute.ChannelId]

	//# There is no corresponding channel for the message, ignore it
	if payerChannel == nil {
		log.Warn("[HandleInitMediator] payerChannel is nil")
		return &TransitionResult{NewState: nil, Events: nil}
	}
	mediatorState := &MediatorTransferState{
		SecretHash: common.SecretHash(fromTransfer.Lock.SecretHash),
	}

	events, err := HandleReceiveLockedTransfer(payerChannel, fromTransfer)
	if err != nil {
		//# If the balance proof is not valid, do *not* create a task. Otherwise it's
		//# possible for an attacker to send multiple invalid transfers, and increase
		//# the memory usage of this Node.
		log.Warn("[HandleInitMediator] HandleReceiveLockedTransfer :", err.Error())
		return &TransitionResult{NewState: nil, Events: events}
	}
	for _, e := range events {
		log.Debug("[HandleInitMediator]: ", reflect.TypeOf(e).String())
	}

	iteration, err := mediateTransfer(mediatorState, routes, payerChannel,
		channelsMap, fromTransfer, blockNumber)

	events = append(events, iteration.Events...)
	for _, e := range iteration.Events {
		log.Debug("[HandleInitMediator]: ", reflect.TypeOf(e).String())
	}

	if IsStateNil(iteration.NewState) {
		log.Debug("[HandleInitMediator]     iteration.NewState == nil")
	}

	return &TransitionResult{NewState: iteration.NewState, Events: events}
}

func MdHandleBlock(mediatorState *MediatorTransferState, stateChange *Block,
	channelsMap map[common.ChannelID]*NettingChannelState) *TransitionResult {
	/*
		        After Raiden learns about a new block this function must be called to
			handle expiration of the hash time locks.
				Args:
			state: The current state.
					Return:
			TransitionResult: The resulting iteration
	*/
	var events []Event
	expiredLocksEvents, err := eventsToRemoveExpiredLocks(mediatorState, channelsMap,
		stateChange.BlockHeight)
	if len(expiredLocksEvents) != 0 {
		log.Debug("[MdHandleBlock] eventsToRemoveExpiredLocks len(expiredLocksEvents) != 0")
	} else {
		log.Debug("[MdHandleBlock] eventsToRemoveExpiredLocks len(expiredLocksEvents) == 0")
	}
	if err != nil {
		return nil
	}

	secretRevealEvents := eventsForOnChainSecretRevealIfDangerzone(channelsMap,
		mediatorState.SecretHash, mediatorState.TransfersPair, stateChange.BlockHeight)
	if len(secretRevealEvents) != 0 {
		log.Debug("[MdHandleBlock] eventsForOnChainSecretRevealIfDangerzone len(secretRevealEvents) != 0")
	} else {
		log.Debug("[MdHandleBlock] eventsForOnChainSecretRevealIfDangerzone len(secretRevealEvents) == 0")
	}

	unlockFailEvents := eventsForExpiredPairs(channelsMap, mediatorState.TransfersPair,
		mediatorState.WaitingTransfer, stateChange.BlockHeight)
	if len(unlockFailEvents) != 0 {
		log.Debug("[MdHandleBlock] eventsForExpiredPairs len(unlockFailEvents) != 0")
	} else {
		log.Debug("[MdHandleBlock] eventsForExpiredPairs len(unlockFailEvents) == 0")
	}

	events = append(events, unlockFailEvents...)
	events = append(events, secretRevealEvents...)
	events = append(events, expiredLocksEvents...)
	return &TransitionResult{NewState: mediatorState, Events: events}
}

func MdHandleRefundTransfer(mediatorState *MediatorTransferState, mediatorStateChange *ReceiveTransferRefund,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {
	//	     Validate and handle a ReceiveTransferRefund mediator_state change.
	//	A node might participate in mediated transfer more than once because of
	//	refund transfers, eg. A-B-C-B-D-T, B tried to mediate the transfer through
	//	C, which didn't have an available route to proceed and refunds B, at this
	//	point B is part of the path again and will try a new partner to proceed
	//	with the mediation through D, D finally reaches the target T.
	//		In the above scenario B has two pairs of payer and payee transfers:
	//payer:A payee:C from the first SendLockedTransfer
	//payer:C payee:D from the following SendRefundTransfer
	//Args:
	//	mediator_state (MediatorTransferState): Current mediator_state.
	//		mediator_state_change (ReceiveTransferRefund): The mediator_state change.
	//		Returns:
	//TransitionResult: The resulting iteration.
	var events []Event
	var iteration *TransitionResult
	if mediatorState.Secret == nil {
		//# The last sent transfer is the only one that may be refunded, all the
		//# previous ones are refunded already.

		pairLen := len(mediatorState.TransfersPair)
		transferPair := mediatorState.TransfersPair[pairLen-1]

		payeeTransfer := transferPair.PayeeTransfer
		payerTransfer := mediatorStateChange.Transfer
		channelId := payerTransfer.BalanceProof.ChannelId
		payerChannel := channelsMap[channelId]
		if payerChannel != nil {
			return &TransitionResult{NewState: mediatorState, Events: nil}
		}
		channelEvents, err := handleRefundTransfer(payeeTransfer, payerChannel, mediatorStateChange)
		if err != nil {
			return &TransitionResult{NewState: mediatorState, Events: channelEvents}
		}

		iteration, err = mediateTransfer(mediatorState, mediatorStateChange.Routes, payerChannel, channelsMap,
			payerTransfer, blockNumber)

		events = append(events, channelEvents...)
		events = append(events, iteration.Events...)
	}

	iteration = &TransitionResult{NewState: mediatorState, Events: events}
	return iteration
}

func handleOffChainSecretReveal(mediatorState *MediatorTransferState, mediatorStateChange *ReceiveSecretReveal,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) (*TransitionResult, error) {

	//""" Handles the secret reveal and sends SendBalanceProof/RevealSecret if necessary. """
	isValidReveal := IsValidSecretReveal(mediatorStateChange.Secret, mediatorState.SecretHash,
		mediatorStateChange.Secret)
	isSecretUnknown := mediatorState.Secret == nil

	//# a SecretReveal should be rejected if the payer transfer
	//# has expired. To check for this, we use the last
	//# transfer pair.
	var err error
	var iteration *TransitionResult
	pairLen := len(mediatorState.TransfersPair)
	transferPair := mediatorState.TransfersPair[pairLen-1]

	payerTransfer := transferPair.PayerTransfer
	payerChannelId := payerTransfer.BalanceProof.ChannelId
	payerChannel := channelsMap[payerChannelId]
	if payerChannel == nil {
		return &TransitionResult{NewState: mediatorState, Events: nil}, nil
	}

	//payeeTransfer := transferPair.PayeeTransfer
	//payeeChannelId := payeeTransfer.BalanceProof.ChannelId

	hasPayerTransferExpired := TransferExpired(transferPair.PayerTransfer,
		payerChannel, blockNumber)

	if isSecretUnknown && isValidReveal && !hasPayerTransferExpired {
		secretHash := sha256.Sum256(mediatorStateChange.Secret)
		iteration, err = secretLearned(mediatorState, channelsMap,
			blockNumber, mediatorStateChange.Secret, secretHash, payerChannelId, mediatorStateChange.Sender)
		if err != nil {
			return nil, err
		}

		sendProcessed := &SendProcessed{
			SendMessageEvent: SendMessageEvent{
				Recipient: common.Address(mediatorStateChange.Sender),
				ChannelId: ChannelIdGlobalQueue,
				MessageId: mediatorStateChange.MessageId,
			},
		}
		iteration.Events = append(iteration.Events, sendProcessed)
	} else {
		iteration = &TransitionResult{NewState: mediatorState, Events: nil}
	}
	return iteration, nil
}

func handleOnChainSecretReveal(mediatorState *MediatorTransferState,
	OnChainSecretReveal *ContractReceiveSecretReveal,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	//The secret was revealed on-chain, set the state of all transfers to secret known.
	var iteration *TransitionResult
	secretHash := OnChainSecretReveal.SecretHash
	isValidReveal := IsValidSecretReveal(OnChainSecretReveal.Secret, mediatorState.SecretHash, OnChainSecretReveal.Secret)

	if isValidReveal {
		var events []Event
		secret := OnChainSecretReveal.Secret
		//# Compare against the block number at which the event was emitted.
		blockNumber = OnChainSecretReveal.BlockHeight

		secretReveal := setOnChainSecret(mediatorState, channelsMap,
			secret, secretHash, blockNumber)

		balanceProof, err := eventsForBalanceProof(channelsMap,
			mediatorState.TransfersPair, blockNumber, secret, secretHash)
		if err != nil {
			return nil
		}
		events = append(events, secretReveal...)
		events = append(events, balanceProof...)
		iteration = &TransitionResult{NewState: mediatorState, Events: events}
	} else {
		iteration = &TransitionResult{NewState: mediatorState, Events: nil}
	}
	return iteration
}

func handleUnlock(mediatorState *MediatorTransferState, stateChange *ReceiveUnlock,
	channelsMap map[common.ChannelID]*NettingChannelState) *TransitionResult {

	//""" Handle a ReceiveUnlock state change. """
	var events []Event
	balanceProofSender := stateChange.BalanceProof.Sender
	channelId := stateChange.BalanceProof.ChannelId

	for _, pair := range mediatorState.TransfersPair {
		if pair.PayerTransfer.BalanceProof.Sender == balanceProofSender {
			channelState := channelsMap[channelId]

			if channelState != nil {
				isValid, channelEvents, _ := HandleUnlock(channelState, stateChange)

				events = append(events, channelEvents...)
				if isValid {
					unlock := &EventUnlockClaimSuccess{
						Identifier: pair.PayeeTransfer.PaymentId,
						SecretHash: common.SecretHash(pair.PayeeTransfer.Lock.SecretHash),
					}
					events = append(events, unlock)
					pair.PayerState = "payer_balance_proof"
				}
			}
		}
	}

	iteration := &TransitionResult{NewState: mediatorState, Events: events}
	return iteration
}

func handleLockExpired(mediatorState *MediatorTransferState, stateChange *ReceiveLockExpired,
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) *TransitionResult {

	var events []Event
	for _, transferPair := range mediatorState.TransfersPair {
		balanceProof := transferPair.PayerTransfer.BalanceProof
		channelState := channelsMap[balanceProof.ChannelId]

		if channelState == nil {
			return &TransitionResult{NewState: mediatorState, Events: nil}
		}
		result := handleReceiveLockExpired(channelState,
			stateChange, blockNumber)

		//Handling a receive_lock_expire should never delete the channel task
		if !IsStateNil(result.NewState) && reflect.TypeOf(result.NewState).String() != "*NettingChannelState" {
			//"Handling a receive_lock_expire should never delete the channel task"
		}

		events = append(events, result.Events...)
		nettingChannelState := result.NewState.(*NettingChannelState)
		if GetLock(nettingChannelState.PartnerState, mediatorState.SecretHash) != nil {
			transferPair.PayerState = "payer_expired"
		}
	}

	if mediatorState.WaitingTransfer != nil {
		waitingChannel := channelsMap[mediatorState.WaitingTransfer.Transfer.BalanceProof.ChannelId]

		if waitingChannel != nil {
			result := handleReceiveLockExpired(
				waitingChannel, stateChange, blockNumber)
			events = append(events, result.Events...)
		}
	}
	return &TransitionResult{NewState: mediatorState, Events: events}
}

func MdStateTransition(mediatorState *MediatorTransferState, stateChange interface{},
	channelsMap map[common.ChannelID]*NettingChannelState,
	blockNumber common.BlockHeight) (*TransitionResult, error) {
	//""" State machine for a node mediating a transfer. """
	//# pylint: disable=too-many-branches
	//# Notes:
	//# - A user cannot cancel a mediated transfer after it was initiated, she
	//#   may only reject to mediate before hand. This is because the mediator
	//#   doesn't control the secret reveal and needs to wait for the lock
	//#   expiration before safely discarding the transfer.

	var err error
	iteration := &TransitionResult{NewState: mediatorState, Events: nil}
	log.Debug("[MdStateTransition]: ", reflect.TypeOf(stateChange).String())
	if mediatorState == nil {
		log.Debug("[MdStateTransition] mediatorState is nil")
	}
	switch stateChange.(type) {
	case *ActionInitMediator:
		if mediatorState == nil {
			sc := stateChange.(*ActionInitMediator)
			iteration = HandleInitMediator(sc, channelsMap, blockNumber)
		} else {
			log.Debug("[MdStateTransition] mediatorState is not nil")
		}
	case *Block:
		sc := stateChange.(*Block)
		iteration = MdHandleBlock(mediatorState, sc, channelsMap)
	case *ReceiveTransferRefund:
		sc := stateChange.(*ReceiveTransferRefund)
		iteration = MdHandleRefundTransfer(mediatorState, sc, channelsMap, blockNumber)
	case *ReceiveSecretReveal:
		sc := stateChange.(*ReceiveSecretReveal)
		iteration, err = handleOffChainSecretReveal(mediatorState, sc, channelsMap, blockNumber)
		if err != nil {
			return nil, err
		}
	case *ContractReceiveSecretReveal:
		sc := stateChange.(*ContractReceiveSecretReveal)
		iteration = handleOnChainSecretReveal(mediatorState, sc, channelsMap, blockNumber)
	case *ReceiveUnlock:
		sc := stateChange.(*ReceiveUnlock)
		iteration = handleUnlock(mediatorState, sc, channelsMap)
	case *ReceiveLockExpired:
		sc := stateChange.(*ReceiveLockExpired)
		iteration = handleLockExpired(mediatorState, sc, channelsMap, blockNumber)
	default:
		log.Error("[MdStateTransition] stateChange Type error: ", reflect.TypeOf(stateChange).String())
		return nil, fmt.Errorf("[MdStateTransition] stateChange Type error ")
	}

	//# this is the place for paranoia
	if !IsStateNil(iteration.NewState) {
		if reflect.TypeOf(iteration.NewState).String() != "*transfer.MediatorTransferState" {
			return nil, fmt.Errorf("State Type is not MediatorTransferState ")
		}
		mediatorTransferState := iteration.NewState.(*MediatorTransferState)
		SanityCheck(mediatorTransferState)
	}
	return clearIfFinalized(iteration, channelsMap), nil
}
