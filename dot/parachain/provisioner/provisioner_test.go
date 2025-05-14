package provisioner

import (
	"testing"
	"time"

	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

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

			p := New()
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

			p := New()
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
