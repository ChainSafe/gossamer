package provisioner

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	disputemessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/btree"
	"go.uber.org/mock/gomock"
)

const (
	MaxDisputeVotesForwardedToRuntimeTest = 200
	VotesSelectionBatchSizeTest           = 11
)

type Enum interface {
	SetValue(any) error
}

func setEnumVariant[E Enum](variant any) E {
	var e E
	t := reflect.TypeOf(e)
	elem := reflect.New(t.Elem())
	e = elem.Interface().(E)

	e.SetValue(variant)
	return e
}

// TODO: this should be a common test function
func makeValidCandidateDescriptorV2(
	paraID parachaintypes.ParaID,
	relayParent common.Hash,
	coreIndex parachaintypes.CoreIndex,
	sessionIndex parachaintypes.SessionIndex,
	persistedValidationDataHash common.Hash,
	povHash common.Hash,
	erasureRoot common.Hash,
	paraHead common.Hash,
	validationCodeHash parachaintypes.ValidationCodeHash,
) *parachaintypes.CandidateDescriptorV2 {
	return &parachaintypes.CandidateDescriptorV2{
		ParaID:                      paraID,
		RelayParent:                 relayParent,
		CurrentVersion:              0,
		CoreIndex:                   uint16(coreIndex.Index),
		SessionIndex:                sessionIndex,
		Reserved1:                   [25]uint8{},
		PersistedValidationDataHash: persistedValidationDataHash,
		PovHash:                     povHash,
		ErasureRoot:                 erasureRoot,
		Reserved2:                   [64]uint8{},
		ParaHead:                    paraHead,
		ValidationCodeHash:          validationCodeHash,
	}
}

// TODO: this should be a common test function
func dummyCandidateDescriptorV2(relayParent common.Hash) *parachaintypes.CandidateDescriptorV2 {
	invalid := common.Hash{}
	return makeValidCandidateDescriptorV2(
		parachaintypes.ParaID(1),
		relayParent,
		parachaintypes.CoreIndex{Index: 1},
		parachaintypes.SessionIndex(1),
		invalid,
		invalid,
		invalid,
		invalid,
		parachaintypes.ValidationCodeHash(invalid),
	)
}

// TODO: this should be a common test function
func dummyCandidateCommitment(hd parachaintypes.HeadData) *parachaintypes.CandidateCommitments {
	return &parachaintypes.CandidateCommitments{
		HeadData:                  hd,
		UpwardMessages:            []parachaintypes.UpwardMessage{},
		NewValidationCode:         nil,
		HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             0,
	}
}

// TODO: this should be a common test function
func dummyCandidateReceiptV2(relayParent common.Hash) *parachaintypes.CandidateReceiptV2 {
	return &parachaintypes.CandidateReceiptV2{
		Descriptor:      *dummyCandidateDescriptorV2(relayParent),
		CommitmentsHash: dummyCandidateCommitment(parachaintypes.HeadData{}).Hash(),
	}
}

func rebuildCollatorField(v2 *parachaintypes.CandidateReceiptV2) parachaintypes.CollatorID {
	collatorID := make([]byte, 0, 32)
	coreIdx := make([]byte, 2)
	sessionIdx := make([]byte, 4)

	binary.NativeEndian.PutUint16(coreIdx, v2.Descriptor.CoreIndex)
	binary.NativeEndian.PutUint32(sessionIdx, uint32(v2.Descriptor.SessionIndex))

	collatorID = append(collatorID, v2.Descriptor.CurrentVersion)
	collatorID = append(collatorID, coreIdx...)
	collatorID = append(collatorID, sessionIdx...)
	collatorID = append(collatorID, v2.Descriptor.Reserved1[:]...)

	return parachaintypes.CollatorID(collatorID)
}

func rebuildSignatureField(v2 *parachaintypes.CandidateReceiptV2) parachaintypes.CollatorSignature {
	return parachaintypes.CollatorSignature(v2.Descriptor.Reserved2[:])
}

