package prospectiveparachains

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	errCandidateAlreadyKnown           = errors.New("candidate already known")
	errZeroLengthCycle                 = errors.New("candidate's parent head is equal to its output head. Would introduce a cycle") //nolint:lll
	errCycle                           = errors.New("candidate would introduce a cycle")
	errMultiplePaths                   = errors.New("candidate would introduce two paths to the same output state")
	errIntroduceBackedCandidate        = errors.New("attempting to directly introduce a Backed candidate. It should first be introduced as Seconded") //nolint:lll,unused
	errParentCandidateNotFound         = errors.New("could not find parent of the candidate")
	errRelayParentMovedBackwards       = errors.New("relay parent would move backwards from the latest candidate in the chain")     //nolint:lll
	errPersistedValidationDataMismatch = errors.New("candidate does not match the persisted validation data provided alongside it") //nolint:lll
	errAppliedNonexistentCodeUpgrade   = errors.New("applied non existent code upgrade")
	errDmpAdvancementRule              = errors.New("dmp advancement rule")
	errCodeUpgradeRestricted           = errors.New("code upgrade restricted")
)

type errRelayParentPrecedesCandidatePendingAvailability struct {
	relayParentA, relayParentB common.Hash
}

func (e errRelayParentPrecedesCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("relay parent %x of the candidate precedes the relay parent %x of a pending availability candidate",
		e.relayParentA, e.relayParentB)
}

type errForkWithCandidatePendingAvailability struct {
	candidateHash parachaintypes.CandidateHash
}

func (e errForkWithCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("candidate would introduce a fork with a pending availability candidate: %x", e.candidateHash.Value)
}

type errForkChoiceRule struct {
	candidateHash parachaintypes.CandidateHash
}

func (e errForkChoiceRule) Error() string {
	return fmt.Sprintf("fork selection rule favours another candidate: %x", e.candidateHash.Value)
}

type errComputeConstraints struct {
	modificationErr error
}

func (e errComputeConstraints) Error() string {
	return fmt.Sprintf("could not compute candidate constraints: %s", e.modificationErr)
}

type errCheckAgainstConstraints struct {
	fragmentValidityErr error
}

func (e errCheckAgainstConstraints) Error() string {
	return fmt.Sprintf("candidate violates constraints: %s", e.fragmentValidityErr)
}

type errRelayParentNotInScope struct {
	relayParentA, relayParentB common.Hash
}

func (e errRelayParentNotInScope) Error() string {
	return fmt.Sprintf("relay parent %s not in scope, earliest relay parent allowed %s",
		e.relayParentA.String(), e.relayParentB.String())
}

type errUnexpectedAncestor struct {
	// The block number that this error occurred at
	number parachaintypes.BlockNumber
	// The previous seen block number, which did not match `number`.
	prev parachaintypes.BlockNumber
}

func (e errUnexpectedAncestor) Error() string {
	return fmt.Sprintf("unexpected ancestor %d, expected %d", e.number, e.prev)
}

type errDisallowedHrmpWatermark struct {
	BlockNumber parachaintypes.BlockNumber
}

func (e *errDisallowedHrmpWatermark) Error() string {
	return fmt.Sprintf("DisallowedHrmpWatermark(BlockNumber: %d)", e.BlockNumber)
}

type errNoSuchHrmpChannel struct {
	paraID parachaintypes.ParaID
}

func (e *errNoSuchHrmpChannel) Error() string {
	return fmt.Sprintf("NoSuchHrmpChannel(ParaId: %d)", e.paraID)
}

type errHrmpMessagesOverflow struct {
	paraID            parachaintypes.ParaID
	messagesRemaining uint32
	messagesSubmitted uint32
}

func (e *errHrmpMessagesOverflow) Error() string {
	return fmt.Sprintf("HrmpMessagesOverflow(ParaId: %d, MessagesRemaining: %d, MessagesSubmitted: %d)",
		e.paraID, e.messagesRemaining, e.messagesSubmitted)
}

