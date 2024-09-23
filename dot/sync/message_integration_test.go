//go:build integration

// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	runtime "github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/mock/gomock"

	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"

	"github.com/stretchr/testify/require"
)

func newWestendDevGenesisWithTrieAndHeader(t *testing.T) (
	gen genesis.Genesis, genesisTrie trie.Trie, genesisHeader types.Header) {
	t.Helper()

	genesisPath := utils.GetWestendDevRawGenesisPath(t)
	genesisPtr, err := genesis.NewGenesisFromJSONRaw(genesisPath)
	require.NoError(t, err)
	gen = *genesisPtr

	genesisTrie, err = runtime.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	parentHash := common.NewHash([]byte{0})
	stateRoot := genesisTrie.MustHash()
	extrinsicRoot := trie.EmptyHash
	const number = 0
	digest := types.NewDigest()
	genesisHeaderPtr := types.NewHeader(parentHash,
		stateRoot, extrinsicRoot, number, digest)
	genesisHeader = *genesisHeaderPtr

	return gen, genesisTrie, genesisHeader
}

func newFullSyncService(t *testing.T) *SyncService {
	ctrl := gomock.NewController(t)

	mockTelemetryClient := NewMockTelemetry(ctrl)
	mockTelemetryClient.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	wazero_runtime.DefaultTestLogLvl = log.Warn

	testDatadirPath := t.TempDir()

	scfg := state.Config{
		Path:              testDatadirPath,
		LogLevel:          log.Info,
		Telemetry:         mockTelemetryClient,
		GenesisBABEConfig: config.BABEConfigurationTestDefault,
	}
	stateSrvc := state.NewService(scfg)
	stateSrvc.UseMemDB()

	gen, genTrie, genHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&gen, &genHeader, genTrie)
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	// initialise runtime
	genState := rtstorage.NewTrieState(genTrie)

	rtCfg := wazero_runtime.Config{
		Storage: genState,
		LogLvl:  log.Critical,
	}

	if stateSrvc != nil {
		rtCfg.NodeStorage.BaseDB = stateSrvc.Base
	} else {
		rtCfg.NodeStorage.BaseDB, err = database.LoadDatabase(filepath.Join(testDatadirPath, "offline_storage"), false)
		require.NoError(t, err)
	}

	rtCfg.CodeHash, err = stateSrvc.Storage.LoadCodeHash(nil)
	require.NoError(t, err)

	instance, err := wazero_runtime.NewRuntimeFromGenesis(rtCfg)
	require.NoError(t, err)

	bestBlockHash := stateSrvc.Block.BestBlockHash()
	stateSrvc.Block.StoreRuntime(bestBlockHash, instance)

	blockImportHandler := NewMockBlockImportHandler(ctrl)
	blockImportHandler.EXPECT().HandleBlockImport(gomock.AssignableToTypeOf(&types.Block{}),
		gomock.AssignableToTypeOf(&rtstorage.TrieState{}), false).DoAndReturn(
		func(block *types.Block, ts *rtstorage.TrieState, _ bool) error {
			// store updates state trie nodes in database
			if err = stateSrvc.Storage.StoreTrie(ts, &block.Header); err != nil {
				logger.Warnf("failed to store state trie for imported block %s: %s", block.Header.Hash(), err)
				return err
			}

			// store block in database
			err = stateSrvc.Block.AddBlock(block)
			require.NoError(t, err)

			stateSrvc.Block.StoreRuntime(block.Header.Hash(), instance)
			logger.Debugf("imported block %s and stored state trie with root %s",
				block.Header.Hash(), ts.Trie().MustHash())
			return nil
		}).AnyTimes()

	mockBabeVerifier := NewMockBabeVerifier(ctrl)
	mockBabeVerifier.EXPECT().VerifyBlock(gomock.AssignableToTypeOf(&types.Header{})).AnyTimes()

	mockFinalityGadget := NewMockFinalityGadget(ctrl)
	mockFinalityGadget.EXPECT().
		VerifyBlockJustification(gomock.AssignableToTypeOf(common.Hash{}),
			gomock.AssignableToTypeOf(uint(0)), gomock.AssignableToTypeOf([]byte{})).
		Return(uint64(1), uint64(1), nil).
		AnyTimes()

	mockNetwork := NewMockNetwork(ctrl)

	fullSyncCfg := &FullSyncConfig{
		BlockState:         stateSrvc.Block,
		StorageState:       stateSrvc.Storage,
		BlockImportHandler: blockImportHandler,
		TransactionState:   stateSrvc.Transaction,
		BabeVerifier:       mockBabeVerifier,
		FinalityGadget:     mockFinalityGadget,
		Telemetry:          mockTelemetryClient,
		RequestMaker:       NewMockRequestMaker(ctrl),
	}

	fullSync := NewFullSyncStrategy(fullSyncCfg)

	serviceCfg := []ServiceConfig{
		WithBlockState(stateSrvc.Block),
		WithNetwork(mockNetwork),
		WithSlotDuration(6 * time.Second),
		WithStrategies(fullSync, nil),
	}

	syncer := NewSyncService(serviceCfg...)
	return syncer
}