func fromCandidateRecepitV2ToV1(
	v2 *parachaintypes.CandidateReceiptV2,
) parachaintypes.CandidateReceipt {
	return parachaintypes.CandidateReceipt{
		Descriptor: parachaintypes.CandidateDescriptor{
			ParaID:                      v2.Descriptor.ParaID,
			RelayParent:                 v2.CommitmentsHash,
			Collator:                    rebuildCollatorField(v2),
			PersistedValidationDataHash: v2.Descriptor.PersistedValidationDataHash,
			PovHash:                     v2.Descriptor.PovHash,
			ErasureRoot:                 v2.Descriptor.ErasureRoot,
			Signature:                   rebuildSignatureField(v2),
			ParaHead:                    v2.Descriptor.ParaHead,
			ValidationCodeHash:          v2.Descriptor.ValidationCodeHash,
		},
		CommitmentsHash: v2.CommitmentsHash,
	}
}

// generateLocalVotes generates votes for a given statementKind for validators
// from startIdx (inclusive) up to count (exclusive).
func generateLocalVotes[T parachaintypes.ValidDisputeStatementKind | parachaintypes.InvalidDisputeStatementKind](
	t *testing.T,
	statementKind T,
	startIdx, count int,
) *btree.Map[parachaintypes.ValidatorIndex, parachaintypes.Vote[T]] {
	require.Less(t, startIdx, count)

	votes := btree.NewMap[parachaintypes.ValidatorIndex, parachaintypes.Vote[T]](count)
	for i := startIdx; i < count; i++ {
		votes.Set(
			parachaintypes.ValidatorIndex(i),
			parachaintypes.Vote[T]{
				Kind:      statementKind,
				Signature: parachaintypes.ValidatorSignature{},
			},
		)
	}
	return votes
}

// generateBitvec returns a bit slice of length validatorCount with indices
// from startIdx to startIdx+count set to true.
func generateBitvec(t *testing.T, validatorCount, startIdx, count int) parachaintypes.BitVec {
	require.Less(t, startIdx, count)
	require.Less(t, startIdx+count, validatorCount)

	bits := make([]bool, validatorCount)
	for i := startIdx; i < startIdx+count; i++ {
		bits[i] = true
	}

	bv, err := parachaintypes.NewBitVec(bits)
	require.NoError(t, err)
	return bv
}

