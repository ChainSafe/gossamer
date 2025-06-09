package statementdistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestInsertingUnconfirmedRejectsOnIncompatibleClaims(t *testing.T) {
	relayHeadDataA := parachaintypes.HeadData{Data: []byte{1, 2, 3}}
	relayHeadDataB := parachaintypes.HeadData{Data: []byte{4, 5, 6}}
	relayHashA, err := relayHeadDataA.Hash()
	require.NoError(t, err)
	relayHashB, err := relayHeadDataB.Hash()
	require.NoError(t, err)

	paraIDA := parachaintypes.ParaID(1)
	paraIDB := parachaintypes.ParaID(2)

	pvdA := util.DummyPVD(relayHeadDataA, 1000)
	candidateA := util.MakeCandidate(
		relayHashA,
		1,
		paraIDA,
		relayHeadDataA,
		parachaintypes.HeadData{Data: []byte{1}},
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov1000"))),
	)

	candidateHashAVal, err := candidateA.Hash()
	require.NoError(t, err)
	candidateHashA := parachaintypes.CandidateHash{Value: candidateHashAVal}

	peerID, err := peer.Decode("12D3KooWJ6X7Qw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v") // deterministic peer ID for test
	require.NoError(t, err)

	groupIndexA := parachaintypes.GroupIndex(100)
	groupIndexB := parachaintypes.GroupIndex(200)

	candidates := &candidates{
		candidates: make(map[parachaintypes.CandidateHash]candidateState),
		byParent:   make(map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}),
	}

	// Confirm a candidate first.
	_, err = candidates.confirmCandidate(candidateHashA, candidateA, pvdA, groupIndexA)
	require.NoError(t, err)

	// Relay parent does not match.
	err = candidates.insertUnconfirmed(
		peerID,
		candidateHashA,
		relayHashB,
		groupIndexA,
		&hashAndParaID{Hash: relayHashA, ParaID: paraIDA},
	)
	require.ErrorIs(t, err, errBadAdvertisement)

	// Group index does not match.
	err = candidates.insertUnconfirmed(
		peerID,
		candidateHashA,
		relayHashA,
		groupIndexB,
		&hashAndParaID{Hash: relayHashA, ParaID: paraIDA},
	)
	require.ErrorIs(t, err, errBadAdvertisement)

	// Parent head data does not match.
	err = candidates.insertUnconfirmed(
		peerID,
		candidateHashA,
		relayHashA,
		groupIndexA,
		&hashAndParaID{Hash: relayHashB, ParaID: paraIDA},
	)
	require.ErrorIs(t, err, errBadAdvertisement)

	// Para ID does not match.
	err = candidates.insertUnconfirmed(
		peerID,
		candidateHashA,
		relayHashA,
		groupIndexA,
		&hashAndParaID{Hash: relayHashA, ParaID: paraIDB},
	)
	require.ErrorIs(t, err, errBadAdvertisement)

	// Everything matches.
	err = candidates.insertUnconfirmed(
		peerID,
		candidateHashA,
		relayHashA,
		groupIndexA,
		&hashAndParaID{Hash: relayHashA, ParaID: paraIDA},
	)
	require.NoError(t, err)
}

