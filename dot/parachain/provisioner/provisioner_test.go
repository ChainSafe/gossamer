package provisioner

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	disputemessages "github.com/ChainSafe/gossamer/dot/parachain/disputes-coordinator/messages"
	prospectiveparachain "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const blockUnderProduction = parachaintypes.BlockNumber(128)
const mockGroupSize = 5
const errMockOverseerTimeout = "mock overseer for selectCandidates failed"

func TestProcessActiveLeavesUpdateSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		perRP              map[common.Hash]*perRelayParent // Initial state of perRelayParent
		update             parachaintypes.ActiveLeavesUpdateSignal
		expectedRemaining  []common.Hash // Expected hashes remaining in perRelayParent
		expectInherentHash *common.Hash  // Expected hash to be sent on availableInherent channel
	}{
		{
			name: "deactivate_single_leaf",
			perRP: map[common.Hash]*perRelayParent{
				{1}: {leaf: &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}}},
			},
			update: parachaintypes.ActiveLeavesUpdateSignal{
				Deactivated: []common.Hash{{1}},
			},
			expectedRemaining: nil,
		},
		{
			name: "activate_new_leaf",
			perRP: map[common.Hash]*perRelayParent{
				{1}: {leaf: &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}}},
			},
			update: parachaintypes.ActiveLeavesUpdateSignal{
				Activated: &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
			},
			expectedRemaining:  []common.Hash{{1}},
			expectInherentHash: &common.Hash{1},
		},
		{
			name: "activate_and_deactivate",
			perRP: map[common.Hash]*perRelayParent{
				{1}: {leaf: &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}}},
				{2}: {leaf: &parachaintypes.ActivatedLeaf{Hash: common.Hash{2}}},
			},
			update: parachaintypes.ActiveLeavesUpdateSignal{
				Activated:   &parachaintypes.ActivatedLeaf{Hash: common.Hash{3}},
				Deactivated: []common.Hash{{1}},
			},
			expectedRemaining:  []common.Hash{{2}, {3}},
			expectInherentHash: &common.Hash{3},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := New(nil, nil)
			p.perRelayParent = tc.perRP

			err := p.ProcessActiveLeavesUpdateSignal(tc.update)
			require.NoError(t, err)

			// Verify remaining items in perRelayParent
			remainingHashes := make([]common.Hash, 0)
			for hash := range p.perRelayParent {
				remainingHashes = append(remainingHashes, hash)
			}
			require.ElementsMatch(t, tc.expectedRemaining, remainingHashes)

			// If we expect an inherent hash, verify it's sent on the channel
			if tc.expectInherentHash != nil {
				select {
				case hash := <-p.availableInherent:
					require.Equal(t, *tc.expectInherentHash, hash)
				case <-time.After(inherentPreProposeTimeout + 100*time.Millisecond):
					t.Fatal("timeout waiting for inherent hash")
				}
			} else {
				// Verify no inherent hash is sent when not expected
				select {
				case hash := <-p.availableInherent:
					t.Fatalf("unexpected inherent hash received: %v", hash)
				case <-time.After(inherentPreProposeTimeout + 100*time.Millisecond):
					// This is expected when no inherent hash should be sent
				}
			}
		})
	}
}

func TestProcessProvisionableData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		provisionableData   provisionermessages.ProvisionableData
		expectedBitfields   []parachaintypes.CheckedSignedAvailabilityBitfield
		shouldStoreBitfield bool
	}{
		{
			name: "bitfield_relay_parent_exists",
			provisionableData: provisionermessages.ProvisionableData{
				RelayParent: common.Hash{1},
				Data: provisionermessages.ProvisionableDataBitfield{
					Bitfield: parachaintypes.CheckedSignedAvailabilityBitfield{},
				},
			},
			expectedBitfields:   []parachaintypes.CheckedSignedAvailabilityBitfield{{}},
			shouldStoreBitfield: true,
		},
		{
			name: "bitfield_relay_parent_not_exists",
			provisionableData: provisionermessages.ProvisionableData{
				RelayParent: common.Hash{2}, // Different hash
				Data: provisionermessages.ProvisionableDataBitfield{
					Bitfield: parachaintypes.CheckedSignedAvailabilityBitfield{},
				},
			},
			expectedBitfields:   []parachaintypes.CheckedSignedAvailabilityBitfield{},
			shouldStoreBitfield: false,
		},
		{
			name: "misbehaviour_report_no_effect",
			provisionableData: provisionermessages.ProvisionableData{
				RelayParent: common.Hash{1},
				Data: provisionermessages.ProvisionableDataMisbehaviorReport{
					ValidatorIndex: 0,
				},
			},
			expectedBitfields:   []parachaintypes.CheckedSignedAvailabilityBitfield{},
			shouldStoreBitfield: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := New(nil, nil)
			p.perRelayParent = dummyPerRelayParentState()

			p.processProvisionableData(tc.provisionableData)

			if tc.shouldStoreBitfield {
				state := p.perRelayParent[tc.provisionableData.RelayParent]
				require.Equal(t, tc.expectedBitfields, state.signedBitfields)
			} else {
				// For misbehaviour reports or non-existent relay parents, verify bitfields unchanged
				state := p.perRelayParent[common.Hash{1}]
				require.Equal(t, tc.expectedBitfields, state.signedBitfields)
			}
		})
	}
}