func TestShouldKeepVoteBehaves(t *testing.T) {
	validatorsFor := bitVector(t, []bool{true, false, true, false, true})

	validatorsAgainst := bitVector(t, []bool{false, true, false, false, true})

	onchainState := parachaintypes.DisputeState{
		ValidatorsFor:     validatorsFor,
		ValidatorsAgainst: validatorsAgainst,
		Start:             1,
		ConcludedAt:       nil,
	}

	localValidKnown := parachaintypes.ValidatorIndex(0)
	localValidUnknown := parachaintypes.ValidatorIndex(3)

	localInvalidKnown := parachaintypes.ValidatorIndex(1)
	localInvalidUnknown := parachaintypes.ValidatorIndex(3)

	require.False(t, isVoteWorthToKeep(
		localValidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.True(t, isVoteWorthToKeep(
		localValidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.ValidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.False(t, isVoteWorthToKeep(
		localInvalidKnown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	require.True(t, isVoteWorthToKeep(
		localInvalidUnknown,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// double voting - onchain knows
	localDoubleVoteOnchainKnows := parachaintypes.ValidatorIndex(4)
	require.False(t, isVoteWorthToKeep(
		localDoubleVoteOnchainKnows,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// double voting - onchain doesn't know
	localDoubleVoteOnchainDoesntKnow := parachaintypes.ValidatorIndex(0)
	require.True(t, isVoteWorthToKeep(
		localDoubleVoteOnchainDoesntKnow,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		onchainState,
	))

	// empty onchain state
	emptyOnchainState := parachaintypes.DisputeState{
		ValidatorsFor:     parachaintypes.BitVec{},
		ValidatorsAgainst: parachaintypes.BitVec{},
		Start:             1,
		ConcludedAt:       nil,
	}
	require.True(t, isVoteWorthToKeep(
		localDoubleVoteOnchainDoesntKnow,
		*setEnumVariant[*parachaintypes.DisputeStatement](
			parachaintypes.InvalidDisputeStatement{
				Kind: *setEnumVariant[*parachaintypes.InvalidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
			},
		),
		emptyOnchainState,
	))
}

func TestPartitioningHappyCase(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)
	timeNow := uint64(time.Now().Unix())

	// Create one dispute for each partition
	inactiveUnknownOnchain := disputemessages.RecentDispute{
		SessionIndex:  0,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveUnknownOnchain)

	inactiveUnconcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  1,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveUnconcludedOnchain)
	onchainKey := parachaintypes.DisputeKey{
		SessionIndex:  inactiveUnconcludedOnchain.SessionIndex,
		CandidateHash: inactiveUnconcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	activeUnknownOnchain := disputemessages.RecentDispute{
		SessionIndex:  2,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x03}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeUnknownOnchain)

	activeUnconcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x04}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeUnconcludedOnchain)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  activeUnconcludedOnchain.SessionIndex,
		CandidateHash: activeUnconcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	activeConcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  4,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x05}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, activeConcludedOnchain)
	onchainConcludeBlockNumber := parachaintypes.BlockNumber(3)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  activeConcludedOnchain.SessionIndex,
		CandidateHash: activeConcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, true, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: &onchainConcludeBlockNumber,
	}

	inactiveConcludedOnchain := disputemessages.RecentDispute{
		SessionIndex:  5,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x06}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](
			parachaintypes.ConcludedFor{Timestamp: timeNow - parachaintypes.ActiveDurationSecs*2}),
	}
	input = append(input, inactiveConcludedOnchain)
	onchainKey = parachaintypes.DisputeKey{
		SessionIndex:  inactiveConcludedOnchain.SessionIndex,
		CandidateHash: inactiveConcludedOnchain.CandidateHash,
	}
	onchain[onchainKey] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: &onchainConcludeBlockNumber,
	}

	result := partitionRecentDisputes(input, onchain)

	// Check results
	require.Len(t, result.inactiveUnknownOnchain, 1)
	require.Equal(t, result.inactiveUnknownOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveUnknownOnchain.SessionIndex,
			CandidateHash: inactiveUnknownOnchain.CandidateHash,
		},
	)

	require.Len(t, result.inactiveUnconcludedOnchain, 1)
	require.Equal(t, result.inactiveUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveUnconcludedOnchain.SessionIndex,
			CandidateHash: inactiveUnconcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeUnknownOnchain, 1)
	require.Equal(t, result.activeUnknownOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeUnknownOnchain.SessionIndex,
			CandidateHash: activeUnknownOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeUnconcludedOnchain, 1)
	require.Equal(t, result.activeUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeUnconcludedOnchain.SessionIndex,
			CandidateHash: activeUnconcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.activeConcludedOnchain, 1)
	require.Equal(t, result.activeConcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  activeConcludedOnchain.SessionIndex,
			CandidateHash: activeConcludedOnchain.CandidateHash,
		},
	)

	require.Len(t, result.inactiveConcludedOnchain, 1)
	require.Equal(t, result.inactiveConcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  inactiveConcludedOnchain.SessionIndex,
			CandidateHash: inactiveConcludedOnchain.CandidateHash,
		},
	)
}

