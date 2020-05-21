package grandpa

import (
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
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
			key: k.Public().(*ed25519.PublicKey),
			id:  uint64(i),
		})
	}

	return voters
}

func TestCheckForEquivocation_NoEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	for _, v := range voters {
		equivocated := gs.checkForEquivocation(v, vote)
		require.False(t, equivocated)
	}
}

func TestCheckForEquivocation_WithEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 3)
		if len(branches) != 0 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter.key.AsBytes()] = vote

	vote2 := NewVoteFromHeader(branches[0])
	require.NoError(t, err)

	equivocated := gs.checkForEquivocation(voter, vote2)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))
	require.Equal(t, 2, len(gs.equivocations[voter.key.AsBytes()]))
}

func TestCheckForEquivocation_WithExistingEquivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) > 1 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter.key.AsBytes()] = vote

	vote2 := NewVoteFromHeader(branches[0])
	require.NoError(t, err)

	equivocated := gs.checkForEquivocation(voter, vote2)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))

	vote3 := NewVoteFromHeader(branches[1])
	require.NoError(t, err)

	equivocated = gs.checkForEquivocation(voter, vote3)
	require.True(t, equivocated)

	require.Equal(t, 0, len(gs.votes))
	require.Equal(t, 1, len(gs.equivocations))
	require.Equal(t, 3, len(gs.equivocations[voter.key.AsBytes()]))
}

func TestValidateMessage_Valid(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	msg, err := gs.CreateVoteMessage(h, kr.Alice)
	require.NoError(t, err)

	vote, err := gs.ValidateMessage(msg)
	require.NoError(t, err)
	require.Equal(t, h.Hash(), vote.hash)
}

func TestValidateMessage_InvalidSignature(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	msg, err := gs.CreateVoteMessage(h, kr.Alice)
	require.NoError(t, err)

	msg.message.signature[63] = 0

	_, err = gs.ValidateMessage(msg)
	require.Equal(t, err, ErrInvalidSignature)
}

func TestValidateMessage_SetIDMismatch(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3)

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	msg, err := gs.CreateVoteMessage(h, kr.Alice)
	require.NoError(t, err)

	gs.state.setID = 1

	_, err = gs.ValidateMessage(msg)
	require.Equal(t, err, ErrSetIDMismatch)
}

func TestValidateMessage_Equivocation(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)

	vote := NewVoteFromHeader(h)
	require.NoError(t, err)

	voter := voters[0]

	gs.votes[voter.key.AsBytes()] = vote

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	msg, err := gs.CreateVoteMessage(branches[0], kr.Alice)
	require.NoError(t, err)

	_, err = gs.ValidateMessage(msg)
	require.Equal(t, ErrEquivocation, err, gs.votes)
}

func TestValidateMessage_BlockDoesNotExist(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)
	state.AddBlocksToState(t, st.Block, 3)

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	fake := &types.Header{
		Number: big.NewInt(77),
	}

	msg, err := gs.CreateVoteMessage(fake, kr.Alice)
	require.NoError(t, err)

	_, err = gs.ValidateMessage(msg)
	require.Equal(t, err, ErrBlockDoesNotExist)
}

func TestValidateMessage_IsNotDescendant(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	for {
		_, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	h, err := st.Block.BestBlockHeader()
	require.NoError(t, err)
	gs.head = h.Hash()

	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	msg, err := gs.CreateVoteMessage(branches[0], kr.Alice)
	require.NoError(t, err)

	_, err = gs.ValidateMessage(msg)
	require.Equal(t, ErrDescendantNotFound, err, gs.votes)
}

func TestPubkeyToVoter(t *testing.T) {
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	state := NewState(voters, 0, 0)
	voter, err := state.pubkeyToVoter(kr.Alice.Public().(*ed25519.PublicKey))
	require.NoError(t, err)
	require.Equal(t, voters[0], voter)
}

func TestGetDirectVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	voteA := &Vote{
		hash:   common.Hash{0xa},
		number: 1,
	}

	voteB := &Vote{
		hash:   common.Hash{0xb},
		number: 1,
	}

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.votes[voter] = voteA
		} else {
			gs.votes[voter] = voteB
		}
	}

	directVotes := gs.getDirectVotes()
	require.Equal(t, 2, len(directVotes))
	require.Equal(t, uint64(5), directVotes[*voteA])
	require.Equal(t, uint64(4), directVotes[*voteB])
}