func TestSelectAvailabilityBitfield(t *testing.T) {
	t.Parallel()
	t.Run("not_more_than_one_per_validator", func(t *testing.T) {
		t.Parallel()
		bitvec := bitVector(t, []bool{true, true})
		cores := []parachaintypes.CoreState{
			coreState(t, occupiedCore(t, parachaintypes.ParaID(0))),
			coreState(t, occupiedCore(t, parachaintypes.ParaID(1))),
		}

		ks := keystore.NewBasicKeystore("test", crypto.Sr25519Type)

		bitfields := []parachaintypes.CheckedSignedAvailabilityBitfield{
			signedBitfield(t, ks, bitvec, parachaintypes.ValidatorIndex(0)),
			signedBitfield(t, ks, bitvec, parachaintypes.ValidatorIndex(1)),
			signedBitfield(t, ks, bitvec, parachaintypes.ValidatorIndex(1)),
		}

		selectedBitfield, err := selectAvailabilityBitfields(cores, bitfields)
		require.NoError(t, err)

		require.Len(t, selectedBitfield, 2)
		require.Equal(t, parachaintypes.ValidatorIndex(0), selectedBitfield[0].ValidatorIndex)

		// we don't know which of the bitfields will be selected
		if !(selectedBitfield[1].ValidatorIndex == bitfields[1].ValidatorIndex ||
			selectedBitfield[1].ValidatorIndex == bitfields[2].ValidatorIndex) {
			t.Errorf("selectedBitfields[1] (%v) should equal either bitfields[1] (%v) or bitfields[2] (%v)",
				selectedBitfield[1], bitfields[1], bitfields[2])
		}
	})

	t.Run("each_corresponds_to_an_occupied_core", func(t *testing.T) {
		t.Parallel()
		ks := keystore.NewBasicKeystore("test", crypto.Sr25519Type)

		// invalid: bit on free core
		bitvec0 := bitVector(t, []bool{true, false, false})

		// invalid: bit on scheduled core
		bitvec1 := bitVector(t, []bool{false, true, false})

		// valid: bit on occupied core.
		bitvec2 := bitVector(t, []bool{false, false, true})

		freeCore := parachaintypes.CoreState{}
		err := freeCore.SetValue(parachaintypes.Free{})
		require.NoError(t, err)

		scheduledCore := parachaintypes.CoreState{}
		err = scheduledCore.SetValue(parachaintypes.ScheduledCore{})
		require.NoError(t, err)

		occupiedCore := coreState(t, occupiedCore(t, parachaintypes.ParaID(2)))

		cores := []parachaintypes.CoreState{freeCore, scheduledCore, occupiedCore}

		bitfields := []parachaintypes.CheckedSignedAvailabilityBitfield{
			signedBitfield(t, ks, bitvec0, parachaintypes.ValidatorIndex(0)),
			signedBitfield(t, ks, bitvec1, parachaintypes.ValidatorIndex(1)),
			signedBitfield(t, ks, bitvec2, parachaintypes.ValidatorIndex(2)),
		}

		selectedBitfield, err := selectAvailabilityBitfields(cores, bitfields)
		require.NoError(t, err)

		require.Len(t, selectedBitfield, 1)
		require.Equal(t, bitvec2, selectedBitfield[0].Payload)
	})

	t.Run("more_set_bits_win_conflicts", func(t *testing.T) {
		t.Parallel()
		ks := keystore.NewBasicKeystore("test", crypto.Sr25519Type)

		bitvec := bitVector(t, []bool{true, false})

		bitvec1 := bitVector(t, []bool{true, true})

		cores := []parachaintypes.CoreState{
			coreState(t, occupiedCore(t, parachaintypes.ParaID(0))),
			coreState(t, occupiedCore(t, parachaintypes.ParaID(1))),
		}

		// Create bitfields signed by the same validator
		bitfields := []parachaintypes.CheckedSignedAvailabilityBitfield{
			signedBitfield(t, ks, bitvec, parachaintypes.ValidatorIndex(1)),
			signedBitfield(t, ks, bitvec1, parachaintypes.ValidatorIndex(1)),
		}

		selectedBitfield, err := selectAvailabilityBitfields(cores, bitfields)
		require.NoError(t, err)

		// Verify that the bitfield with more set bits wins
		require.Len(t, selectedBitfield, 1)
		require.Equal(t, bitvec1, selectedBitfield[0].Payload)
	})

	t.Run("complex_bitfield_selection", func(t *testing.T) {
		t.Parallel()
		ks := keystore.NewBasicKeystore("test", crypto.Sr25519Type)

		cores := []parachaintypes.CoreState{
			coreState(t, occupiedCore(t, parachaintypes.ParaID(0))),
			coreState(t, occupiedCore(t, parachaintypes.ParaID(1))),
			coreState(t, occupiedCore(t, parachaintypes.ParaID(2))),
			coreState(t, occupiedCore(t, parachaintypes.ParaID(3))),
		}

		// Create bitvectors with different patterns
		bitvec0 := bitVector(t, []bool{true, false, true, false})  // 1,0,1,0
		bitvec1 := bitVector(t, []bool{false, true, false, false}) // 0,1,0,0
		bitvec2 := bitVector(t, []bool{false, false, true, false}) // 0,0,1,0
		bitvec3 := bitVector(t, []bool{true, true, true, true})    // 1,1,1,1

		// Create bitfields out of order but they should be selected in order
		// The better bitfield for validator 3 should be selected
		bitfields := []parachaintypes.CheckedSignedAvailabilityBitfield{
			signedBitfield(t, ks, bitvec2, parachaintypes.ValidatorIndex(3)),
			signedBitfield(t, ks, bitvec3, parachaintypes.ValidatorIndex(3)),
			signedBitfield(t, ks, bitvec0, parachaintypes.ValidatorIndex(0)),
			signedBitfield(t, ks, bitvec2, parachaintypes.ValidatorIndex(2)),
			signedBitfield(t, ks, bitvec1, parachaintypes.ValidatorIndex(1)),
		}

		selectedBitfield, err := selectAvailabilityBitfields(cores, bitfields)
		require.NoError(t, err)

		// Verify the selected bitfields
		require.Len(t, selectedBitfield, 4)
		require.Equal(t, bitvec0, selectedBitfield[0].Payload)
		require.Equal(t, bitvec1, selectedBitfield[1].Payload)
		require.Equal(t, bitvec2, selectedBitfield[2].Payload)
		require.Equal(t, bitvec3, selectedBitfield[3].Payload)
	})
}

