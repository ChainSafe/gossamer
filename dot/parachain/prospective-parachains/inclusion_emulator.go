package prospectiveparachains

import (
	"bytes"
	"fmt"
	"iter"
	"maps"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ethereum/go-ethereum/common/math"
)

// prospectiveCandidate includes key informations that represents a candidate
// without pinning it to a particular session. For example, commitments are
// represented here, but the erasure-root is not. This means that, prospective
// candidates are not correlated to any session in particular.
type prospectiveCandidate struct {
	Commitments             parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
	PoVHash                 common.Hash
	ValidationCodeHash      parachaintypes.ValidationCodeHash
}

// relayChainBlockInfo contains minimum information about a relay-chain block.
type relayChainBlockInfo struct {
	Hash        common.Hash
	StorageRoot common.Hash
	Number      parachaintypes.BlockNumber
}

func checkModifications(c *parachaintypes.Constraints, modifications *constraintModifications) error {
	if modifications.HrmpWatermark != nil && modifications.HrmpWatermark.Type == Trunk {
		if !slices.Contains(c.HRMPInbound.ValidWatermarks, modifications.HrmpWatermark.Watermark()) {
			return &errDisallowedHrmpWatermark{BlockNumber: modifications.HrmpWatermark.Watermark()}
		}
	}

	for id, outboundHrmpMod := range modifications.OutboundHrmp {
		outbound, ok := c.HRMPChannelsOut[id]
		if !ok {
			return &errNoSuchHrmpChannel{paraID: id}
		}

		_, overflow := math.SafeSub(uint64(outbound.BytesRemaining), uint64(outboundHrmpMod.BytesSubmitted))
		if overflow {
			return &errHrmpBytesOverflow{
				paraID:         id,
				bytesRemaining: outbound.BytesRemaining,
				bytesSubmitted: outboundHrmpMod.BytesSubmitted,
			}
		}

		_, overflow = math.SafeSub(uint64(outbound.MessagesRemaining), uint64(outboundHrmpMod.MessagesSubmitted))
		if overflow {
			return &errHrmpMessagesOverflow{
				paraID:            id,
				messagesRemaining: outbound.MessagesRemaining,
				messagesSubmitted: outboundHrmpMod.MessagesSubmitted,
			}
		}
	}

	_, overflow := math.SafeSub(uint64(c.UMPRemaining), uint64(modifications.UmpMessagesSent))
	if overflow {
		return &errUmpMessagesOverflow{
			messagesRemaining: c.UMPRemaining,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}

	_, overflow = math.SafeSub(uint64(c.UMPRemainingBytes), uint64(modifications.UmpBytesSent))
	if overflow {
		return &errUmpBytesOverflow{
			bytesRemaining: c.UMPRemainingBytes,
			bytesSubmitted: modifications.UmpBytesSent,
		}
	}

	_, overflow = math.SafeSub(uint64(len(c.DMPRemainingMessages)), uint64(modifications.DmpMessagesProcessed))
	if overflow {
		return &errDmpMessagesUnderflow{
			messagesRemaining: uint32(len(c.DMPRemainingMessages)),
			messagesProcessed: modifications.DmpMessagesProcessed,
		}
	}

	if c.FutureValidationCode == nil && modifications.CodeUpgradeApplied {
		return errAppliedNonexistentCodeUpgrade
	}

	return nil
}

func applyModifications(c *parachaintypes.Constraints, modifications *constraintModifications) (
	*parachaintypes.Constraints, error) {
	newConstraints := c.Clone()

	if modifications.RequiredParent != nil {
		newConstraints.RequiredParent = *modifications.RequiredParent
	}

	if modifications.HrmpWatermark != nil {
		pos, found := slices.BinarySearch(
			newConstraints.HRMPInbound.ValidWatermarks,
			modifications.HrmpWatermark.Watermark())

		if found {
			// Exact match, so this is OK in all cases.
			newConstraints.HRMPInbound.ValidWatermarks = newConstraints.HRMPInbound.ValidWatermarks[pos+1:]
		} else {
			switch modifications.HrmpWatermark.Type {
			case Head:
				// Updates to Head are always OK.
				newConstraints.HRMPInbound.ValidWatermarks = newConstraints.HRMPInbound.ValidWatermarks[pos:]
			case Trunk:
				// Trunk update landing on disallowed watermark is not OK.
				return nil, &errDisallowedHrmpWatermark{BlockNumber: modifications.HrmpWatermark.Block}
			}
		}
	}

	for id, outboundHrmpMod := range modifications.OutboundHrmp {
		outbound, ok := newConstraints.HRMPChannelsOut[id]
		if !ok {
			return nil, &errNoSuchHrmpChannel{id}
		}

		if outboundHrmpMod.BytesSubmitted > outbound.BytesRemaining {
			return nil, &errHrmpBytesOverflow{
				paraID:         id,
				bytesRemaining: outbound.BytesRemaining,
				bytesSubmitted: outboundHrmpMod.BytesSubmitted,
			}
		}

		if outboundHrmpMod.MessagesSubmitted > outbound.MessagesRemaining {
			return nil, &errHrmpMessagesOverflow{
				paraID:            id,
				messagesRemaining: outbound.MessagesRemaining,
				messagesSubmitted: outboundHrmpMod.MessagesSubmitted,
			}
		}

		outbound.BytesRemaining -= outboundHrmpMod.BytesSubmitted
		outbound.MessagesRemaining -= outboundHrmpMod.MessagesSubmitted
	}

	if modifications.UmpMessagesSent > newConstraints.UMPRemaining {
		return nil, &errUmpMessagesOverflow{
			messagesRemaining: newConstraints.UMPRemaining,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}
	newConstraints.UMPRemaining -= modifications.UmpMessagesSent

	if modifications.UmpBytesSent > newConstraints.UMPRemainingBytes {
		return nil, &errUmpBytesOverflow{
			bytesRemaining: newConstraints.UMPRemainingBytes,
			bytesSubmitted: modifications.UmpBytesSent,
		}
	}
	newConstraints.UMPRemainingBytes -= modifications.UmpBytesSent

	if modifications.DmpMessagesProcessed > uint32(len(newConstraints.DMPRemainingMessages)) {
		return nil, &errDmpMessagesUnderflow{
			messagesRemaining: uint32(len(newConstraints.DMPRemainingMessages)),
			messagesProcessed: modifications.DmpMessagesProcessed,
		}
	} else {
		newConstraints.DMPRemainingMessages = newConstraints.DMPRemainingMessages[modifications.DmpMessagesProcessed:]
	}

	if modifications.CodeUpgradeApplied {
		if newConstraints.FutureValidationCode == nil {
			return nil, errAppliedNonexistentCodeUpgrade
		}

		newConstraints.ValidationCodeHash = newConstraints.FutureValidationCode.ValidationCodeHash
	}

	return newConstraints, nil
}

// outboundHrmpChannelModification represents modifications to outbound HRMP channels.
type outboundHrmpChannelModification struct {
	BytesSubmitted    uint32
	MessagesSubmitted uint32
}

// hrmpWatermarkUpdate represents an update to the HRMP Watermark.
type hrmpWatermarkUpdate struct {
	Type  hrmpWatermarkUpdateType
	Block parachaintypes.BlockNumber
}

// hrmpWatermarkUpdateType defines the type of HrmpWatermarkUpdate.
type hrmpWatermarkUpdateType int

const (
	Head hrmpWatermarkUpdateType = iota
	Trunk
)

// Watermark returns the block number of the HRMP Watermark update.
func (h hrmpWatermarkUpdate) Watermark() parachaintypes.BlockNumber {
	return h.Block
}

// constraintModifications represents modifications to constraints as a result of prospective candidates.
type constraintModifications struct {
	// The required parent head to build upon.
	RequiredParent *parachaintypes.HeadData
	// The new HRMP watermark.
	HrmpWatermark *hrmpWatermarkUpdate
	// Outbound HRMP channel modifications.
	OutboundHrmp map[parachaintypes.ParaID]outboundHrmpChannelModification
	// The amount of UMP XCM messages sent. `UMPSignal` and separator are excluded.
	UmpMessagesSent uint32
	// The amount of UMP XCM bytes sent. `UMPSignal` and separator are excluded.
	UmpBytesSent uint32
	// The amount of DMP messages processed.
	DmpMessagesProcessed uint32
	// Whether a pending code upgrade has been applied.
	CodeUpgradeApplied bool
}

func (cm *constraintModifications) Clone() *constraintModifications {
	return &constraintModifications{
		RequiredParent:       cm.RequiredParent,
		HrmpWatermark:        cm.HrmpWatermark,
		OutboundHrmp:         maps.Clone(cm.OutboundHrmp),
		UmpMessagesSent:      cm.UmpMessagesSent,
		UmpBytesSent:         cm.UmpBytesSent,
		DmpMessagesProcessed: cm.DmpMessagesProcessed,
		CodeUpgradeApplied:   cm.CodeUpgradeApplied,
	}
}

// Identity returns the 'identity' modifications: these can be applied to
// any constraints and yield the exact same result.
func NewConstraintModificationsIdentity() *constraintModifications {
	return &constraintModifications{
		OutboundHrmp: make(map[parachaintypes.ParaID]outboundHrmpChannelModification),
	}
}

// Stack stacks other modifications on top of these. This does no sanity-checking, so if
// `other` is garbage relative to `self`, then the new value will be garbage as well.
// This is an addition which is not commutative.
func (cm *constraintModifications) Stack(other *constraintModifications) {
	if other.RequiredParent != nil {
		cm.RequiredParent = other.RequiredParent
	}

	if other.HrmpWatermark != nil {
		cm.HrmpWatermark = other.HrmpWatermark
	}

	for id, mods := range other.OutboundHrmp {
		record, ok := cm.OutboundHrmp[id]
		if !ok {
			record = outboundHrmpChannelModification{}
		}

		record.BytesSubmitted += mods.BytesSubmitted
		record.MessagesSubmitted += mods.MessagesSubmitted
		cm.OutboundHrmp[id] = record
	}

	cm.UmpMessagesSent += other.UmpMessagesSent
	cm.UmpBytesSent += other.UmpBytesSent
	cm.DmpMessagesProcessed += other.DmpMessagesProcessed
	cm.CodeUpgradeApplied = cm.CodeUpgradeApplied || other.CodeUpgradeApplied
}

// Fragment represents another prospective parachain block
// This is a type which guarantees that the candidate is valid under the operating constraints
type Fragment struct {
	relayParent          *relayChainBlockInfo
	operatingConstraints *parachaintypes.Constraints
	candidate            *prospectiveCandidate
	modifications        *constraintModifications
}

func (f *Fragment) RelayParent() *relayChainBlockInfo {
	return f.relayParent
}

func (f *Fragment) Candidate() *prospectiveCandidate {
	return f.candidate
}

func (f *Fragment) ConstraintModifications() *constraintModifications {
	return f.modifications
}

// NewFragment creates a new Fragment. This fails if the fragment isnt in line
// with the operating constraints. That is, either its inputs or outputs fail
// checks against the constraints.
// This does not check that the collator signature is valid or whether the PoV is
// small enough.
func NewFragment(
	relayParent *relayChainBlockInfo,
	operatingConstraints *parachaintypes.Constraints,
	candidate *prospectiveCandidate) (*Fragment, error) {

	modifications, err := checkAgainstConstraints(
		relayParent,
		operatingConstraints,
		candidate.Commitments,
		candidate.ValidationCodeHash,
		candidate.PersistedValidationData,
	)
	if err != nil {
		return nil, err
	}

	return &Fragment{
		relayParent:          relayParent,
		operatingConstraints: operatingConstraints,
		candidate:            candidate,
		modifications:        modifications,
	}, nil
}

func checkAgainstConstraints(
	relayParent *relayChainBlockInfo,
	operatingConstraints *parachaintypes.Constraints,
	commitments parachaintypes.CandidateCommitments,
	validationCodeHash parachaintypes.ValidationCodeHash,
	persistedValidationData parachaintypes.PersistedValidationData,
) (*constraintModifications, error) {
	upwardMessages := make([]parachaintypes.UpwardMessage, 0)
	// filter UMP signals
	for upwardMessage := range skipUmpSignals(commitments.UpwardMessages) {
		upwardMessages = append(upwardMessages, upwardMessage)
	}

	umpMessagesSent := len(upwardMessages)
	umpBytesSent := 0
	for _, message := range upwardMessages {
		umpBytesSent += len(message)
	}

	hrmpWatermark := hrmpWatermarkUpdate{
		Type:  Trunk,
		Block: parachaintypes.BlockNumber(commitments.HrmpWatermark),
	}

	if parachaintypes.BlockNumber(commitments.HrmpWatermark) == relayParent.Number {
		hrmpWatermark.Type = Head
	}

	outboundHrmp := make(map[parachaintypes.ParaID]outboundHrmpChannelModification)
	var lastRecipient *parachaintypes.ParaID

	for i, message := range commitments.HorizontalMessages {
		if lastRecipient != nil && *lastRecipient >= parachaintypes.ParaID(message.Recipient) {
			return nil, &errHrmpMessagesDescendingOrDuplicate{index: uint(i)}
		}

		recipientParaID := parachaintypes.ParaID(message.Recipient)
		lastRecipient = &recipientParaID
		record, ok := outboundHrmp[recipientParaID]
		if !ok {
			record = outboundHrmpChannelModification{}
		}

		record.BytesSubmitted += uint32(len(message.Data))
		record.MessagesSubmitted++
		outboundHrmp[recipientParaID] = record
	}

	codeUpgradeApplied := false
	if operatingConstraints.FutureValidationCode != nil {
		codeUpgradeApplied = relayParent.Number >= operatingConstraints.FutureValidationCode.BlockNumber
	}

	modifications := &constraintModifications{
		RequiredParent:       &commitments.HeadData,
		HrmpWatermark:        &hrmpWatermark,
		OutboundHrmp:         outboundHrmp,
		UmpMessagesSent:      uint32(umpMessagesSent),
		UmpBytesSent:         uint32(umpBytesSent),
		DmpMessagesProcessed: commitments.ProcessedDownwardMessages,
		CodeUpgradeApplied:   codeUpgradeApplied,
	}

	err := validateAgainstConstraints(
		operatingConstraints,
		relayParent,
		commitments,
		persistedValidationData,
		validationCodeHash,
		modifications,
	)
	if err != nil {
		return nil, err
	}

	return modifications, nil
}

// skipUmpSignals is a utility function for skipping the UMP signals.
func skipUmpSignals(upwardMessages []parachaintypes.UpwardMessage) iter.Seq[parachaintypes.UpwardMessage] {
	var UmpSeparator = []byte{}
	return func(yield func(parachaintypes.UpwardMessage) bool) {
		for _, message := range upwardMessages {
			if !bytes.Equal([]byte(message), UmpSeparator) {
				if !yield([]byte(message)) {
					return
				}
			}

			return //nolint:staticcheck
		}
	}
}

func validateAgainstConstraints(
	constraints *parachaintypes.Constraints,
	relayParent *relayChainBlockInfo,
	commitments parachaintypes.CandidateCommitments,
	persistedValidationData parachaintypes.PersistedValidationData,
	validationCodeHash parachaintypes.ValidationCodeHash,
	modifications *constraintModifications,
) error {
	expectedPVD := parachaintypes.PersistedValidationData{
		ParentHead:             constraints.RequiredParent,
		RelayParentNumber:      uint32(relayParent.Number),
		RelayParentStorageRoot: relayParent.StorageRoot,
		MaxPovSize:             constraints.MaxPoVSize,
	}

	if !expectedPVD.Equal(persistedValidationData) {
		return fmt.Errorf("%w, expected: %v, got: %v",
			errPersistedValidationDataMismatch, expectedPVD, persistedValidationData)
	}

	if constraints.ValidationCodeHash != validationCodeHash {
		return &errValidationCodeMismatch{
			expected: constraints.ValidationCodeHash,
			got:      validationCodeHash,
		}
	}

	if relayParent.Number < constraints.MinRelayParentNumber {
		return &errRelayParentTooOld{
			minAllowed: constraints.MinRelayParentNumber,
			current:    relayParent.Number,
		}
	}

	if commitments.NewValidationCode != nil {
		restriction, err := constraints.UpgradeRestriction.Value()
		if err != nil {
			return fmt.Errorf("while retrieving value: %w", err)
		}
		switch restriction.(type) {
		case *parachaintypes.Present:
			return errCodeUpgradeRestricted
		}
	}

	announcedCodeSize := 0
	if commitments.NewValidationCode != nil {
		announcedCodeSize = len(*commitments.NewValidationCode)
	}

	if uint32(announcedCodeSize) > constraints.MaxCodeSize {
		return &errCodeSizeTooLarge{
			maxAllowed: constraints.MaxCodeSize,
			newSize:    uint32(announcedCodeSize),
		}
	}

	if modifications.DmpMessagesProcessed == 0 {
		if len(constraints.DMPRemainingMessages) > 0 && constraints.DMPRemainingMessages[0] <= relayParent.Number {
			return errDmpAdvancementRule
		}
	}

	if len(commitments.HorizontalMessages) > int(constraints.MaxNumHRMPPerCandidate) {
		return &errHrmpMessagesPerCandidateOverflow{
			messagesAllowed:   constraints.MaxNumHRMPPerCandidate,
			messagesSubmitted: uint32(len(commitments.HorizontalMessages)),
		}
	}

	if modifications.UmpMessagesSent > constraints.MaxNumUMPPerCandidate {
		return &errUmpMessagesPerCandidateOverflow{
			messagesAllowed:   constraints.MaxNumUMPPerCandidate,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}

	if err := checkModifications(constraints, modifications); err != nil {
		return &errOutputsInvalid{ModificationError: err}
	}

	return nil
}