// This test verifies the double voting behaviour. Currently we don't care if a supermajority is
// achieved with or without the 'help' of a double vote (a validator voting for and against at the
// same time). This makes the test a bit pointless but anyway I'm leaving it here to make this
// decision explicit and have the test code ready in case this behaviour needs to be further tested
// in the future. Link to the PR with the discussions: https://github.com/paritytech/polkadot/pull/5567
func TestPartitioningDoubledOnchainVote(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)

	// Dispute A relies on a 'double onchain vote' to conclude. Validator with index 0 has voted
	// both `for` and `against`. Despite that this dispute should be considered 'can conclude
	// onchain'.
	disputeA := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, disputeA)

	// Dispute B has supermajority + 1 votes, so the doubled onchain vote doesn't affect it. It
	// should be considered as 'can conclude onchain'.
	disputeB := disputemessages.RecentDispute{
		SessionIndex:  4,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x02}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, disputeB)

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  disputeA.SessionIndex,
		CandidateHash: disputeA.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{true, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  disputeB.SessionIndex,
		CandidateHash: disputeB.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, true, true, true, true, true, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{true, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	result := partitionRecentDisputes(input, onchain)

	require.Len(t, result.activeUnconcludedOnchain, 0)
	require.Len(t, result.activeConcludedOnchain, 2)
}

func TestPartitioningDuplicatedDispute(t *testing.T) {
	input := []disputemessages.RecentDispute{}
	onchain := make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState)

	someDispute := disputemessages.RecentDispute{
		SessionIndex:  3,
		CandidateHash: parachaintypes.CandidateHash{Value: common.Hash{0x01}},
		DisputeStatus: *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{}),
	}
	input = append(input, someDispute)
	input = append(input, someDispute)

	onchain[parachaintypes.DisputeKey{
		SessionIndex:  someDispute.SessionIndex,
		CandidateHash: someDispute.CandidateHash,
	}] = parachaintypes.DisputeState{
		ValidatorsFor: bitVector(t,
			[]bool{true, true, true, false, false, false, false, false, false},
		),
		ValidatorsAgainst: bitVector(t,
			[]bool{false, false, false, false, false, false, false, false, false},
		),
		Start:       1,
		ConcludedAt: nil,
	}

	result := partitionRecentDisputes(input, onchain)

	require.Len(t, result.activeUnconcludedOnchain, 1)
	require.Equal(t, result.activeUnconcludedOnchain[0],
		parachaintypes.DisputeKey{
			SessionIndex:  someDispute.SessionIndex,
			CandidateHash: someDispute.CandidateHash,
		},
	)
}

func newLeaf() *parachaintypes.ActivatedLeaf {
	return &parachaintypes.ActivatedLeaf{
		Hash:   common.Hash(bytes.Repeat([]byte{0xAA}, 32)),
		Number: 0xAA,
	}
}

// TestDisputes mimics the Rust struct for testing purposes.
type TestDisputes struct {
	LocalDisputes   []disputemessages.RecentDispute
	VotesDB         map[parachaintypes.DisputeKey]parachaintypes.CandidateVotes
	OnchainDisputes map[parachaintypes.DisputeKey]parachaintypes.DisputeState
	ValidatorsCount int
}

func NewTestDisputes(vc int) *TestDisputes {
	return &TestDisputes{
		LocalDisputes:   []disputemessages.RecentDispute{},
		VotesDB:         make(map[parachaintypes.DisputeKey]parachaintypes.CandidateVotes),
		OnchainDisputes: make(map[parachaintypes.DisputeKey]parachaintypes.DisputeState),
		ValidatorsCount: vc,
	}
}

// addOffchainDispute adds an offchain dispute to TestDisputes.
func (td *TestDisputes) addOffchainDispute(
	t *testing.T,
	session parachaintypes.SessionIndex,
	candidateHash parachaintypes.CandidateHash,
	disputeStatus parachaintypes.DisputeStatus,
	localVotesCount int,
	dummyReceipt parachaintypes.CandidateReceipt,
) {
	// Create dispute tuple.
	dispute := disputemessages.RecentDispute{
		SessionIndex:  session,
		CandidateHash: candidateHash,
		DisputeStatus: disputeStatus,
	}
	td.LocalDisputes = append(td.LocalDisputes, dispute)

	key := parachaintypes.DisputeKey{
		SessionIndex:  session,
		CandidateHash: candidateHash,
	}

	// Generate valid votes using Explicit statement kind.
	validVotes := generateLocalVotes(
		t,
		*setEnumVariant[*parachaintypes.ValidDisputeStatementKind](parachaintypes.ExplicitStatement{}),
		0,
		localVotesCount,
	)

	// Insert CandidateVotes into VotesDB.
	td.VotesDB[key] = parachaintypes.CandidateVotes{
		CandidateReceipt: dummyReceipt,
		Valid:            validVotes,
		Invalid: btree.NewMap[parachaintypes.ValidatorIndex, parachaintypes.Vote[parachaintypes.InvalidDisputeStatementKind]](
			0,
		),
	}
}