func TestSelectCandidates(t *testing.T) {
	t.Parallel()

	t.Run("can_succeed_old", func(t *testing.T) {
		t.Parallel()
		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, nil, nil, overseerChan)
		}()

		candidates, err := provisioner.selectCandidates(
			[]parachaintypes.CoreState{},
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case overseerErr := <-errCh:
			if overseerErr != nil {
				t.Fatal(overseerErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		require.Empty(t, candidates)
	})

	t.Run("selects_max_one_code_upgrade_one_core_per_para", func(t *testing.T) {
		t.Parallel()
		mockCores := mockAvailabilityCoresOnePerPara(t)
		emptyHash, err := parachaintypes.PersistedValidationData{}.Hash()
		require.NoError(t, err)

		cores := []int{1, 4, 7, 8, 10, 12}
		coresWithCode := []int{1, 4, 8}

		// We can't be sure which one code upgrade the provisioner will pick. We can only assert
		// that it only picks one. These are the possible cores for which the provisioner will
		// supply candidates.
		// There are multiple possibilities depending on which code upgrade it
		// chooses.
		possibleExpectedCores := [][]int{
			{1, 7, 10, 12},
			{4, 7, 10, 12},
			{7, 8, 10, 12},
		}

		// Create committed receipts for each core
		committedReceipts := make([]*parachaintypes.CommittedCandidateReceiptV2, len(mockCores)+1)
		for i := 0; i <= len(mockCores); i++ {
			descriptor := dummyCandidateDescriptorV2(common.Hash{})
			descriptor.ParaID = parachaintypes.ParaID(i)
			descriptor.PersistedValidationDataHash = emptyHash
			descriptor.PovHash = hashFromU64BE(uint64(i))

			// Create commitments with validation code for specific cores
			var validationCode *parachaintypes.ValidationCode
			if slices.Contains(coresWithCode, i) {
				validationCode = new(parachaintypes.ValidationCode)
			}

			committedReceipts[i] = &parachaintypes.CommittedCandidateReceiptV2{
				Descriptor: *descriptor,
				Commitments: parachaintypes.CandidateCommitments{
					NewValidationCode: validationCode,
				},
			}
		}

		candidates := make([]parachaintypes.CandidateReceiptV2, len(committedReceipts))
		for i, receipt := range committedReceipts {
			candidates[i] = receipt.ToPlain()
		}

		backedCandidates := make([]parachaintypes.BackedCandidate, len(committedReceipts))
		for i, receipt := range committedReceipts {
			backed, err := parachaintypes.NewBackedCandidate(
				*receipt, nil, make([]bool, mockGroupSize), &parachaintypes.CoreIndex{Index: 0})
			require.NoError(t, err)
			backedCandidates[i] = *backed
		}

		expectedBacked := make([]parachaintypes.BackedCandidate, 0, len(cores))
		for _, core := range cores {
			expectedBacked = append(expectedBacked, backedCandidates[core])
		}

		// Create possible expected candidates based on the cores that could be selected
		expectedBackedFiltered := make([][]parachaintypes.CandidateReceiptV2, len(possibleExpectedCores))
		for i, indices := range possibleExpectedCores {
			expectedBackedFiltered[i] = make([]parachaintypes.CandidateReceiptV2, len(indices))
			for j, idx := range indices {
				expectedBackedFiltered[i][j] = candidates[idx]
			}
		}

		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, expectedBacked, nil, overseerChan)
		}()

		selectedCandidates, err := provisioner.selectCandidates(
			mockCores,
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case overseerErr := <-errCh:
			if overseerErr != nil {
				t.Fatal(overseerErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		require.Len(t, selectedCandidates, 4)
		require.True(t, matchesAnyExpectedBacked(selectedCandidates, expectedBackedFiltered, false))
	})

	t.Run("selects_max_one_code_upgrade_multiple_cores_per_para", func(t *testing.T) {
		t.Parallel()
		mockCores := []parachaintypes.CoreState{
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(1)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(2)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(2)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(2)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(3)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(3)}),
			coreState(t, parachaintypes.ScheduledCore{ParaID: parachaintypes.ParaID(3)}),
		}
		coresWithCode := []int{0, 2, 4, 5}
		possibleExpectedCores := [][]int{{0, 1}, {1, 2, 3}, {4, 1}}
		emptyHash, err := parachaintypes.PersistedValidationData{}.Hash()
		require.NoError(t, err)

		receipts := make([]parachaintypes.CommittedCandidateReceiptV2, len(mockCores))
		for i := 0; i < len(mockCores); i++ {
			descriptor := dummyCandidateDescriptorV2(common.Hash{})
			core, err := mockCores[i].Value()
			require.NoError(t, err)

			scheduledCore, ok := core.(parachaintypes.ScheduledCore)
			require.True(t, ok)

			descriptor.ParaID = scheduledCore.ParaID
			descriptor.PersistedValidationDataHash = emptyHash
			descriptor.PovHash = hashFromU64BE(uint64(i))

			var validationCode *parachaintypes.ValidationCode
			if slices.Contains(coresWithCode, i) {
				validationCode = new(parachaintypes.ValidationCode)
			}

			receipts[i] = parachaintypes.CommittedCandidateReceiptV2{
				Descriptor: *descriptor,
				Commitments: parachaintypes.CandidateCommitments{
					NewValidationCode: validationCode,
				},
			}
		}

		candidates := make([]parachaintypes.CandidateReceiptV2, len(receipts))
		for i, receipt := range receipts {
			candidates[i] = receipt.ToPlain()
		}

		backedCandidates := make([]parachaintypes.BackedCandidate, len(candidates))
		for i, receipt := range receipts {
			backed, err := parachaintypes.NewBackedCandidate(
				receipt, nil, make([]bool, mockGroupSize), &parachaintypes.CoreIndex{Index: 0})
			require.NoError(t, err)
			backedCandidates[i] = *backed
		}

		expectedBacked := make([]parachaintypes.BackedCandidate, 0, len(mockCores))
		for i := 0; i < len(mockCores); i++ {
			expectedBacked = append(expectedBacked, backedCandidates[i])
		}

		// Create possible expected candidates based on the cores that could be selected
		expectedBackedFiltered := make([][]parachaintypes.CandidateReceiptV2, len(possibleExpectedCores))
		for i, indices := range possibleExpectedCores {
			expectedBackedFiltered[i] = make([]parachaintypes.CandidateReceiptV2, len(indices))
			for j, idx := range indices {
				expectedBackedFiltered[i][j] = candidates[idx]
			}
		}

		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, expectedBacked, nil, overseerChan)
		}()

		selectedCandidates, err := provisioner.selectCandidates(
			mockCores,
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case overseerErr := <-errCh:
			if overseerErr != nil {
				t.Fatal(overseerErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		require.True(t, matchesAnyExpectedBacked(selectedCandidates, expectedBackedFiltered, true))
	})

	t.Run("one_core_per_para", func(t *testing.T) {
		t.Parallel()
		mockCores := mockAvailabilityCoresOnePerPara(t)
		expectedCandidates := []int{1, 4, 7, 8, 10, 12}
		candidates, expectedBackedCandidates := makeCandidates(t, 1+len(mockCores), expectedCandidates)

		requiredAncestors := make(map[string]prospectiveparachain.Ancestors)
		requiredAncestors[candidates[4].String()] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(41)}: struct{}{},
		}
		requiredAncestors[candidates[8].String()] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(81)}: struct{}{},
		}

		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, expectedBackedCandidates, requiredAncestors, overseerChan)
		}()

		selectedCandidates, err := provisioner.selectCandidates(
			mockCores,
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case overseerErr := <-errCh:
			if overseerErr != nil {
				t.Fatal(overseerErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		require.Equal(t, len(expectedCandidates), len(selectedCandidates))
		for _, selected := range selectedCandidates {
			found := false
			for _, expected := range expectedBackedCandidates {
				if correspondsTo(selected.Candidate, expected.Candidate.ToPlain()) {
					found = true
					break
				}
			}
			require.True(t, found, "Failed to find matching candidate")
		}
	})

	t.Run("multiple_cores_per_para_elastic_scaling_mvp", func(t *testing.T) {
		t.Parallel()
		mockCores := mockAvailabilityCoresMultiplePerPara(t)
		expectedCandidates := []int{1, 4, 7, 8, 10, 12, 12, 12, 12, 12, 13, 13, 13, 14, 14, 14, 15, 15}

		candidates, expectedBackedCandidates := makeCandidates(t, len(mockCores), expectedCandidates)

		requiredAncestors := make(map[string]prospectiveparachain.Ancestors)
		requiredAncestors[candidates[4].String()] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(41)}: struct{}{},
		}
		requiredAncestors[candidates[8].String()] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(81)}: struct{}{},
		}

		hashes12 := []string{candidates[12].String(), candidates[12].String(), candidates[12].String()}
		k12 := strings.Join(hashes12, `,`)
		requiredAncestors[k12] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(121)}: struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(122)}: struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(123)}: struct{}{},
		}

		hashes13 := []string{candidates[13].String(), candidates[13].String(), candidates[13].String()}
		k13 := strings.Join(hashes13, `,`)
		requiredAncestors[k13] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(131)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(132)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(133)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(134)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(135)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(136)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(137)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(138)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(139)}:  struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(1398)}: struct{}{},
		}

		hashes15 := []string{candidates[15].String(), candidates[15].String()}
		k15 := strings.Join(hashes15, `,`)
		requiredAncestors[k15] = prospectiveparachain.Ancestors{
			parachaintypes.CandidateHash{Value: hashFromU64BE(151)}: struct{}{},
			parachaintypes.CandidateHash{Value: hashFromU64BE(152)}: struct{}{},
		}

		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, expectedBackedCandidates, requiredAncestors, overseerChan)
		}()

		selectedCandidates, err := provisioner.selectCandidates(
			mockCores,
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case mockErr := <-errCh:
			if mockErr != nil {
				t.Fatal(mockErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		require.Equal(t, len(expectedBackedCandidates), len(selectedCandidates))

		for _, c := range selectedCandidates {
			found := false
			for _, expected := range expectedBackedCandidates {
				if correspondsTo(c.Candidate, expected.Candidate.ToPlain()) {
					found = true
					break
				}
			}
			require.True(t, found, fmt.Sprintf("Failed to find candidate: %+v", c))
		}
	})

	t.Run("request_receipts_based_on_relay_parent", func(t *testing.T) {
		t.Parallel()
		mockCores := mockAvailabilityCoresOnePerPara(t)
		emptyHash, err := parachaintypes.PersistedValidationData{}.Hash()
		require.NoError(t, err)

		candidateTemplate := dummyCandidateReceiptV2(common.Hash{})
		candidateTemplate.Descriptor.PersistedValidationDataHash = emptyHash

		candidates := make([]parachaintypes.CandidateReceiptV2, len(mockCores)+1)
		for i := 0; i <= len(mockCores); i++ {
			candidate := *candidateTemplate
			candidate.Descriptor.ParaID = parachaintypes.ParaID(i)
			candidate.Descriptor.RelayParent = hashFromU64BE(uint64(i))
			candidates[i] = candidate
		}

		// Expected indices based on mock availability cores
		expectedIndices := []int{1, 4, 7, 8, 10, 12}
		expectedCandidates := make([]parachaintypes.CandidateReceiptV2, len(expectedIndices))
		for i, idx := range expectedIndices {
			expectedCandidates[i] = candidates[idx]
		}

		expectedBacked := make([]parachaintypes.BackedCandidate, len(expectedCandidates))
		for i, candidate := range expectedCandidates {
			committed := parachaintypes.CommittedCandidateReceiptV2{
				Descriptor:  candidate.Descriptor,
				Commitments: parachaintypes.CandidateCommitments{}, // Default empty commitments
			}

			backedCandidate, err := parachaintypes.NewBackedCandidate(
				committed,
				nil,
				make([]bool, mockGroupSize),
				&parachaintypes.CoreIndex{Index: 0},
			)
			require.NoError(t, err)
			expectedBacked[i] = *backedCandidate
		}

		overseerChan := make(chan any)
		ctrl := gomock.NewController(t)
		bs := NewMockBlockState(ctrl)
		provisioner := New(overseerChan, bs)

		errCh := make(chan error, 1)
		go func() {
			errCh <- mockOverseerForSelectCandidates(t, expectedBacked, nil, overseerChan)
		}()

		selectedCandidates, err := provisioner.selectCandidates(
			mockCores,
			[]parachaintypes.CheckedSignedAvailabilityBitfield{},
			&parachaintypes.ActivatedLeaf{Hash: common.Hash{}, Number: uint32(blockUnderProduction - 1)},
		)

		close(overseerChan)
		select {
		case overseerErr := <-errCh:
			if overseerErr != nil {
				t.Fatal(overseerErr)
			}
		case <-time.After(time.Second * 6):
			t.Fatal(errMockOverseerTimeout)
		}

		require.NoError(t, err)
		// Verify each selected candidate corresponds to one of the expected candidates
		for _, selected := range selectedCandidates {
			found := false
			for _, expected := range expectedCandidates {
				if correspondsTo(selected.Candidate, expected) {
					found = true
					break
				}
			}
			require.True(t, found, fmt.Sprintf("Failed to find candidate: %+v", selected))
		}
	})
}

