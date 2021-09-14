package sync

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	syncmocks "github.com/ChainSafe/gossamer/dot/sync/mocks"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/ChainSafe/gossamer/lib/trie"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testTimeout = time.Second * 5

func newTestChainSync(t *testing.T) (*chainSync, <-chan *types.BlockData) { //nolint
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(0), types.Digest{})
	require.NoError(t, err)

	bs := new(syncmocks.MockBlockState)
	bs.On("BestBlockHeader").Return(header, nil)

	net := new(syncmocks.MockNetwork)
	net.On("DoBlockRequest", mock.AnythingOfType("peer.ID"), mock.AnythingOfType("*network.BlockRequestMessage")).Return(nil, nil)

	readyBlocks := make(chan *types.BlockData, MAX_RESPONSE_SIZE)
	return newChainSync(bs, net, readyBlocks), readyBlocks
}

func TestChainSync_SetPeerHead(t *testing.T) {
	cs, _ := newTestChainSync(t)

	testPeer := peer.ID("noot")
	hash := common.Hash{0xa, 0xb}
	number := big.NewInt(1000)
	err := cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)

	expected := &peerState{
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	require.Equal(t, expected, <-cs.workQueue)
	require.True(t, cs.pendingBlocks.hasBlock(hash))

	// test case where peer has a lower head than us, but they are on the same chain as us
	cs.blockState = new(syncmocks.MockBlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.Digest{})
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(998), types.Digest{})
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(hash, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put chain we already have into work queue")
	default:
	}

	// test case where peer has a lower head than us, and they are on an invalid fork
	cs.blockState = new(syncmocks.MockBlockState)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.Digest{})
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.True(t, errors.Is(err, errPeerOnInvalidFork))
	expected = &peerState{
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put invalid fork into work queue")
	default:
	}

	// test case where peer has a lower head than us, but they are on a valid fork (that is not our chain)
	cs.blockState = new(syncmocks.MockBlockState)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)
	fin, err = types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(998), types.Digest{})
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHighestFinalisedHeader").Return(fin, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("GetHashByNumber", mock.AnythingOfType("*big.Int")).Return(common.Hash{}, nil)
	cs.blockState.(*syncmocks.MockBlockState).On("HasHeader", mock.AnythingOfType("common.Hash")).Return(true, nil)

	number = big.NewInt(999)
	err = cs.setPeerHead(testPeer, hash, number)
	require.NoError(t, err)
	expected = &peerState{
		hash:   hash,
		number: number,
	}
	require.Equal(t, expected, cs.peerState[testPeer])
	select {
	case <-cs.workQueue:
		t.Fatal("should not put fork we already have into work queue")
	default:
	}
}

func TestChainSync_sync_bootstrap_withWorkerError(t *testing.T) {
	cs, _ := newTestChainSync(t)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: big.NewInt(1000),
	}

	cs.workQueue <- cs.peerState[testPeer]

	select {
	case res := <-cs.resultQueue:
		expected := &workerError{
			err: errNilResponse, // since MockNetwork returns a nil response
			who: testPeer,
		}
		require.Equal(t, expected, res.err)
	case <-time.After(testTimeout):
		t.Fatal("did not get worker response")
	}

	require.Equal(t, bootstrap, cs.state)
}

func TestChainSync_sync_tip(t *testing.T) {
	cs, _ := newTestChainSync(t)
	cs.blockState = new(syncmocks.MockBlockState)
	header, err := types.NewHeader(common.NewHash([]byte{0}), trie.EmptyHash, trie.EmptyHash, big.NewInt(1000), types.Digest{})
	require.NoError(t, err)
	cs.blockState.(*syncmocks.MockBlockState).On("BestBlockHeader").Return(header, nil)

	go cs.sync()
	defer cs.cancel()

	testPeer := peer.ID("noot")
	cs.peerState[testPeer] = &peerState{
		number: big.NewInt(999),
	}

	cs.workQueue <- cs.peerState[testPeer]
	time.Sleep(time.Second)
	require.Equal(t, tip, cs.state)
}

func TestChainSync_getTarget(t *testing.T) {
	cs, _ := newTestChainSync(t)
	require.Equal(t, big.NewInt(2<<32-1), cs.getTarget())

	cs.peerState = map[peer.ID]*peerState{
		peer.ID("testA"): &peerState{
			number: big.NewInt(1000),
		},
	}

	require.Equal(t, big.NewInt(1000), cs.getTarget())

	cs.peerState = map[peer.ID]*peerState{
		peer.ID("testA"): &peerState{
			number: big.NewInt(1000),
		},
		peer.ID("testB"): &peerState{
			number: big.NewInt(2000),
		},
	}

	require.Equal(t, big.NewInt(1500), cs.getTarget())
}

func TestWorkerToRequests(t *testing.T) {
	w := &worker{
		startNumber:  big.NewInt(1),
		targetNumber: big.NewInt(129),
		direction:    DIR_ASCENDING,
	}

	start, _ := variadic.NewUint64OrHash(w.startNumber.Uint64())
	expected := []*BlockRequestMessage{
		&BlockRequestMessage{
			RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
			StartingBlock: start,
			EndBlockHash:  optional.NewHash(false, common.Hash{}),
			Direction:     DIR_ASCENDING,
			Max:           optional.NewUint32(true, uint32(128)),
		},
	}

	reqs, err := workerToRequests(w)
	require.NoError(t, err)
	require.Equal(t, 1, len(reqs))
	require.Equal(t, expected, reqs)
}