// addOnchainDispute adds an onchain dispute to TestDisputes.
func (td *TestDisputes) addOnchainDispute(
	t *testing.T,
	session parachaintypes.SessionIndex,
	candidate parachaintypes.CandidateHash,
	status parachaintypes.DisputeStatus,
	onchainVotesCount int,
) {
	var concludedAt *parachaintypes.BlockNumber

	disputeStatus, err := status.Value()
	if err != nil {
		panic("error while getting dispute status")
	}

	switch disputeStatus.(type) {
	case parachaintypes.Active, parachaintypes.Confirmed:
		concludedAt = nil
	case parachaintypes.ConcludedAgainst, parachaintypes.ConcludedFor:
		var bn parachaintypes.BlockNumber = 1
		concludedAt = &bn
	default:
		concludedAt = nil
	}

	key := parachaintypes.DisputeKey{
		SessionIndex:  session,
		CandidateHash: candidate,
	}

	bv, err := parachaintypes.NewBitVec(make([]bool, td.ValidatorsCount))
	require.NoError(t, err)

	td.OnchainDisputes[key] = parachaintypes.DisputeState{
		ValidatorsFor:     generateBitvec(t, td.ValidatorsCount, 0, onchainVotesCount),
		ValidatorsAgainst: bv,
		Start:             1,
		ConcludedAt:       concludedAt,
	}
}

// addUnconfirmedDisputesConcludedOnchain adds unconfirmed disputes that are concluded onchain.
// It returns the used session index and the total difference: (localVotesCount - onchainVotesCount) * disputeCount.
func (td *TestDisputes) addUnconfirmedDisputesConcludedOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 90 / 100
	onchainVotesCount := td.ValidatorsCount * 80 / 100
	sessionIdx := parachaintypes.SessionIndex(0)
	lf := newLeaf()
	dummyReceiptv2 := dummyCandidateReceiptV2(lf.Hash)

	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)

		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceiptv2),
		)
		td.addOnchainDispute(t, sessionIdx, candidate, disputeStatus, onchainVotesCount)
	}
	diff := (localVotesCount - onchainVotesCount) * disputeCount
	return sessionIdx, diff
}

// addUnconfirmedDisputesUnconcludedOnchain adds unconfirmed disputes that are unconcluded onchain.
// Returns (sessionIdx, (localVotesCount - onchainVotesCount) * disputeCount).
func (td *TestDisputes) addUnconfirmedDisputesUnconcludedOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 90 / 100
	onchainVotesCount := td.ValidatorsCount * 40 / 100

	fmt.Println("onchain votes count", onchainVotesCount)

	sessionIdx := parachaintypes.SessionIndex(1)
	lf := newLeaf()
	dummyReceipt := dummyCandidateReceiptV2(lf.Hash)
	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)
		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceipt),
		)
		td.addOnchainDispute(t, sessionIdx, candidate, disputeStatus, onchainVotesCount)
	}
	diff := (localVotesCount - onchainVotesCount) * disputeCount
	return sessionIdx, diff
}

func (td *TestDisputes) addConfirmedDisputesUnkonwnOnChain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 90 / 100
	sessionIdx := parachaintypes.SessionIndex(2)
	lf := newLeaf()
	dummyReceiptv2 := dummyCandidateReceiptV2(lf.Hash)

	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)

		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Confirmed{})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceiptv2),
		)
	}
	return sessionIdx, localVotesCount * disputeCount
}

func (td *TestDisputes) addConcludedDisputesKnownOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) { //nolint:unparam
	localVotesCount := td.ValidatorsCount * 90 / 100
	onchainVotesCount := td.ValidatorsCount * 75 / 100
	sessionIdx := parachaintypes.SessionIndex(3)
	lf := newLeaf()
	dummyReceiptv2 := dummyCandidateReceiptV2(lf.Hash)

	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)

		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.ConcludedFor{Timestamp: 0})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceiptv2),
		)
		td.addOnchainDispute(t, sessionIdx, candidate, disputeStatus, onchainVotesCount)
	}
	diff := (localVotesCount - onchainVotesCount) * disputeCount
	return sessionIdx, diff
}

// addConcludedDisputesUnknownOnchain adds concluded disputes unknown onchain.
// Returns (sessionIdx, localVotesCount * disputeCount).
func (td *TestDisputes) addConcludedDisputesUnknownOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 90 / 100
	sessionIdx := parachaintypes.SessionIndex(4)
	lf := newLeaf()
	dummyReceipt := dummyCandidateReceiptV2(lf.Hash)
	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)
		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.ConcludedFor{Timestamp: 0})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceipt),
		)
	}
	return sessionIdx, localVotesCount * disputeCount
}

