package sync

import (
	"errors"
	"fmt"
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
		"testA": {
			number: big.NewInt(1000),
		},
	}

	require.Equal(t, big.NewInt(1000), cs.getTarget())

	cs.peerState = map[peer.ID]*peerState{
		"testA": {
			number: big.NewInt(1000),
		},
		"testB": {
			number: big.NewInt(2000),
		},
	}

	require.Equal(t, big.NewInt(1500), cs.getTarget())
}

func TestWorkerToRequests(t *testing.T) {
	_, err := workerToRequests(&worker{})
	require.Equal(t, errWorkerMissingStartNumber, err)

	w := &worker{
		startNumber: big.NewInt(1),
	}
	_, err = workerToRequests(w)
	require.Equal(t, errWorkerMissingTargetNumber, err)

	w = &worker{
		startNumber:  big.NewInt(10),
		targetNumber: big.NewInt(1),
		direction:    DIR_ASCENDING,
	}
	_, err = workerToRequests(w)
	require.Equal(t, errInvalidDirection, err)

	type testCase struct {
		w        *worker
		expected []*BlockRequestMessage
	}

	testCases := []testCase{
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + MAX_RESPONSE_SIZE),
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(1),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, uint32(128)),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + (MAX_RESPONSE_SIZE * 2)),
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(1),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(false, 0),
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: variadic.MustNewUint64OrHash(1 + MAX_RESPONSE_SIZE),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, uint32(128)),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(10),
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(1),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, 9),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(10),
				targetNumber: big.NewInt(1),
				direction:    DIR_DESCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(10),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_DESCENDING,
					Max:           optional.NewUint32(true, 9),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(1 + MAX_RESPONSE_SIZE + (MAX_RESPONSE_SIZE / 2)),
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(1),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(false, 0),
				},
				{
					RequestedData: network.RequestedDataHeader + network.RequestedDataBody + network.RequestedDataJustification,
					StartingBlock: variadic.MustNewUint64OrHash(1 + MAX_RESPONSE_SIZE),
					EndBlockHash:  optional.NewHash(false, common.Hash{}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, uint32(MAX_RESPONSE_SIZE/2)),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				targetNumber: big.NewInt(10),
				targetHash:   common.Hash{0xa},
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(1),
					EndBlockHash:  optional.NewHash(true, common.Hash{0xa}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, 9),
				},
			},
		},
		{
			w: &worker{
				startNumber:  big.NewInt(1),
				startHash:    common.Hash{0xb},
				targetNumber: big.NewInt(10),
				targetHash:   common.Hash{0xc},
				direction:    DIR_ASCENDING,
				requestData:  bootstrapRequestData,
			},
			expected: []*BlockRequestMessage{
				{
					RequestedData: bootstrapRequestData,
					StartingBlock: variadic.MustNewUint64OrHash(common.Hash{0xb}),
					EndBlockHash:  optional.NewHash(true, common.Hash{0xc}),
					Direction:     DIR_ASCENDING,
					Max:           optional.NewUint32(true, 9),
				},
			},
		},
	}

	for i, tc := range testCases {
		reqs, err := workerToRequests(tc.w)
		require.NoError(t, err, fmt.Sprintf("case %d failed", i))
		require.Equal(t, len(tc.expected), len(reqs), fmt.Sprintf("case %d failed", i))
		require.Equal(t, tc.expected, reqs, fmt.Sprintf("case %d failed", i))
	}
}

func TestValidateBlockData(t *testing.T) {
	req := &BlockRequestMessage{
		RequestedData: bootstrapRequestData,
	}

	err := validateBlockData(req, nil)
	require.Equal(t, errNilBlockData, err)

	err = validateBlockData(req, &types.BlockData{})
	require.Equal(t, errNilHeaderInResponse, err)

	err = validateBlockData(req, &types.BlockData{
		Header: &optional.Header{},
	})
	require.Equal(t, errNilBodyInResponse, err)

	err = validateBlockData(req, &types.BlockData{
		Header: &optional.Header{},
		Body:   &optional.Body{},
	})
	require.NoError(t, err)
}

func TestChainSync_validateResponse(t *testing.T) {
	cs, _ := newTestChainSync(t)
	err := cs.validateResponse(nil, nil)
	require.Equal(t, errEmptyBlockData, err)

	req := &BlockRequestMessage{
		RequestedData: network.RequestedDataHeader,
	}

	resp := &BlockResponseMessage{
		BlockData: []*types.BlockData{
			{
				Header: (&types.Header{
					Number: big.NewInt(1),
				}).AsOptional(),
				Body: (&types.Body{}).AsOptional(),
			},
			{
				Header: (&types.Header{
					Number: big.NewInt(2),
				}).AsOptional(),
				Body: (&types.Body{}).AsOptional(),
			},
		},
	}

	hash := (&types.Header{
		Number: big.NewInt(2),
	}).Hash()
	err = cs.validateResponse(req, resp)
	require.Equal(t, errResponseIsNotChain, err)
	require.True(t, cs.pendingBlocks.hasBlock(hash))
}