// Tests that:
//
//   - When the advertisement matches, confirming does not change the parent hash index.
//   - When it doesn't match, confirming updates the index. Specifically, confirming should prune
//     unconfirmed claims.
func TestConfirmingMaintainsParentHashIndex(t *testing.T) {
	relayHeadData := parachaintypes.HeadData{Data: []byte{1, 2, 3}}
	relayHash, err := relayHeadData.Hash()
	require.NoError(t, err)

	candidateHeadDataA := parachaintypes.HeadData{Data: []byte{1}}
	candidateHeadDataB := parachaintypes.HeadData{Data: []byte{2}}
	candidateHeadDataC := parachaintypes.HeadData{Data: []byte{3}}
	candidateHeadDataD := parachaintypes.HeadData{Data: []byte{4}}
	candidateHeadDataHashA, err := candidateHeadDataA.Hash()
	require.NoError(t, err)
	candidateHeadDataHashB, err := candidateHeadDataB.Hash()
	require.NoError(t, err)
	candidateHeadDataHashC, err := candidateHeadDataC.Hash()
	require.NoError(t, err)

	// ParaID 1 for all except new_candidate_c
	paraID1 := parachaintypes.ParaID(1)
	paraID2 := parachaintypes.ParaID(2)

	// Candidates and PVDs
	candidateA, pvdA := util.MakeCandidate(
		relayHash,
		1,
		paraID1,
		relayHeadData,
		candidateHeadDataA,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov1000"))),
	), util.DummyPVD(relayHeadData, 1000)

	candidateB, pvdB := util.MakeCandidate(
		relayHash,
		1,
		paraID1,
		candidateHeadDataA,
		candidateHeadDataB,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov2000"))),
	), util.DummyPVD(candidateHeadDataA, 2000)

	candidateC, _ := util.MakeCandidate(
		relayHash,
		1,
		paraID1,
		candidateHeadDataB,
		candidateHeadDataC,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov3000"))),
	), util.DummyPVD(candidateHeadDataB, 3000)

	candidateD, pvdD := util.MakeCandidate(
		relayHash,
		1,
		paraID1,
		candidateHeadDataC,
		candidateHeadDataD,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov4000"))),
	), util.DummyPVD(candidateHeadDataC, 4000)

	candidateHashAVal, err := candidateA.Hash()
	require.NoError(t, err)
	candidateHashA := parachaintypes.CandidateHash{Value: candidateHashAVal}

	candidateHashBVal, err := candidateB.Hash()
	require.NoError(t, err)
	candidateHashB := parachaintypes.CandidateHash{Value: candidateHashBVal}

	candidateHashCVal, err := candidateC.Hash()
	require.NoError(t, err)
	candidateHashC := parachaintypes.CandidateHash{Value: candidateHashCVal}

	candidateHashDVal, err := candidateD.Hash()
	require.NoError(t, err)
	candidateHashD := parachaintypes.CandidateHash{Value: candidateHashDVal}

	peerID, err := peer.Decode("12D3KooWJ6X7Qw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v")
	require.NoError(t, err)
	groupIndex := parachaintypes.GroupIndex(100)

	candidates := &candidates{
		candidates: make(map[parachaintypes.CandidateHash]candidateState),
		byParent:   make(map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}),
	}

	// Advertise A without parent hash.
	err = candidates.insertUnconfirmed(peerID, candidateHashA, relayHash, groupIndex, nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(candidates.byParent))

	// Advertise A with parent hash and ID.
	err = candidates.insertUnconfirmed(peerID, candidateHashA, relayHash, groupIndex,
		&hashAndParaID{Hash: relayHash, ParaID: paraID1})
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}: {candidateHashA: {}},
	}, candidates.byParent)

	// Advertise B with parent A.
	err = candidates.insertUnconfirmed(peerID, candidateHashB, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}},
	}, candidates.byParent)

	// Advertise C with parent A.
	err = candidates.insertUnconfirmed(peerID, candidateHashC, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}},
	}, candidates.byParent)

	// Advertise D with parent A.
	err = candidates.insertUnconfirmed(peerID, candidateHashD, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}, candidateHashD: {}},
	}, candidates.byParent)

	// Confirmed candidates and check parent hash index.
	// Confirmation matches advertisement. Index should be unchanged.
	_, err = candidates.confirmCandidate(candidateHashA, candidateA, pvdA, groupIndex)
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}, candidateHashD: {}},
	}, candidates.byParent)

	_, err = candidates.confirmCandidate(candidateHashB, candidateB, pvdB, groupIndex)
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}, candidateHashD: {}},
	}, candidates.byParent)

	// Confirmation does not match advertisement. Index should be updated.
	_, err = candidates.confirmCandidate(candidateHashD, candidateD, pvdD, groupIndex)
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}},
		{Hash: candidateHeadDataHashC, ParaID: paraID1}: {candidateHashD: {}},
	}, candidates.byParent)

	// Make a new candidate for C with a different para ID.
	newCandidateC, newPvdC := util.MakeCandidate(
		relayHash,
		1,
		paraID2,
		candidateHeadDataB,
		candidateHeadDataC,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov3000"))),
	), util.DummyPVD(candidateHeadDataB, 3000)
	_, err = candidates.confirmCandidate(candidateHashC, newCandidateC, newPvdC, groupIndex)
	require.NoError(t, err)
	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}},
		{Hash: candidateHeadDataHashB, ParaID: paraID2}: {candidateHashC: {}},
		{Hash: candidateHeadDataHashC, ParaID: paraID1}: {candidateHashD: {}},
	}, candidates.byParent)
}