func addTestBlocksToState(t *testing.T, depth uint, blockState BlockState) {
	previousHash := blockState.(*state.BlockState).BestBlockHash()
	previousNum, err := blockState.BestBlockNumber()
	require.NoError(t, err)

	digest := types.NewDigest()
	prd, err := types.NewBabeSecondaryPlainPreDigest(0, 1).ToPreRuntimeDigest()
	require.NoError(t, err)
	err = digest.Add(*prd)
	require.NoError(t, err)

	for i := uint(1); i <= depth; i++ {
		block := &types.Block{
			Header: types.Header{
				ParentHash: previousHash,
				Number:     previousNum + i,
				StateRoot:  trie.EmptyHash,
				Digest:     digest,
			},
			Body: types.Body{},
		}

		previousHash = block.Header.Hash()

		err := blockState.(*state.BlockState).AddBlock(block)
		require.NoError(t, err)
	}
}

func TestService_CreateBlockResponse_MaxSize(t *testing.T) {
	s := newFullSyncService(t)
	addTestBlocksToState(t, messages.MaxBlocksInResponse*2, s.blockState)

	// test ascending
	start := messages.NewFromBlock(uint(1))

	req := &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Ascending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(1), resp.BlockData[0].Number())
	require.Equal(t, uint(128), resp.BlockData[127].Number())

	max := uint32(messages.MaxBlocksInResponse + 100)
	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Ascending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(1), resp.BlockData[0].Number())
	require.Equal(t, uint(128), resp.BlockData[127].Number())

	max = uint32(16)
	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Ascending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(max), len(resp.BlockData))
	require.Equal(t, uint(1), resp.BlockData[0].Number())
	require.Equal(t, uint(16), resp.BlockData[15].Number())

	// test descending
	start = messages.NewFromBlock(uint(128))
	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(128), resp.BlockData[0].Number())
	require.Equal(t, uint(1), resp.BlockData[127].Number())

	max = uint32(messages.MaxBlocksInResponse + 100)
	start = messages.NewFromBlock(uint(256))

	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(256), resp.BlockData[0].Number())
	require.Equal(t, uint(129), resp.BlockData[127].Number())

	max = uint32(16)
	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           &max,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(max), len(resp.BlockData))
	require.Equal(t, uint(256), resp.BlockData[0].Number())
	require.Equal(t, uint(241), resp.BlockData[15].Number())
}

