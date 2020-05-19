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
		ok, err := gs.checkForEquivocation(v, vote)
		require.NoError(t, err)
		require.False(t, ok)
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

	ok, err := gs.checkForEquivocation(voter, vote2)
	require.NoError(t, err)
	require.False(t, ok)
}