// addUnconfirmedDisputesKnownOnchain adds unconfirmed disputes that are known onchain.
// It registers both offchain and onchain disputes.
// Returns (sessionIdx, (localVotesCount - onchainVotesCount) * disputeCount).
func (td *TestDisputes) addUnconfirmedDisputesKnownOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 10 / 100
	onchainVotesCount := td.ValidatorsCount * 10 / 100
	sessionIdx := parachaintypes.SessionIndex(5)
	lf := newLeaf()
	dummyReceipt := dummyCandidateReceiptV2(lf.Hash)
	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)
		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceipt),
		)
		td.addOnchainDispute(t, sessionIdx, candidate, disputeStatus, onchainVotesCount)
	}
	diff := (localVotesCount - onchainVotesCount) * disputeCount
	return sessionIdx, diff
}

// addUnconfirmedDisputesUnknownOnchain adds unconfirmed disputes unknown onchain.
// Only offchain disputes are registered.
// Returns (sessionIdx, localVotesCount * disputeCount).
func (td *TestDisputes) addUnconfirmedDisputesUnknownOnchain(
	t *testing.T,
	disputeCount int,
) (parachaintypes.SessionIndex, int) {
	localVotesCount := td.ValidatorsCount * 10 / 100
	sessionIdx := parachaintypes.SessionIndex(6)
	lf := newLeaf()
	dummyReceipt := dummyCandidateReceiptV2(lf.Hash)
	for i := 0; i < disputeCount; i++ {
		rndHash := make([]byte, 32)
		_, err := rand.Read(rndHash)
		require.NoError(t, err)
		candidate := parachaintypes.CandidateHash{Value: common.Hash(rndHash)}
		disputeStatus := *setEnumVariant[*parachaintypes.DisputeStatus](parachaintypes.Active{})
		td.addOffchainDispute(
			t,
			sessionIdx,
			candidate,
			disputeStatus,
			localVotesCount,
			fromCandidateRecepitV2ToV1(dummyReceipt),
		)
	}
	return sessionIdx, localVotesCount * disputeCount
}

func mockRuntime(t *testing.T, disputesDB *TestDisputes) BlockState {
	ctrl := gomock.NewController(t)

	mockRuntime := NewMockInstance(ctrl)
	mockRuntime.EXPECT().
		ParachainHostDisputes().
		Return(disputesDB.OnchainDisputes, nil).
		AnyTimes()

	mockBlockState := NewMockBlockState(ctrl)
	mockBlockState.EXPECT().
		GetRuntime(gomock.AssignableToTypeOf(common.Hash{})).
		Return(mockRuntime, nil).
		AnyTimes()

	return mockBlockState
}

// mockOverseer processes only supported DisputeCoordinator messages.
func mockOverseer(receiver <-chan any, disputesDB *TestDisputes, voteQueriesCount *int) {
	for msg := range receiver {
		switch m := msg.(type) {
		case disputemessages.GetRecentDisputes:
			// Respond with local disputes
			m.Response <- disputesDB.LocalDisputes
		case disputemessages.QueryCandidateVotes:
			*voteQueriesCount++
			var res []disputemessages.CandidateVotesResponse
			for _, d := range m.Query {
				votes, ok := disputesDB.VotesDB[d]
				if !ok {
					// Use empty votes if none found.
					votes = parachaintypes.CandidateVotes{}
				}
				res = append(res, disputemessages.CandidateVotesResponse{
					SessionIndex:   d.SessionIndex,
					CandidateHash:  d.CandidateHash,
					CandidateVotes: votes,
				})
			}
			m.Response <- res
		default:
			panic(fmt.Sprintf("Unexpected message: %+v", m))
		}
	}
}