func TestService_CreateBlockResponse_StartHash(t *testing.T) {
	s := newFullSyncService(t)
	addTestBlocksToState(t, uint(messages.MaxBlocksInResponse*2), s.blockState)

	// test ascending with nil endBlockHash
	startHash, err := s.blockState.GetHashByNumber(1)
	require.NoError(t, err)

	req := &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *messages.NewFromBlock(startHash),
		Direction:     messages.Ascending,
		Max:           nil,
	}

	resp, err := s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(1), resp.BlockData[0].Number())
	require.Equal(t, uint(128), resp.BlockData[127].Number())

	// test descending with nil endBlockHash
	startHash, err = s.blockState.GetHashByNumber(16)
	require.NoError(t, err)

	start := messages.NewFromBlock(startHash)

	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(16), len(resp.BlockData))
	require.Equal(t, uint(16), resp.BlockData[0].Number())
	require.Equal(t, uint(1), resp.BlockData[15].Number())

	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(16), len(resp.BlockData))
	require.Equal(t, uint(16), resp.BlockData[0].Number())
	require.Equal(t, uint(1), resp.BlockData[15].Number())

	// test descending with nil endBlockHash and start > messages.MaxBlocksInResponse
	startHash, err = s.blockState.GetHashByNumber(256)
	require.NoError(t, err)

	start = messages.NewFromBlock(startHash)

	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, int(messages.MaxBlocksInResponse), len(resp.BlockData))
	require.Equal(t, uint(256), resp.BlockData[0].Number())
	require.Equal(t, uint(129), resp.BlockData[127].Number())

	startHash, err = s.blockState.GetHashByNumber(128)
	require.NoError(t, err)

	start = messages.NewFromBlock(startHash)

	req = &messages.BlockRequestMessage{
		RequestedData: 3,
		StartingBlock: *start,
		Direction:     messages.Descending,
		Max:           nil,
	}

	resp, err = s.CreateBlockResponse(peer.ID("alice"), req)
	require.NoError(t, err)
	require.Equal(t, messages.MaxBlocksInResponse, len(resp.BlockData))
	require.Equal(t, uint(128), resp.BlockData[0].Number())
	require.Equal(t, uint(1), resp.BlockData[127].Number())
}

