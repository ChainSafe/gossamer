package grandpa

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/stretchr/testify/require"
)

// testGenesisHeader is a test block header
var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

func newTestState(t *testing.T) *state.Service {
	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err := stateSrvc.Initialize(genesisData, testGenesisHeader, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	return stateSrvc
}

func newTestVoters(t *testing.T) []*Voter {
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	voters := []*Voter{}
	for i, k := range kr.Keys {
		voters = append(voters, &Voter{
			key:     k.Public().(*ed25519.PublicKey),
			voterID: uint64(i),
		})
	}

	return voters
}

func TestCheckForEquivocation_NoEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs := NewService(st.Block, voters)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	for _, v := range voters {
		equivocated, err := gs.checkForEquivocation(v, vote)
		require.NoError(t, err)
		require.False(t, equivocated)
	}
}

func TestCheckForEquivocation_NoEquivocation_MultipleVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs := NewService(st.Block, voters)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()
	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter] = vote

	h2, err := st.Block.GetHeader(h.ParentHash)
	vote2 := NewVoteFromHeader(h2)
	require.NoError(t, err)

	equivocated, err := gs.checkForEquivocation(voter, vote2)
	require.NoError(t, err)
	require.False(t, equivocated)
	// TODO: if the same voter votes for multiple blocks in a round, but the blocks are on the same chain, are all
	// those votes counted? if so, we will need to change `votes` to be a map of Voter to array of Votes.
	require.Equal(t, 1, len(gs.votes))
	require.Equal(t, 0, len(gs.equivocations))
}

func TestCheckForEquivocation_WithEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs := NewService(st.Block, voters)
	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 3)
		if len(branches) != 0 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter] = vote

	vote2 := NewVoteFromHeader(branches[0])
	require.NoError(t, err)

	equivocated, err := gs.checkForEquivocation(voter, vote2)
	require.NoError(t, err)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))
}

func TestCheckForEquivocation_WithExistingEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs := NewService(st.Block, voters)
	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) > 1 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter] = vote

	vote2 := NewVoteFromHeader(branches[0])
	require.NoError(t, err)

	equivocated, err := gs.checkForEquivocation(voter, vote2)
	require.NoError(t, err)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))

	vote3 := NewVoteFromHeader(branches[1])
	require.NoError(t, err)

	equivocated, err = gs.checkForEquivocation(voter, vote3)
	require.NoError(t, err)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))
	require.Equal(t, 3, len(gs.equivocations[voter]))
}