func TestNormalFlow(t *testing.T) {
	const (
		validatorCount                     = 10
		disputesPerBatch                   = 2
		acceptableRuntimeVotesQueriesCount = 1
	)

	input := NewTestDisputes(validatorCount)

	// active, concluded onchain
	thirdIdx, thirdVotes := input.addUnconfirmedDisputesConcludedOnchain(t, disputesPerBatch)
	fmt.Println(thirdIdx, thirdVotes)

	// active unconcluded onchain
	firstIdx, firstVotes := input.addUnconfirmedDisputesUnconcludedOnchain(t, disputesPerBatch)
	fmt.Println(firstIdx, firstVotes)

	// concluded disputes unknown onchain
	fifthIdx, fifthVotes := input.addConcludedDisputesUnknownOnchain(t, disputesPerBatch)
	fmt.Println(fifthIdx, fifthVotes)

	// concluded disputes known onchain - these should be ignored
	_, _ = input.addConcludedDisputesKnownOnchain(t, disputesPerBatch)

	// confirmed disputes unknown onchain
	secondIdx, secondVotes := input.addConfirmedDisputesUnkonwnOnChain(t, disputesPerBatch)
	fmt.Println(secondIdx, secondVotes)

	voteQueries := 0

	overseerCh := make(chan any, 1)

	lf := newLeaf()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mockOverseer(overseerCh, input, &voteQueries)
	}()

	mockRT := mockRuntime(t, input)
	result := SelectDisputes(overseerCh, mockRT, lf,
		MaxDisputeVotesForwardedToRuntimeTest, VotesSelectionBatchSizeTest)
	close(overseerCh)
	wg.Wait()

	require.NotEmpty(t, result)
	require.Len(t, result, 4*disputesPerBatch)

	// Naive checks that the result is partitioned correctly
	fst_batch, rst := split(
		result,
		func(d parachaintypes.DisputeStatementSet) bool { return d.Session == firstIdx },
	)
	require.Len(t, fst_batch, disputesPerBatch)
	fmt.Println(accStatements(fst_batch))

	snd_batch, rst := split(
		rst,
		func(d parachaintypes.DisputeStatementSet) bool { return d.Session == secondIdx },
	)
	require.Len(t, snd_batch, disputesPerBatch)
	fmt.Println(accStatements(snd_batch))

	trd_batch, rst := split(
		rst,
		func(d parachaintypes.DisputeStatementSet) bool { return d.Session == thirdIdx },
	)
	require.Len(t, trd_batch, disputesPerBatch)
	fmt.Println(accStatements(trd_batch))

	fifth_batch, rst := split(
		rst,
		func(d parachaintypes.DisputeStatementSet) bool { return d.Session == fifthIdx },
	)
	require.Len(t, fifth_batch, disputesPerBatch)
	fmt.Println(accStatements(fifth_batch))

	// Ensure there are no more disputes - fourth_batch should be dropped
	require.Empty(t, rst)

	require.Equal(t, accStatements(fst_batch), firstVotes)
	require.Equal(t, accStatements(snd_batch), secondVotes)
	require.Equal(t, accStatements(trd_batch), thirdVotes)
	require.Equal(t, accStatements(fifth_batch), fifthVotes)

	require.LessOrEqual(t, voteQueries, acceptableRuntimeVotesQueriesCount)
}

func TestManyBatches(t *testing.T) {
	const (
		validatorCount                     = 10
		disputesPerPartition               = 10
		acceptableRuntimeVotesQueriesCount = 4 // ~4 queries since 40 disputes / 11 batch size
	)

	input := NewTestDisputes(validatorCount)

	// active which can conclude onchain
	input.addUnconfirmedDisputesConcludedOnchain(t, disputesPerPartition)

	// active which can't conclude onchain
	input.addUnconfirmedDisputesUnconcludedOnchain(t, disputesPerPartition)

	// concluded disputes unknown onchain
	input.addConcludedDisputesUnknownOnchain(t, disputesPerPartition)

	// concluded disputes known onchain
	input.addConcludedDisputesKnownOnchain(t, disputesPerPartition)

	// confirmed disputes unknown onchain
	input.addConfirmedDisputesUnkonwnOnChain(t, disputesPerPartition)

	voteQueries := 0
	overseerCh := make(chan any, 1)

	lf := newLeaf()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mockOverseer(overseerCh, input, &voteQueries)
	}()

	mockRT := mockRuntime(t, input)
	result := SelectDisputes(overseerCh, mockRT, lf,
		MaxDisputeVotesForwardedToRuntimeTest, VotesSelectionBatchSizeTest)
	close(overseerCh)
	wg.Wait()

	voteCount := accStatements(result)
	require.LessOrEqual(t, voteCount, MaxDisputeVotesForwardedToRuntimeTest)
	require.LessOrEqual(t,
		MaxDisputeVotesForwardedToRuntimeTest-validatorCount,
		voteCount)

	require.LessOrEqual(t, voteQueries, acceptableRuntimeVotesQueriesCount)
}