type errHrmpBytesOverflow struct {
	paraID         parachaintypes.ParaID
	bytesRemaining uint32
	bytesSubmitted uint32
}

func (e *errHrmpBytesOverflow) Error() string {
	return fmt.Sprintf("HrmpBytesOverflow(ParaId: %d, BytesRemaining: %d, BytesSubmitted: %d)",
		e.paraID, e.bytesRemaining, e.bytesSubmitted)
}

type errUmpMessagesOverflow struct {
	messagesRemaining uint32
	messagesSubmitted uint32
}

func (e *errUmpMessagesOverflow) Error() string {
	return fmt.Sprintf("UmpMessagesOverflow(MessagesRemaining: %d, MessagesSubmitted: %d)",
		e.messagesRemaining, e.messagesSubmitted)
}

type errUmpBytesOverflow struct {
	bytesRemaining uint32
	bytesSubmitted uint32
}

func (e *errUmpBytesOverflow) Error() string {
	return fmt.Sprintf("UmpBytesOverflow(BytesRemaining: %d, BytesSubmitted: %d)", e.bytesRemaining, e.bytesSubmitted)
}

type errDmpMessagesUnderflow struct {
	messagesRemaining uint32
	messagesProcessed uint32
}

func (e *errDmpMessagesUnderflow) Error() string {
	return fmt.Sprintf("DmpMessagesUnderflow(MessagesRemaining: %d, MessagesProcessed: %d)",
		e.messagesRemaining, e.messagesProcessed)
}

type errValidationCodeMismatch struct {
	expected parachaintypes.ValidationCodeHash
	got      parachaintypes.ValidationCodeHash
}

func (e *errValidationCodeMismatch) Error() string {
	return fmt.Sprintf("ValidationCodeMismatch(Expected: %v, Got: %v)", e.expected, e.got)
}

type errOutputsInvalid struct {
	ModificationError error
}

func (e *errOutputsInvalid) Error() string {
	return fmt.Sprintf("OutputsInvalid(ModificationError: %v)", e.ModificationError)
}

type errCodeSizeTooLarge struct {
	maxAllowed uint32
	newSize    uint32
}

func (e *errCodeSizeTooLarge) Error() string {
	return fmt.Sprintf("CodeSizeTooLarge(MaxAllowed: %d, NewSize: %d)", e.maxAllowed, e.newSize)
}

type errRelayParentTooOld struct {
	minAllowed parachaintypes.BlockNumber
	current    parachaintypes.BlockNumber
}

func (e *errRelayParentTooOld) Error() string {
	return fmt.Sprintf("RelayParentTooOld(MinAllowed: %d, Current: %d)", e.minAllowed, e.current)
}

type errUmpMessagesPerCandidateOverflow struct {
	messagesAllowed   uint32
	messagesSubmitted uint32
}

func (e *errUmpMessagesPerCandidateOverflow) Error() string {
	return fmt.Sprintf("UmpMessagesPerCandidateOverflow(MessagesAllowed: %d, MessagesSubmitted: %d)",
		e.messagesAllowed, e.messagesSubmitted)
}

type errHrmpMessagesPerCandidateOverflow struct {
	messagesAllowed   uint32
	messagesSubmitted uint32
}

func (e *errHrmpMessagesPerCandidateOverflow) Error() string {
	return fmt.Sprintf("HrmpMessagesPerCandidateOverflow(MessagesAllowed: %d, MessagesSubmitted: %d)",
		e.messagesAllowed, e.messagesSubmitted)
}

type errHrmpMessagesDescendingOrDuplicate struct {
	index uint
}

func (e *errHrmpMessagesDescendingOrDuplicate) Error() string {
	return fmt.Sprintf("HrmpMessagesDescendingOrDuplicate(Index: %d)", e.index)
}