func TestSendInherentData(t *testing.T) {
	t.Parallel()

	bitVec, err := parachaintypes.NewBitVec([]bool{false})
	require.NoError(t, err)

	validationData := &parachaintypes.PersistedValidationData{
		RelayParentNumber:      uint32(1),
		RelayParentStorageRoot: common.Hash{0x03},
		MaxPovSize:             uint32(1024),
	}
	validationHash, err := validationData.Hash()
	require.NoError(t, err)

	occupiedCore := parachaintypes.OccupiedCore{
		NextUpOnAvailable: nil,
		NextUpOnTimeOut:   nil,
		Availability:      bitVec,
		GroupResponsible:  parachaintypes.GroupIndex(0),
		CandidateHash:     common.Hash{},
		CandidateDescriptor: parachaintypes.CandidateDescriptorV2{
			ParaID:                      parachaintypes.ParaID(1),
			RelayParent:                 common.Hash{0x03},
			PersistedValidationDataHash: validationHash,
			PovHash:                     common.Hash{},
		},
	}

	testCases := []struct {
		name            string
		leaf            *parachaintypes.ActivatedLeaf
		signedBitfields []parachaintypes.CheckedSignedAvailabilityBitfield
		mockCores       []parachaintypes.CoreState
		expectError     bool
		setupMocks      func(*gomock.Controller) (BlockState, runtime.Instance)
	}{
		{
			name: "successful_inherent_data",
			leaf: &parachaintypes.ActivatedLeaf{
				Hash:   common.Hash{0x01},
				Number: uint32(blockUnderProduction), // Convert to uint32
			},
			signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{
				{
					ValidatorIndex: parachaintypes.ValidatorIndex(0),
					Signature:      [64]byte{0x02},             // Using 64-byte array for signature
					Payload:        bitVector(t, []bool{true}), // Initialize BitVec with bool array - one bit for one core
				},
			},
			mockCores: []parachaintypes.CoreState{
				coreState(t, occupiedCore),
			},
			expectError: false,
			setupMocks: func(ctrl *gomock.Controller) (BlockState, runtime.Instance) {
				mockBlockState := NewMockBlockState(ctrl)
				mockRuntime := NewMockInstance(ctrl)

				mockBlockState.EXPECT().
					GetRuntime(gomock.Any()).
					Return(mockRuntime, nil).AnyTimes()

				mockRuntime.EXPECT().
					ParachainHostAvailabilityCores().
					Return([]parachaintypes.CoreState{
						coreState(t, occupiedCore),
					}, nil)

				mockRuntime.EXPECT().
					ParachainHostDisputes().
					Return(map[parachaintypes.DisputeKey]parachaintypes.DisputeState{}, nil)

				return mockBlockState, mockRuntime
			},
		},
		{
			name: "get_runtime_error",
			leaf: &parachaintypes.ActivatedLeaf{
				Hash:   common.Hash{0x02},
				Number: uint32(blockUnderProduction),
			},
			signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
			mockCores:       []parachaintypes.CoreState{},
			expectError:     true,
			setupMocks: func(ctrl *gomock.Controller) (BlockState, runtime.Instance) {
				mockBlockState := NewMockBlockState(ctrl)
				mockRuntime := NewMockInstance(ctrl)

				mockBlockState.EXPECT().
					GetRuntime(gomock.Any()).
					Return(nil, fmt.Errorf("runtime error"))

				return mockBlockState, mockRuntime
			},
		},
		{
			name: "cores_error",
			leaf: &parachaintypes.ActivatedLeaf{
				Hash:   common.Hash{0x03},
				Number: uint32(blockUnderProduction),
			},
			signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
			mockCores:       []parachaintypes.CoreState{},
			expectError:     true,
			setupMocks: func(ctrl *gomock.Controller) (BlockState, runtime.Instance) {
				mockBlockState := NewMockBlockState(ctrl)
				mockRuntime := NewMockInstance(ctrl)

				mockBlockState.EXPECT().
					GetRuntime(gomock.Any()).
					Return(mockRuntime, nil).AnyTimes()

				mockRuntime.EXPECT().
					ParachainHostAvailabilityCores().
					Return(nil, fmt.Errorf("cores error"))

				return mockBlockState, mockRuntime
			},
		},
		{
			name: "empty_bitfields",
			leaf: &parachaintypes.ActivatedLeaf{
				Hash:   common.Hash{0x04},
				Number: uint32(blockUnderProduction),
			},
			signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
			mockCores: []parachaintypes.CoreState{
				coreState(t, parachaintypes.Free{}),
			},
			expectError: false,
			setupMocks: func(ctrl *gomock.Controller) (BlockState, runtime.Instance) {
				mockBlockState := NewMockBlockState(ctrl)
				mockRuntime := NewMockInstance(ctrl)

				mockBlockState.EXPECT().
					GetRuntime(gomock.Any()).
					Return(mockRuntime, nil).AnyTimes()

				mockRuntime.EXPECT().
					ParachainHostAvailabilityCores().
					Return([]parachaintypes.CoreState{
						coreState(t, parachaintypes.Free{}),
					}, nil)

				mockRuntime.EXPECT().
					ParachainHostDisputes().
					Return(map[parachaintypes.DisputeKey]parachaintypes.DisputeState{}, nil)

				return mockBlockState, mockRuntime
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockBlockState, _ := tc.setupMocks(ctrl)

			// Create channels for test coordination
			overseerCh := make(chan any)
			responseSender := make(chan provisionermessages.ProvisionerInherentData)
			done := make(chan struct{})

			p := &Provisioner{
				blockState:          mockBlockState,
				subSystemToOverseer: overseerCh,
			}

			inherentDataErr := make(chan error, 1)

			// Handle overseer messages
			go func() {
				defer close(overseerCh)
				for {
					select {
					case <-done:
						return
					case msg := <-overseerCh:
						switch m := msg.(type) {
						case disputemessages.GetRecentDisputes:
							if m.Response != nil {
								m.Response <- []disputemessages.RecentDispute{}
								close(m.Response)
							}
						case backing.GetBackableCandidatesMessage:
							if m.ResCh != nil {
								// Return empty map of backed candidates
								m.ResCh <- make(map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate)
							}
						case prospectiveparachain.GetBackableCandidates:
							if m.Response != nil {
								// Return empty array of candidates
								m.Response <- []*parachaintypes.CandidateHashAndRelayParent{}
							}
						}
					}
				}
			}()

			// Create a goroutine to verify inherent data
			go func() {
				defer close(done)
				defer close(inherentDataErr)
				if !tc.expectError {
					select {
					case inherentData := <-responseSender:
						var err error
						// Verify bitfield data
						if len(inherentData.Bitfield) != len(tc.signedBitfields) {
							err = fmt.Errorf("expected %d bitfields, got %d", len(tc.signedBitfields), len(inherentData.Bitfield))
						} else if len(tc.signedBitfields) > 0 {
							if !reflect.DeepEqual(inherentData.Bitfield[0].ValidatorIndex, tc.signedBitfields[0].ValidatorIndex) {
								err = fmt.Errorf("validator index mismatch: expected %v, got %v",
									tc.signedBitfields[0].ValidatorIndex, inherentData.Bitfield[0].ValidatorIndex)
							} else if !reflect.DeepEqual(inherentData.Bitfield[0].Signature, tc.signedBitfields[0].Signature) {
								err = fmt.Errorf("signature mismatch: expected %v, got %v",
									tc.signedBitfields[0].Signature, inherentData.Bitfield[0].Signature)
							} else if !reflect.DeepEqual(inherentData.Bitfield[0].Payload, tc.signedBitfields[0].Payload) {
								err = fmt.Errorf("payload mismatch: expected %v, got %v",
									tc.signedBitfields[0].Payload, inherentData.Bitfield[0].Payload)
							}
						}

						// Verify disputes field exists
						if inherentData.Disputes == nil {
							err = fmt.Errorf("disputes field is nil")
						}
						inherentDataErr <- err
					case <-time.After(2 * time.Second):
						inherentDataErr <- fmt.Errorf("timeout waiting for inherent data")
					}
				} else {
					// For error cases, just verify that no data is received
					select {
					case <-responseSender:
						inherentDataErr <- fmt.Errorf("received unexpected inherent data in error case")
					case <-time.After(100 * time.Millisecond):
						inherentDataErr <- nil // Expected timeout in error cases
					}
				}
			}()

			err := p.sendInherentData(
				tc.leaf, tc.signedBitfields, []chan provisionermessages.ProvisionerInherentData{responseSender})
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Wait for the response goroutine to complete with reduced timeout
			select {
			case <-done:
				// goroutine completed successfully
				if err, ok := <-inherentDataErr; ok {
					require.NoError(t, err)
				}
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for response goroutine to complete")
			}
		})
	}
}