func TestVotesAboveLimit(t *testing.T) {
	const (
		validatorCount                     = 10
		disputesPerPartition               = 50
		acceptableRuntimeVotesQueriesCount = 4
	)

	input := NewTestDisputes(validatorCount)

	// active which can conclude onchain
	_, secondVotes := input.addUnconfirmedDisputesConcludedOnchain(t, disputesPerPartition)
	// active which can't conclude onchain
	_, firstVotes := input.addUnconfirmedDisputesUnconcludedOnchain(t, disputesPerPartition)
	// concluded disputes unknown onchain
	_, thirdVotes := input.addConcludedDisputesUnknownOnchain(t, disputesPerPartition)

	totalVotes := firstVotes + secondVotes + thirdVotes
	require.Greater(t, totalVotes, 3*MaxDisputeVotesForwardedToRuntimeTest)

	voteQueries := 0
	overseerCh := make(chan any, 1)

	lf := newLeaf()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mockOverseer(overseerCh, input, &voteQueries)
	}()

	mockRT := mockRuntime(t, input)

	result := SelectDisputes(overseerCh, mockRT, lf,
		MaxDisputeVotesForwardedToRuntimeTest, VotesSelectionBatchSizeTest)
	close(overseerCh)
	wg.Wait()

	require.NotEmpty(t, result)

	// Accumulate the total vote count
	voteCount := accStatements(result)
	require.LessOrEqual(t, voteCount, MaxDisputeVotesForwardedToRuntimeTest)
	require.LessOrEqual(t,
		MaxDisputeVotesForwardedToRuntimeTest-validatorCount,
		voteCount)

	require.LessOrEqual(t, voteQueries, acceptableRuntimeVotesQueriesCount)
}

func TestUnconfirmedAreHandledCorrectly(t *testing.T) {
	const (
		validatorCount       = 10
		disputesPerPartition = 50
	)

	input := NewTestDisputes(validatorCount)

	// Add unconfirmed known onchain -> this should be pushed
	pushedIdx, _ := input.addUnconfirmedDisputesKnownOnchain(t, disputesPerPartition)
	input.addUnconfirmedDisputesUnknownOnchain(t, disputesPerPartition)

	voteQueries := 0
	overseerCh := make(chan any, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		mockOverseer(overseerCh, input, &voteQueries)
	}()

	lf := newLeaf()
	mockRT := mockRuntime(t, input)

	result := SelectDisputes(overseerCh, mockRT, lf,
		MaxDisputeVotesForwardedToRuntimeTest, VotesSelectionBatchSizeTest)

	close(overseerCh)
	wg.Wait()

	require.Equal(t, len(result), disputesPerPartition)
	for _, d := range result {
		require.Equal(t, d.Session, pushedIdx)
	}
}

func split(
	input []parachaintypes.DisputeStatementSet,
	p func(parachaintypes.DisputeStatementSet) bool,
) ([]parachaintypes.DisputeStatementSet, []parachaintypes.DisputeStatementSet) {
	var left, right []parachaintypes.DisputeStatementSet
	for _, d := range input {
		if p(d) {
			left = append(left, d)
		} else {
			right = append(right, d)
		}
	}

	return left, right
}

func accStatements(input []parachaintypes.DisputeStatementSet) int {
	var acc int
	for _, d := range input {
		acc += len(d.Statements)
	}
	return acc
}

func bitVector(t *testing.T, bits []bool) parachaintypes.BitVec {
	t.Helper()

	bv, err := parachaintypes.NewBitVec(bits)
	require.NoError(t, err)
	return bv
}
