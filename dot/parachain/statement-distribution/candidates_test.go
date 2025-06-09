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

// TestReturnedPostConfirmation mirrors the Rust test `test_returned_post_confirmation`.
func TestReturnedPostConfirmation(t *testing.T) {
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

	paraID1 := parachaintypes.ParaID(1)
	paraID2 := parachaintypes.ParaID(2)

	candidateA := util.MakeCandidate(relayHash, 1, paraID1, relayHeadData, candidateHeadDataA,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov1000"))))
	pvdA := util.DummyPVD(relayHeadData, 1000)
	candidateB := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataA, candidateHeadDataB,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov2000"))))
	pvdB := util.DummyPVD(candidateHeadDataA, 2000)
	candidateC := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataA, candidateHeadDataC,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov3000"))))
	// no pvd for candidateC
	candidateD := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataB, candidateHeadDataD,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov4000"))))
	pvdD := util.DummyPVD(candidateHeadDataB, 4000)

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

	// Deterministic peer IDs for reproducibility
	peerA, err := peer.Decode("12D3KooWJ6X7Qw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v")
	require.NoError(t, err)
	peerB, err := peer.Decode("12D3KooWJ6X7Qw2vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v")
	require.NoError(t, err)
	peerC, err := peer.Decode("12D3KooWJ6X7Qw3vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v")
	require.NoError(t, err)
	peerD, err := peer.Decode("12D3KooWJ6X7Qw4vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1vQw1v")
	require.NoError(t, err)

	groupIndex := parachaintypes.GroupIndex(100)

	candidates := &candidates{
		candidates: make(map[parachaintypes.CandidateHash]candidateState),
		byParent:   make(map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}),
	}

	// Advertise A without parent hash.
	err = candidates.insertUnconfirmed(peerA, candidateHashA, relayHash, groupIndex, nil)
	require.NoError(t, err)

	// Advertise A with parent hash and ID.
	err = candidates.insertUnconfirmed(peerA, candidateHashA, relayHash, groupIndex,
		&hashAndParaID{Hash: relayHash, ParaID: paraID1})
	require.NoError(t, err)

	// (Correctly) advertise B with parent A. Do it from a couple of peers.
	err = candidates.insertUnconfirmed(peerA, candidateHashB, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)
	err = candidates.insertUnconfirmed(peerB, candidateHashB, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)

	// (Wrongly) advertise C with parent A. Do it from a couple peers.
	err = candidates.insertUnconfirmed(peerB, candidateHashC, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)
	err = candidates.insertUnconfirmed(peerC, candidateHashC, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)

	// Advertise D. Do it correctly from one peer (parent B) and wrongly from another (parent A).
	err = candidates.insertUnconfirmed(peerC, candidateHashD, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashB, ParaID: paraID1})
	require.NoError(t, err)
	err = candidates.insertUnconfirmed(peerD, candidateHashD, relayHash, groupIndex,
		&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)

	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}, candidateHashD: {}},
		{Hash: candidateHeadDataHashB, ParaID: paraID1}: {candidateHashD: {}},
	}, candidates.byParent)

	// Confirm candidate A
	postConf, err := candidates.confirmCandidate(candidateHashA, candidateA, pvdA, groupIndex)
	require.NoError(t, err)
	require.NotNil(t, postConf)
	require.Equal(t, &parachaintypes.HypotheticalCandidateComplete{
		ClaimedCandidateHash:      candidateHashA,
		CommittedCandidateReceipt: candidateA,
		PersistedValidationData:   pvdA,
	}, postConf.hypothetical)
	require.Equal(t, map[peer.ID]struct{}{peerA: {}}, postConf.reckoning.correct)
	require.Empty(t, postConf.reckoning.incorrect)

	// Confirm candidate B
	postConf, err = candidates.confirmCandidate(candidateHashB, candidateB, pvdB, groupIndex)
	require.NoError(t, err)
	require.NotNil(t, postConf)
	require.Equal(t, &parachaintypes.HypotheticalCandidateComplete{
		ClaimedCandidateHash:      candidateHashB,
		CommittedCandidateReceipt: candidateB,
		PersistedValidationData:   pvdB,
	}, postConf.hypothetical)
	require.ElementsMatch(t, []peer.ID{peerA, peerB}, keys(postConf.reckoning.correct))
	require.Empty(t, postConf.reckoning.incorrect)

	// Confirm candidate C with two wrong peers (different group index)
	newCandidateC := util.MakeCandidate(relayHash, 1, paraID2, candidateHeadDataB, candidateHeadDataC,
		parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov3000"))))
	newPvdC := util.DummyPVD(candidateHeadDataB, 3000)
	postConf, err = candidates.confirmCandidate(candidateHashC, newCandidateC, newPvdC, groupIndex)
	require.NoError(t, err)
	require.NotNil(t, postConf)
	require.Equal(t, &parachaintypes.HypotheticalCandidateComplete{
		ClaimedCandidateHash:      candidateHashC,
		CommittedCandidateReceipt: newCandidateC,
		PersistedValidationData:   newPvdC,
	}, postConf.hypothetical)
	require.Empty(t, postConf.reckoning.correct)
	require.ElementsMatch(t, []peer.ID{peerB, peerC}, keys(postConf.reckoning.incorrect))

	// Confirm candidate D with one correct and one wrong peer
	postConf, err = candidates.confirmCandidate(candidateHashD, candidateD, pvdD, groupIndex)
	require.NoError(t, err)
	require.NotNil(t, postConf)
	require.Equal(t, &parachaintypes.HypotheticalCandidateComplete{
		ClaimedCandidateHash:      candidateHashD,
		CommittedCandidateReceipt: candidateD,
		PersistedValidationData:   pvdD,
	}, postConf.hypothetical)
	require.Equal(t, map[peer.ID]struct{}{peerC: {}}, postConf.reckoning.correct)
	require.Equal(t, map[peer.ID]struct{}{peerD: {}}, postConf.reckoning.incorrect)
}