func matchesAnyExpectedBacked(
	result []parachaintypes.BackedCandidate,
	expectedBackedFiltered [][]parachaintypes.CandidateReceiptV2,
	matchLen bool,
) bool {
	for _, expectedBacked := range expectedBackedFiltered {
		if matchLen && len(result) != len(expectedBacked) {
			continue
		}

		allMatch := true
		for _, resultCandidate := range result {
			// Check if this result candidate matches any candidate in the expected set
			found := false
			for _, expectedCandidate := range expectedBacked {
				if correspondsTo(resultCandidate.Candidate, expectedCandidate) {
					found = true
					break
				}
			}
			if !found {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// correspondsTo checks if two candidate receipts correspond to each other
func correspondsTo(c1 parachaintypes.CommittedCandidateReceiptV2, c2 parachaintypes.CandidateReceiptV2) bool {
	return c1.Descriptor.Equals(c2.Descriptor) && c1.Commitments.Hash() == c2.CommitmentsHash
}

func dummyPerRelayParentState() map[common.Hash]*perRelayParent {
	perRP := perRelayParent{
		leaf: &parachaintypes.ActivatedLeaf{
			Hash: common.Hash{1},
		},
		signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
	}

	return map[common.Hash]*perRelayParent{
		{1}: &perRP,
	}
}

func mockOverseerForSelectCandidates(
	t *testing.T,
	expected []parachaintypes.BackedCandidate,
	expectedAncestors map[string]prospectiveparachain.Ancestors,
	overseerCh chan any) error {
	t.Helper()

	backed := make(map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate)
	for _, candidate := range expected {
		paraID := candidate.Candidate.Descriptor.ParaID
		backed[paraID] = append(backed[paraID], &candidate)
	}

	// Sort expected candidates by para ID
	sortedExpected := make([]parachaintypes.BackedCandidate, len(expected))
	copy(sortedExpected, expected)
	sort.Slice(sortedExpected, func(i, j int) bool {
		return sortedExpected[i].Candidate.Descriptor.ParaID < sortedExpected[j].Candidate.Descriptor.ParaID
	})

	candidatesIter := make([]*parachaintypes.CandidateHashAndRelayParent, 0)
	for _, backedCandidate := range sortedExpected {
		candidatesIter = append(candidatesIter, &parachaintypes.CandidateHashAndRelayParent{
			CandidateHash:        candidateHash(t, backedCandidate.Candidate),
			CandidateRelayParent: backedCandidate.Candidate.Descriptor.RelayParent,
		})
	}

	iterIndex := 0

	for msg := range overseerCh {
		switch m := msg.(type) {
		case backing.GetBackableCandidatesMessage:
			response := make(map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate)
			for paraID, requestedCandidates := range m.Candidates {
				if candidates, ok := backed[paraID]; ok {
					numCandidates := len(requestedCandidates)
					response[paraID] = candidates[:numCandidates]
					backed[paraID] = candidates[numCandidates:]
				}
			}

			expected := make(map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent)
			for paraID, candidates := range response {
				v := make([]*parachaintypes.CandidateHashAndRelayParent, 0)
				for _, candidate := range candidates {
					v = append(v, &parachaintypes.CandidateHashAndRelayParent{
						CandidateHash:        candidateHash(t, candidate.Candidate),
						CandidateRelayParent: candidate.Candidate.Descriptor.RelayParent,
					})
				}
				expected[paraID] = v
			}

			if !reflect.DeepEqual(expected, m.Candidates) {
				return fmt.Errorf("expected candidates %v but got %v", expected, m.Candidates)
			}
			m.ResCh <- response

		case prospectiveparachain.GetBackableCandidates:
			if m.RequestedQty == 0 {
				m.Response <- nil
				continue
			}

			remaining := len(candidatesIter) - iterIndex
			count := int(m.RequestedQty)
			if count > remaining {
				count = remaining
			}

			candidates := candidatesIter[iterIndex : iterIndex+count]
			if len(candidates) != count {
				return fmt.Errorf("expected %d candidates but got %d", count, len(candidates))
			}

			if len(expectedAncestors) > 0 {
				numOfHash := len(m.Ancestors)
				if len(m.Ancestors) > len(candidates) {
					numOfHash = len(candidates)
				}

				var hashes []string
				for i := 0; i < numOfHash; i++ {
					hashes = append(hashes, candidates[i].CandidateHash.String())
				}

				candidateKeys := strings.Join(hashes, `,`)

				if ancestors, ok := expectedAncestors[candidateKeys]; ok {
					if !reflect.DeepEqual(ancestors, m.Ancestors) {
						return fmt.Errorf("expected ancestors %v but got %v", ancestors, m.Ancestors)
					}
					delete(expectedAncestors, candidateKeys)
				} else if len(m.Ancestors) > 0 {
					return fmt.Errorf("expected no ancestors but got %v", m.Ancestors)
				}
			}

			iterIndex += count
			m.Response <- candidates

		default:
			return fmt.Errorf("unexpected message type: %T", m)
		}
	}

	if iterIndex != len(candidatesIter) {
		return fmt.Errorf("expected iterIndex %d to equal len(candidatesIter) %d", iterIndex, len(candidatesIter))
	}
	if len(expectedAncestors) > 0 {
		return fmt.Errorf("not all expected ancestors were processed: %v", expectedAncestors)
	}
	return nil
}

func mockAvailabilityCoresOnePerPara(t *testing.T) []parachaintypes.CoreState {
	t.Helper()
	free := coreState(t, parachaintypes.Free{})
	c1 := coreState(t, parachaintypes.ScheduledCore{ParaID: 1})
	c2 := coreState(t, occupiedCore(t, 2))

	oc3 := occupiedCore(t, 3)
	oc3.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 3}
	c3 := coreState(t, oc3)

	oc4 := occupiedCore(t, 4)
	oc4.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 4}
	oc4.Availability.Not()
	oc4.CandidateHash = hashFromU64BE(41)
	c4 := coreState(t, oc4)

	oc5 := occupiedCore(t, 5)
	oc5.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 5}
	c5 := coreState(t, oc5)

	oc6 := occupiedCore(t, 6)
	oc6.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 6}
	oc6.TimeoutAt = blockUnderProduction
	oc6.Availability.Not()
	c6 := coreState(t, oc6)

	oc7 := occupiedCore(t, 7)
	oc7.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 7}
	oc7.TimeoutAt = blockUnderProduction
	oc7.CandidateHash = hashFromU64BE(71)
	c7 := coreState(t, oc7)

	oc8 := occupiedCore(t, 8)
	oc8.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 8}
	oc8.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 8}
	oc8.Availability.Not()
	oc8.CandidateHash = hashFromU64BE(81)
	c8 := coreState(t, oc8)

	oc9 := occupiedCore(t, 9)
	oc9.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 9}
	oc9.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 9}
	c9 := coreState(t, oc9)

	oc10 := occupiedCore(t, 10)
	oc10.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 10}
	oc10.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 10}
	oc10.TimeoutAt = blockUnderProduction
	oc10.CandidateHash = hashFromU64BE(101)
	c10 := coreState(t, oc10)

	oc11 := occupiedCore(t, 11)
	oc11.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 12}
	oc11.Availability.Not()
	c11 := coreState(t, oc11)

	return []parachaintypes.CoreState{free, c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11}
}