func TestGetVotesForBlock_NoDescendantVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	var chain []*types.Header

	for {
		chain, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	voteA := NewVoteFromHeader(chain[7])
	voteB := NewVoteFromHeader(branches[0])

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 5 {
			gs.votes[voter] = voteA
		} else {
			gs.votes[voter] = voteB
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.hash)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForA)

	votesForB, err := gs.getVotesForBlock(voteB.hash)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForB)
}

func TestGetVotesForBlock_DescendantVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	var chain []*types.Header

	for {
		chain, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	// A is a descendant of B
	voteA := NewVoteFromHeader(chain[7])
	voteB := NewVoteFromHeader(chain[5])
	voteC := NewVoteFromHeader(branches[0])

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.votes[voter] = voteA
		} else if i < 5 {
			gs.votes[voter] = voteB
		} else {
			gs.votes[voter] = voteC
		}
	}

	votesForA, err := gs.getVotesForBlock(voteA.hash)
	require.NoError(t, err)
	require.Equal(t, uint64(3), votesForA)

	// votesForB should be # of votes for A + # of votes for B
	votesForB, err := gs.getVotesForBlock(voteB.hash)
	require.NoError(t, err)
	require.Equal(t, uint64(5), votesForB)

	votesForC, err := gs.getVotesForBlock(voteC.hash)
	require.NoError(t, err)
	require.Equal(t, uint64(4), votesForC)
}

func TestGetPossiblePreVotedBlocks_OneBlock(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var branches []*types.Header
	var chain []*types.Header

	for {
		chain, branches = state.AddBlocksToState(t, st.Block, 8)
		if len(branches) != 0 {
			break
		}
	}

	voteA := NewVoteFromHeader(chain[7])
	voteB := NewVoteFromHeader(branches[0])

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 6 {
			gs.votes[voter] = voteA
		} else {
			gs.votes[voter] = voteB
		}
	}

	blocks, err := gs.getPossiblePreVotedBlocks()
	require.NoError(t, err)
	require.Equal(t, 1, len(blocks))
	require.Equal(t, *voteA, blocks[0])
}

func TestGetPossiblePreVotedBlocks_EqualVotes(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 2 {
			break
		}
	}

	// 1/3 voters each vote for a block on a different chain
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.votes[voter] = voteA
		} else if i < 6 {
			gs.votes[voter] = voteB
		} else {
			gs.votes[voter] = voteC
		}
	}

	t.Log(st.Block.BlocktreeAsString())

	blocks, err := gs.getPossiblePreVotedBlocks()
	require.NoError(t, err)

	for _, v := range blocks {
		t.Log((&v).String())
	}

	// TODO: this should return the highest common predecessors of (a, b), (a, c), (b, c)
	require.Equal(t, 3, len(blocks))
}

func TestGetPossiblePreVotedBlocks_OneThirdEquivocating(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 1 {
			break
		}
	}

	// 1/3 of voters equivocate; ie. vote for both blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 3 {
			gs.votes[voter] = voteA
		} else if i < 6 {
			gs.votes[voter] = voteB
		} else {
			gs.equivocations[voter] = []*Vote{voteA, voteB}
		}
	}

	blocks, err := gs.getPossiblePreVotedBlocks()
	require.NoError(t, err)
	require.Equal(t, 2, len(blocks))
}

func TestGetPossiblePreVotedBlocks_MoreThanOneThirdEquivocating(t *testing.T) {
	st := newTestState(t)
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	gs, err := NewService(st.Block, voters)
	require.NoError(t, err)

	var leaves []common.Hash
	for {
		state.AddBlocksToState(t, st.Block, 8)
		leaves = gs.blockState.Leaves()
		if len(leaves) > 2 {
			break
		}
	}

	// this tests a byzantine case where >1/3 of voters equivocate; ie. vote for multiple blocks
	voteA, err := NewVoteFromHash(leaves[0], st.Block)
	require.NoError(t, err)
	voteB, err := NewVoteFromHash(leaves[1], st.Block)
	require.NoError(t, err)
	voteC, err := NewVoteFromHash(leaves[2], st.Block)
	require.NoError(t, err)

	for i, k := range kr.Keys {
		voter := k.Public().(*ed25519.PublicKey).AsBytes()

		if i < 2 {
			// 2 votes for A
			gs.votes[voter] = voteA
		} else if i < 4 {
			// 2 votes for B
			gs.votes[voter] = voteB
		} else if i < 5 {
			// 1 vote for C
			gs.votes[voter] = voteC
		} else {
			// 4 equivocators
			gs.equivocations[voter] = []*Vote{voteA, voteB}
		}
	}

	blocks, err := gs.getPossiblePreVotedBlocks()
	require.NoError(t, err)
	require.Equal(t, 2, len(blocks))
}