// TestHypotheticalFrontiers mirrors the Rust test `test_hypothetical_frontiers`.
func TestHypotheticalFrontiers(t *testing.T) {
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
	candidateHeadDataHashD, err := candidateHeadDataD.Hash()
	require.NoError(t, err)

	paraID1 := parachaintypes.ParaID(1)

	candidateA := util.MakeCandidate(relayHash, 1, paraID1, relayHeadData, candidateHeadDataA, parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov1000"))))
	pvdA := util.DummyPVD(relayHeadData, 1000)
	candidateB := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataA, candidateHeadDataB, parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov2000"))))
	candidateC := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataA, candidateHeadDataC, parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov3000"))))
	candidateD := util.MakeCandidate(relayHash, 1, paraID1, candidateHeadDataB, candidateHeadDataD, parachaintypes.ValidationCodeHash(common.MustBlake2bHash([]byte("pov4000"))))

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

	// Confirm A.
	_, err = candidates.confirmCandidate(candidateHashA, candidateA, pvdA, groupIndex)
	require.NoError(t, err)

	// Advertise B with parent A.
	err = candidates.insertUnconfirmed(peerID, candidateHashB, relayHash, groupIndex, &hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)

	// Advertise C with parent A.
	err = candidates.insertUnconfirmed(peerID, candidateHashC, relayHash, groupIndex, &hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.NoError(t, err)

	// Advertise D with parent B.
	err = candidates.insertUnconfirmed(peerID, candidateHashD, relayHash, groupIndex, &hashAndParaID{Hash: candidateHeadDataHashB, ParaID: paraID1})
	require.NoError(t, err)

	require.Equal(t, map[hashAndParaID]map[parachaintypes.CandidateHash]struct{}{
		{Hash: relayHash, ParaID: paraID1}:              {candidateHashA: {}},
		{Hash: candidateHeadDataHashA, ParaID: paraID1}: {candidateHashB: {}, candidateHashC: {}},
		{Hash: candidateHeadDataHashB, ParaID: paraID1}: {candidateHashD: {}},
	}, candidates.byParent)

	hypotheticalA := &parachaintypes.HypotheticalCandidateComplete{
		ClaimedCandidateHash:      candidateHashA,
		CommittedCandidateReceipt: candidateA,
		PersistedValidationData:   pvdA,
	}
	hypotheticalB := &parachaintypes.HypotheticalCandidateIncomplete{
		ClaimedCandidateHash: candidateHashB,
		CandidateParaID:      paraID1,
		ParentHeadDataHash:   candidateHeadDataHashA,
		RelayParent:          relayHash,
	}
	hypotheticalC := &parachaintypes.HypotheticalCandidateIncomplete{
		ClaimedCandidateHash: candidateHashC,
		CandidateParaID:      paraID1,
		ParentHeadDataHash:   candidateHeadDataHashA,
		RelayParent:          relayHash,
	}
	hypotheticalD := &parachaintypes.HypotheticalCandidateIncomplete{
		ClaimedCandidateHash: candidateHashD,
		CandidateParaID:      paraID1,
		ParentHeadDataHash:   candidateHeadDataHashB,
		RelayParent:          relayHash,
	}

	// Test frontierHypotheticals for parent (relayHash, paraID1)
	hypotheticals := candidates.frontierHypotheticals(&hashAndParaID{Hash: relayHash, ParaID: paraID1})
	require.Equal(t, []parachaintypes.HypotheticalCandidate{hypotheticalA}, hypotheticals)

	// Test for parent (candidateHeadDataHashA, ParaID(2))
	hypotheticals = candidates.frontierHypotheticals(&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: parachaintypes.ParaID(2)})
	require.Len(t, hypotheticals, 0)

	// Test for parent (candidateHeadDataHashA, paraID1)
	hypotheticals = candidates.frontierHypotheticals(&hashAndParaID{Hash: candidateHeadDataHashA, ParaID: paraID1})
	require.Len(t, hypotheticals, 2)
	require.Contains(t, hypotheticals, hypotheticalB)
	require.Contains(t, hypotheticals, hypotheticalC)

	// Test for parent (candidateHeadDataHashD, paraID1)
	hypotheticals = candidates.frontierHypotheticals(&hashAndParaID{Hash: candidateHeadDataHashD, ParaID: paraID1})
	require.Len(t, hypotheticals, 0)

	// Test for parent nil (all hypotheticals)
	hypotheticals = candidates.frontierHypotheticals(nil)
	require.Len(t, hypotheticals, 4)
	require.Contains(t, hypotheticals, hypotheticalA)
	require.Contains(t, hypotheticals, hypotheticalB)
	require.Contains(t, hypotheticals, hypotheticalC)
	require.Contains(t, hypotheticals, hypotheticalD)
}

// keys returns the keys of a map[peer.ID]struct{} as a slice.
func keys(m map[peer.ID]struct{}) []peer.ID {
	out := make([]peer.ID, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