func mockAvailabilityCoresMultiplePerPara(t *testing.T) []parachaintypes.CoreState {
	t.Helper()
	free := coreState(t, parachaintypes.Free{})
	c1 := coreState(t, parachaintypes.ScheduledCore{ParaID: 1})

	c2 := coreState(t, occupiedCore(t, 2))

	oc3 := occupiedCore(t, 3)
	oc3.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 3}
	c3 := coreState(t, oc3)

	oc4 := occupiedCore(t, 4)
	oc4.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 4}
	oc4.Availability.Not()
	oc4.CandidateHash = hashFromU64BE(41)
	c4 := coreState(t, oc4)

	oc5 := occupiedCore(t, 5)
	oc5.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 5}
	c5 := coreState(t, oc5)

	oc6 := occupiedCore(t, 6)
	oc6.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 6}
	oc6.TimeoutAt = blockUnderProduction
	oc6.Availability.Not()
	c6 := coreState(t, oc6)

	oc7 := occupiedCore(t, 7)
	oc7.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 7}
	oc7.TimeoutAt = blockUnderProduction
	oc7.CandidateHash = hashFromU64BE(71)
	c7 := coreState(t, oc7)

	oc8 := occupiedCore(t, 8)
	oc8.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 8}
	oc8.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 8}
	oc8.Availability.Not()
	oc8.CandidateHash = hashFromU64BE(81)
	c8 := coreState(t, oc8)

	oc9 := occupiedCore(t, 9)
	oc9.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 9}
	oc9.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 9}
	c9 := coreState(t, oc9)

	oc10 := occupiedCore(t, 10)
	oc10.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 10}
	oc10.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 10}
	oc10.TimeoutAt = blockUnderProduction
	oc10.CandidateHash = hashFromU64BE(101)
	c10 := coreState(t, oc10)

	oc11 := occupiedCore(t, 11)
	oc11.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 12}
	oc11.Availability.Not()
	c11 := coreState(t, oc11)

	oc12a := occupiedCore(t, 12)
	oc12a.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 12}
	oc12a.Availability.Not()
	oc12a.CandidateHash = hashFromU64BE(121)
	c12a := coreState(t, oc12a)

	oc12b := occupiedCore(t, 12)
	oc12b.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 12}
	oc12b.Availability.Not()
	oc12b.CandidateHash = hashFromU64BE(122)
	c12b := coreState(t, oc12b)

	oc12c := occupiedCore(t, 12)
	oc12c.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 12}
	oc12c.Availability.Not()
	oc12c.CandidateHash = hashFromU64BE(123)
	c12c := coreState(t, oc12c)

	c15 := coreState(t, parachaintypes.ScheduledCore{ParaID: 12})

	oc16 := occupiedCore(t, 13)
	oc16.CandidateHash = hashFromU64BE(131)
	c16 := coreState(t, oc16)

	oc17 := occupiedCore(t, 13)
	oc17.Availability.Not()
	oc17.CandidateHash = hashFromU64BE(132)
	c17 := coreState(t, oc17)

	oc18 := occupiedCore(t, 13)
	oc18.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 13}
	oc18.CandidateHash = hashFromU64BE(133)
	c18 := coreState(t, oc18)

	oc19 := occupiedCore(t, 13)
	oc19.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 13}
	oc19.Availability.Not()
	oc19.CandidateHash = hashFromU64BE(134)
	c19 := coreState(t, oc19)

	oc20 := occupiedCore(t, 13)
	oc20.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 13}
	oc20.CandidateHash = hashFromU64BE(135)
	c20 := coreState(t, oc20)

	oc21 := occupiedCore(t, 13)
	oc21.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 14}
	oc21.Availability.Not()
	oc21.CandidateHash = hashFromU64BE(136)
	c21 := coreState(t, oc21)

	oc22 := occupiedCore(t, 13)
	oc22.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 14}
	oc22.CandidateHash = hashFromU64BE(137)
	c22 := coreState(t, oc22)

	oc23 := occupiedCore(t, 13)
	oc23.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 14}
	oc23.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 14}
	oc23.Availability.Not()
	oc23.CandidateHash = hashFromU64BE(138)
	c23 := coreState(t, oc23)

	oc24 := occupiedCore(t, 13)
	oc24.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 14}
	oc24.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 14}
	oc24.TimeoutAt = blockUnderProduction
	oc24.CandidateHash = hashFromU64BE(1399)
	c24 := coreState(t, oc24)

	oc25 := occupiedCore(t, 13)
	oc25.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 15}
	oc25.Availability.Not()
	oc25.CandidateHash = hashFromU64BE(139)
	c25 := coreState(t, oc25)

	oc26 := occupiedCore(t, 15)
	oc26.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 13}
	oc26.Availability.Not()
	oc26.CandidateHash = hashFromU64BE(151)
	c26 := coreState(t, oc26)

	oc27 := occupiedCore(t, 15)
	oc27.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 15}
	oc27.Availability.Not()
	oc27.TimeoutAt = blockUnderProduction
	oc27.CandidateHash = hashFromU64BE(152)
	c27 := coreState(t, oc27)

	oc28 := occupiedCore(t, 13)
	oc28.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 13}
	oc28.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 13}
	oc28.CandidateHash = hashFromU64BE(1398)
	c28 := coreState(t, oc28)

	oc29 := occupiedCore(t, 13)
	oc29.NextUpOnAvailable = &parachaintypes.ScheduledCore{ParaID: 13}
	oc29.NextUpOnTimeOut = &parachaintypes.ScheduledCore{ParaID: 13}
	oc29.TimeoutAt = blockUnderProduction
	oc29.CandidateHash = hashFromU64BE(1397)
	c29 := coreState(t, oc29)

	return []parachaintypes.CoreState{free, c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12a, c12b, c12c, c15, c16,
		c17, c18, c19, c20, c21, c22, c23, c24, c25, c26, c27, c28, c29}
}