func TestService_checkOrGetDescendantHash_integration(t *testing.T) {
	t.Parallel()
	s := newFullSyncService(t)
	branches := map[uint]int{
		8: 1,
	}
	state.AddBlocksToStateWithFixedBranches(t, s.blockState.(*state.BlockState), 16, branches)

	// base case
	ancestor, err := s.blockState.GetHashByNumber(1)
	require.NoError(t, err)
	descendant, err := s.blockState.GetHashByNumber(16)
	require.NoError(t, err)
	const descendantNumber uint = 16

	res, err := s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, descendant, res)

	// supply descendant that's not on canonical chain
	leaves := s.blockState.(*state.BlockState).Leaves()
	require.Equal(t, 2, len(leaves))

	ancestor, err = s.blockState.GetHashByNumber(1)
	require.NoError(t, err)
	descendant, err = s.blockState.GetHashByNumber(descendantNumber)
	require.NoError(t, err)

	for _, leaf := range leaves {
		if leaf != descendant {
			descendant = leaf
			break
		}
	}

	res, err = s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, descendant, res)

	// supply descedant that's not on same chain as ancestor
	ancestor, err = s.blockState.GetHashByNumber(9)
	require.NoError(t, err)
	_, err = s.checkOrGetDescendantHash(ancestor, &descendant, descendantNumber)
	require.Error(t, err)

	// don't supply descendant, should return block on canonical chain
	// as ancestor is on canonical chain
	expected, err := s.blockState.GetHashByNumber(descendantNumber)
	require.NoError(t, err)

	res, err = s.checkOrGetDescendantHash(ancestor, nil, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, expected, res)

	// don't supply descendant and provide ancestor not on canonical chain
	// should return descendant block also not on canonical chain
	block9s, err := s.blockState.GetAllBlocksAtNumber(9)
	require.NoError(t, err)
	canonical, err := s.blockState.GetHashByNumber(9)
	require.NoError(t, err)

	// set ancestor to non-canonical block 9
	for _, block := range block9s {
		if canonical != block {
			ancestor = block
			break
		}
	}

	// expected is non-canonical block 16
	for _, leaf := range leaves {
		is, err := s.blockState.IsDescendantOf(ancestor, leaf)
		require.NoError(t, err)
		if is {
			expected = leaf
			break
		}
	}

	res, err = s.checkOrGetDescendantHash(ancestor, nil, descendantNumber)
	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestService_CreateBlockResponse_Fields(t *testing.T) {
	s := newFullSyncService(t)
	addTestBlocksToState(t, 2, s.blockState)

	bestHash := s.blockState.(*state.BlockState).BestBlockHash()
	bestBlock, err := s.blockState.(*state.BlockState).GetBlockByNumber(1)
	require.NoError(t, err)

	// set some nils and check no error is thrown
	bds := &types.BlockData{
		Hash:          bestHash,
		Header:        nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}
	err = s.blockState.CompareAndSetBlockData(bds)
	require.NoError(t, err)

	// set receipt message and justification
	a := []byte("asdf")
	b := []byte("ghjkl")
	c := []byte("qwerty")
	bds = &types.BlockData{
		Hash:          bestHash,
		Receipt:       &a,
		MessageQueue:  &b,
		Justification: &c,
	}

	start := messages.NewFromBlock(uint(1))
	err = s.blockState.CompareAndSetBlockData(bds)
	require.NoError(t, err)

	testCases := []struct {
		description      string
		value            *messages.BlockRequestMessage
		expectedMsgValue *messages.BlockResponseMessage
	}{
		{
			description: "test get Header and Body",
			value: &messages.BlockRequestMessage{
				RequestedData: 3,
				StartingBlock: *start,
				Direction:     messages.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &messages.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   bestHash,
						Header: &bestBlock.Header,
						Body:   &bestBlock.Body,
					},
				},
			},
		},
		{
			description: "test get Header",
			value: &messages.BlockRequestMessage{
				RequestedData: 1,
				StartingBlock: *start,
				Direction:     messages.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &messages.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:   bestHash,
						Header: &bestBlock.Header,
						Body:   nil,
					},
				},
			},
		},
		{
			description: "test get Receipt",
			value: &messages.BlockRequestMessage{
				RequestedData: 4,
				StartingBlock: *start,
				Direction:     messages.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &messages.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:    bestHash,
						Header:  nil,
						Body:    nil,
						Receipt: bds.Receipt,
					},
				},
			},
		},
		{
			description: "test get MessageQueue",
			value: &messages.BlockRequestMessage{
				RequestedData: 8,
				StartingBlock: *start,
				Direction:     messages.Ascending,
				Max:           nil,
			},
			expectedMsgValue: &messages.BlockResponseMessage{
				BlockData: []*types.BlockData{
					{
						Hash:         bestHash,
						Header:       nil,
						Body:         nil,
						MessageQueue: bds.MessageQueue,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			resp, err := s.CreateBlockResponse(peer.ID("alice"), test.value)
			require.NoError(t, err)
			require.Len(t, resp.BlockData, 2)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Hash, bestHash)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Header, resp.BlockData[0].Header)
			require.Equal(t, test.expectedMsgValue.BlockData[0].Body, resp.BlockData[0].Body)

			if test.expectedMsgValue.BlockData[0].Receipt != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].Receipt, resp.BlockData[1].Receipt)
			}

			if test.expectedMsgValue.BlockData[0].MessageQueue != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].MessageQueue, resp.BlockData[1].MessageQueue)
			}

			if test.expectedMsgValue.BlockData[0].Justification != nil {
				require.Equal(t, test.expectedMsgValue.BlockData[0].Justification, resp.BlockData[1].Justification)
			}
		})
	}
}
