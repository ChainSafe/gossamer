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