func makeCandidates(t *testing.T, coreCount int, expectedBackedIndices []int) (
	[]parachaintypes.CandidateHash,
	[]parachaintypes.BackedCandidate,
) {
	t.Helper()
	emptyHash, err := parachaintypes.PersistedValidationData{}.Hash()
	require.NoError(t, err)

	candidateTemplate := dummyCandidateReceiptV2(common.Hash{})
	candidateTemplate.Descriptor.PersistedValidationDataHash = emptyHash

	candidates := make([]parachaintypes.CandidateReceiptV2, coreCount)
	candidateHashes := make([]parachaintypes.CandidateHash, coreCount)

	for i := 0; i < coreCount; i++ {
		candidate := *candidateTemplate
		candidate.Descriptor.ParaID = parachaintypes.ParaID(i)
		candidates[i] = candidate

		hash, err := parachaintypes.GetCandidateHash(candidate)
		require.NoError(t, err)
		candidateHashes[i] = hash
	}

	expectedBacked := make([]parachaintypes.BackedCandidate, len(expectedBackedIndices))
	for i, idx := range expectedBackedIndices {
		committed := parachaintypes.CommittedCandidateReceiptV2{
			Descriptor:  candidates[idx].Descriptor,
			Commitments: parachaintypes.CandidateCommitments{}, // Default empty commitments
		}

		backedCandidate, _ := parachaintypes.NewBackedCandidate(
			committed,
			nil,
			make([]bool, mockGroupSize),
			&parachaintypes.CoreIndex{Index: 0},
		)
		expectedBacked[i] = *backedCandidate
	}

	return candidateHashes, expectedBacked
}

func candidateHash(t *testing.T, candidate parachaintypes.Hashable) parachaintypes.CandidateHash {
	t.Helper()

	hash, err := parachaintypes.GetCandidateHash(candidate)
	require.NoError(t, err)
	return hash
}

// FromLowU64BE creates a new Hash from a uint64 value in big-endian format.
// The uint64 value will be placed at the end of the hash buffer.
func hashFromU64BE(val uint64) common.Hash {
	h := common.Hash{}
	// Convert uint64 to 8 bytes in big-endian format
	b := make([]byte, 8)
	b[0] = byte(val >> 56)
	b[1] = byte(val >> 48)
	b[2] = byte(val >> 40)
	b[3] = byte(val >> 32)
	b[4] = byte(val >> 24)
	b[5] = byte(val >> 16)
	b[6] = byte(val >> 8)
	b[7] = byte(val)

	// Place the 8 bytes at the end of the hash
	copy(h[common.HashLength-8:], b)
	return h
}

func TestRequestInherentData(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		perRP              map[common.Hash]*perRelayParent
		mockBlockState     func(*gomock.Controller) BlockState
		expectErr          bool
		expectAwaitInherit bool
	}{
		{
			name:  "unknown_relay_parent",
			perRP: map[common.Hash]*perRelayParent{},
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			expectErr: false,
		},
		{
			name: "inherent_not_ready",
			perRP: map[common.Hash]*perRelayParent{
				{1}: {
					leaf:            &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
					isInherentReady: false,
				},
			},
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			expectErr:          false,
			expectAwaitInherit: true,
		},
		{
			// This case triggers the sendInherentData method which has its own unit test.
			// To avoid redundant testing, we simulate a GetRuntime error here.
			name: "inherent_ready",
			perRP: map[common.Hash]*perRelayParent{
				{1}: {
					leaf:            &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
					isInherentReady: true,
					signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
				},
			},
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				bs := NewMockBlockState(ctrl)
				bs.EXPECT().
					GetRuntime(common.Hash{1}).
					Return(nil, fmt.Errorf("mock runtime error")).
					Times(1)
				return bs
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			msg := provisionermessages.RequestInherentData{
				RelayParent:             common.Hash{1},
				ProvisionerInherentData: make(chan provisionermessages.ProvisionerInherentData),
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			overseerChan := make(chan any)
			bs := tc.mockBlockState(ctrl)
			provisioner := New(overseerChan, bs)
			provisioner.perRelayParent = tc.perRP

			err := provisioner.requestInherentData(msg)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectAwaitInherit {
				perRP := provisioner.perRelayParent[msg.RelayParent]
				require.Len(t, perRP.awaitingInherent, 1)
				require.Equal(t, msg.ProvisionerInherentData, perRP.awaitingInherent[0])
			}
		})
	}
}

func TestProcessAvailableInherent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		relayParent    common.Hash
		perRP          map[common.Hash]*perRelayParent
		mockBlockState func(*gomock.Controller) BlockState
		expectErr      bool
		validateState  func(*testing.T, *perRelayParent)
	}{
		{
			name:        "relay_parent_not_exists",
			relayParent: common.Hash{1},
			perRP:       make(map[common.Hash]*perRelayParent),
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			expectErr: false,
		},
		{
			name:        "no_awaiting_inherent",
			relayParent: common.Hash{1},
			perRP: map[common.Hash]*perRelayParent{
				{1}: {
					leaf:             &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
					signedBitfields:  []parachaintypes.CheckedSignedAvailabilityBitfield{},
					isInherentReady:  false,
					awaitingInherent: nil,
				},
			},
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				return NewMockBlockState(ctrl)
			},
			expectErr: false,
			validateState: func(t *testing.T, perRP *perRelayParent) {
				require.True(t, perRP.isInherentReady)
				require.Empty(t, perRP.awaitingInherent)
			},
		},
		{
			// This case triggers the sendInherentData method which has its own unit test.
			// To avoid redundant testing, we simulate a GetRuntime error here.
			name:        "with_awaiting_inherent_get_runtime_error",
			relayParent: common.Hash{1},
			perRP: map[common.Hash]*perRelayParent{
				{1}: {
					leaf:            &parachaintypes.ActivatedLeaf{Hash: common.Hash{1}},
					signedBitfields: []parachaintypes.CheckedSignedAvailabilityBitfield{},
					isInherentReady: false,
					awaitingInherent: []chan provisionermessages.ProvisionerInherentData{
						make(chan provisionermessages.ProvisionerInherentData),
					},
				},
			},
			mockBlockState: func(ctrl *gomock.Controller) BlockState {
				bs := NewMockBlockState(ctrl)
				bs.EXPECT().
					GetRuntime(gomock.Any()).
					Return(nil, fmt.Errorf("mock runtime error"))
				return bs
			},
			expectErr: true,
			validateState: func(t *testing.T, perRP *perRelayParent) {
				require.True(t, perRP.isInherentReady)
				require.Empty(t, perRP.awaitingInherent)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			overseerChan := make(chan any)
			bs := tc.mockBlockState(ctrl)
			provisioner := New(overseerChan, bs)
			provisioner.perRelayParent = tc.perRP

			err := provisioner.processAvailableInherent(tc.relayParent)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.validateState != nil {
				perRP := provisioner.perRelayParent[tc.relayParent]
				tc.validateState(t, perRP)
			}
		})
	}
}
