// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"sync"
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/db"
	"github.com/ChainSafe/gossamer/internal/client/mocks"
	statedb "github.com/ChainSafe/gossamer/internal/client/state-db"
	memorykvdb "github.com/ChainSafe/gossamer/internal/kvdb/memory-kvdb"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	primivite_consensus_common "github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/overlayedchanges"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/version"
	"github.com/stretchr/testify/require"
)

type TestClient struct {
	Client[
		hash.H256, runtime.BlakeTwo256, uint64, runtime.OpaqueExtrinsic, ExecutorT, *generic.Header[
			uint64, hash.H256, runtime.BlakeTwo256],
	]
}

var (
	_ api.BlockchainEvents[
		hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	] = &TestClient{}
	_ api.PreCommitActions[
		hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	] = &TestClient{}
	_ api.LockImportRun[
		hash.H256, uint64, runtime.BlakeTwo256, *generic.Header[
			uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic,
	] = &TestClient{}
)

type TestExecutor struct{}

func (e *TestExecutor) Call(
	ext externalities.Extensions,
	runtimeCode core.RuntimeCode,
	method string,
	data []byte,
	context core.CallContext,
) (result []byte, native bool, err error) {
	panic("not implemented")
}

func (e *TestExecutor) RuntimeVersion(
	externalities externalities.Externalities,
	runtimeCode core.RuntimeCode,
) (version.RuntimeVersion, error) {
	panic("not implemented")
}

func NewTestExecutor(t *testing.T) ExecutorT {
	t.Helper()

	return &TestExecutor{}
}

func NewTestBackend(t *testing.T,
	blocksPruning db.BlocksPruning, canonicalizationDelay uint64,
) api.Backend[
	hash.H256,
	uint64,
	runtime.BlakeTwo256,
	*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	runtime.OpaqueExtrinsic,
] {
	t.Helper()

	kvdb := memorykvdb.New(13)
	var statePruning statedb.PruningMode
	switch blocksPruning := blocksPruning.(type) {
	case db.BlocksPruningKeepAll:
		statePruning = statedb.PruningModeArchiveAll{}
	case db.BlocksPruningKeepFinalized:
		statePruning = statedb.PruningModeArchiveCanonical{}
	case db.BlocksPruningSome:
		statePruning = statedb.NewPruningModeConstrained(uint32(blocksPruning))
	default:
		t.Fatalf("unreachable")
	}
	trieCacheMaxSize := uint(16 * 1024 * 1024)
	dbSetting := db.DatabaseConfig{
		TrieCacheMaximumSize: &trieCacheMaxSize,
		StatePruning:         statePruning,
		Source:               db.DatabaseSource{DB: database.NewDBAdapter[hash.H256](kvdb), RequireCreateFlag: true},
		BlocksPruning:        blocksPruning,
	}

	backend, err := db.NewBackend[
		hash.H256,
		uint64,
		runtime.OpaqueExtrinsic,
		runtime.BlakeTwo256,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	](dbSetting, canonicalizationDelay)
	if err != nil {
		panic(err)
	}
	return backend
}

func TestNew(t *testing.T) {
	c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
	require.NotNil(t, c)
}

type BlockImportOperation = api.BlockImportNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]] //nolint:lll
type FinalityNotification = api.FinalityNotification[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]    //nolint:lll

func TestBlockchainEvents(t *testing.T) {
	t.Run("register_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
		blockImport := c.RegisterImportNotificationStream()
		require.NotNil(t, blockImport)
		_, ok := c.importNotificationChans[blockImport]
		require.True(t, ok)

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for range blockImport {
			}
			wg.Done()
		}()

		everyImport := c.RegisterEveryImportNotificationStream()
		require.NotNil(t, everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for range everyImport {
			}
			wg.Done()
		}()

		finality := c.RegisterFinalityNotificationStream()
		require.NotNil(t, finality)
		_, ok = c.finalityNotificationChans[finality]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for range finality {
			}
			wg.Done()
		}()

		c.UnregisterImportNotificationStream(blockImport)
		_, ok = c.importNotificationChans[blockImport]
		require.False(t, ok)

		c.UnregisterEveryImportNotificationStream(everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.False(t, ok)

		c.UnregisterFinalityNotificationStream(finality)
		_, ok = c.finalityNotificationChans[finality]
		require.False(t, ok)

		wg.Wait()
	})

	t.Run("register_receive_block_import_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
		blockImport := c.RegisterImportNotificationStream()
		require.NotNil(t, blockImport)
		_, ok := c.importNotificationChans[blockImport]
		require.True(t, ok)

		var blockImportNotifications []BlockImportOperation
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for notif := range blockImport {
				blockImportNotifications = append(blockImportNotifications, notif)
			}
			wg.Done()
		}()

		var everyImportNotifications []BlockImportOperation
		everyImport := c.RegisterEveryImportNotificationStream()
		require.NotNil(t, everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.True(t, ok)

		wg.Add(1)
		go func() {
			for notif := range everyImport {
				everyImportNotifications = append(everyImportNotifications, notif)
			}
			wg.Done()
		}()

		// sends to both
		c.notifyImported(&BlockImportOperation{}, api.BothBlockImportNotificationAction, nil)
		// sends to import
		c.notifyImported(&BlockImportOperation{}, api.RecentBlockImportNotificationAction, nil)
		// sends to every
		c.notifyImported(&BlockImportOperation{}, api.EveryBlockImportNotificationAction, nil)

		c.UnregisterImportNotificationStream(blockImport)
		_, ok = c.importNotificationChans[blockImport]
		require.False(t, ok)

		c.UnregisterEveryImportNotificationStream(everyImport)
		_, ok = c.everyImportNotificationChans[everyImport]
		require.False(t, ok)

		wg.Wait()

		require.Len(t, blockImportNotifications, 2)
		require.Len(t, everyImportNotifications, 2)
	})

	t.Run("register_receive_block_import_storage_changes_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
		wg := sync.WaitGroup{}

		topStorage := c.StorageChangesNotificationStream(nil, []api.ChildFilterKeys{})
		wg.Add(1)
		go func() {
			msg := <-topStorage.Chan()
			require.Len(t, msg.Changes, 1)
			require.Len(t, msg.ChildChanges, 0)
			wg.Done()
		}()

		childStorage := c.StorageChangesNotificationStream([]storage.StorageKey{}, []api.ChildFilterKeys{
			{Key: storage.StorageKey("child0"), FilterKeys: []storage.StorageKey{storage.StorageKey("child0")}},
		})
		wg.Add(1)
		go func() {
			msg := <-childStorage.Chan()
			require.Len(t, msg.Changes, 0)
			require.Len(t, msg.ChildChanges, 1)
			wg.Done()
		}()

		wildCard := c.StorageChangesNotificationStream(nil, []api.ChildFilterKeys{
			{Key: storage.StorageKey("child0"), FilterKeys: []storage.StorageKey{storage.StorageKey("child0")}},
		})
		wg.Add(1)
		go func() {
			msg := <-wildCard.Chan()
			require.Len(t, msg.Changes, 1)
			require.Len(t, msg.ChildChanges, 1)
			wg.Done()
		}()

		// sends to both
		c.notifyImported(
			&BlockImportOperation{},
			api.BothBlockImportNotificationAction,
			&api.StorageChanges{
				StorageCollection: overlayedchanges.StorageCollection{
					{StorageKey: overlayedchanges.StorageKey("top0"), StorageValue: overlayedchanges.StorageValue("top0")},
				},
				ChildStorageCollection: []struct {
					overlayedchanges.StorageKey
					overlayedchanges.StorageCollection
				}{
					{
						StorageKey: overlayedchanges.StorageKey("child0"),
						StorageCollection: overlayedchanges.StorageCollection{
							{StorageKey: overlayedchanges.StorageKey("child0"), StorageValue: overlayedchanges.StorageValue("child0")},
						},
					},
				},
			},
		)

		wg.Wait()
		topStorage.Drop()
		childStorage.Drop()
	})

	t.Run("register_receive_finality_unregister", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
		finality := c.RegisterFinalityNotificationStream()
		require.NotNil(t, finality)
		_, ok := c.finalityNotificationChans[finality]
		require.True(t, ok)

		var finalityNotifications []FinalityNotification
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			for notif := range finality {
				finalityNotifications = append(finalityNotifications, notif)
			}
			wg.Done()
		}()

		c.notifyFinalized(&FinalityNotification{})

		c.UnregisterFinalityNotificationStream(finality)
		_, ok = c.finalityNotificationChans[finality]
		require.False(t, ok)

		wg.Wait()

		require.Len(t, finalityNotifications, 1)
	})
}

type ClientImportOperation = api.ClientImportOperation[
	hash.H256,
	runtime.BlakeTwo256,
	uint64,
	*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
	runtime.OpaqueExtrinsic,
]

func TestLockImportRun(t *testing.T) {
	c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
	_, err := c.LockImportRun(func(cio *ClientImportOperation) (any, error) {
		return nil, nil
	})
	require.NoError(t, err)
}

func TestPreCommitActions(t *testing.T) {
	t.Run("register_import_and_finality_actions", func(t *testing.T) {
		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))

		var count int
		c.RegisterImportAction(func(bin BlockImportOperation) api.AuxDataOperations {
			count++
			return api.AuxDataOperations{}
		})
		c.RegisterFinalityAction(func(fn FinalityNotification) api.AuxDataOperations {
			count++
			return api.AuxDataOperations{}
		})

		_, err := c.LockImportRun(func(cio *ClientImportOperation) (any, error) {
			cio.NotifyFinalized = &api.FinalizeSummary[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{} //nolint:lll
			cio.NotifyImported = &api.ImportSummary[hash.H256, uint64, *generic.Header[uint64, hash.H256, runtime.BlakeTwo256]]{}    //nolint:lll
			return nil, nil
		})
		require.NoError(t, err)

		require.Equal(t, 2, count)
	})
}

func TestHeaderBackendImplementation(t *testing.T) {
	backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
		runtime.OpaqueExtrinsic](t)

	blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

	expectedHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256("extrinsicsroot"),
		hash.H256("stateroot"),
		hash.H256("parent"),
		runtime.Digest{},
	)
	expectedHash := expectedHeader.Hash()
	expectedNumber := expectedHeader.Number()

	blockchainMock.EXPECT().Header(expectedHash).Return(&expectedHeader, nil)
	blockchainMock.EXPECT().Number(expectedHash).Return(&expectedNumber, nil)
	blockchainMock.EXPECT().Hash(expectedNumber).Return(&expectedHash, nil)

	expectedExtrinsics := []runtime.OpaqueExtrinsic{}
	blockchainMock.EXPECT().Body(expectedHash).Return(expectedExtrinsics, nil)

	expectedInfo := blockchain.Info[hash.H256, uint64]{
		BestHash:        expectedHash,
		BestNumber:      expectedHeader.Number(),
		GenesisHash:     expectedHeader.ParentHash(),
		FinalizedHash:   expectedHash,
		FinalizedNumber: expectedHeader.Number(),
		FinalizedState:  nil,
	}
	blockchainMock.EXPECT().Info().Return(expectedInfo)

	expectedStatus := blockchain.BlockStatusInChain
	blockchainMock.EXPECT().Status(expectedHash).Return(expectedStatus, nil)

	blockNumberId := generic.NewBlockID[hash.H256, uint64](generic.BlockIDNumber[uint64]{Number: expectedNumber})
	blockHashId := generic.NewBlockID[hash.H256, uint64](generic.BlockIDHash[hash.H256]{Hash: expectedHash})
	blockchainMock.EXPECT().BlockHashFromID(blockNumberId).Return(&expectedHash, nil)
	blockchainMock.EXPECT().BlockNumberFromID(blockHashId).Return(&expectedNumber, nil)

	backendMock.EXPECT().Blockchain().Return(blockchainMock)

	c := New(backendMock, NewTestExecutor(t))

	// Get Header
	header, err := c.Header(expectedHash)
	require.NoError(t, err)
	require.NotNil(t, header)
	require.Equal(t, expectedHeader, *header)

	// Get Body
	extrinsics, err := c.Body(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedExtrinsics, extrinsics)

	// Get Info
	info := c.Info()
	require.Equal(t, expectedInfo, info)

	// Get Status
	status, err := c.Status(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)

	// Get Number
	number, err := c.Number(expectedHash)
	require.NoError(t, err)
	require.NotNil(t, number)
	require.Equal(t, expectedNumber, *number)

	// Get Number
	hash, err := c.Hash(expectedNumber)
	require.NoError(t, err)
	require.NotNil(t, hash)
	require.Equal(t, expectedHash, *hash)

	// Get BlockHashFromID
	blockHash, err := c.BlockHashFromID(blockNumberId)
	require.NoError(t, err)
	require.NotNil(t, blockHash)
	require.Equal(t, expectedHash, *blockHash)

	// Get BlockNumberFromId
	blockNumber, err := c.BlockNumberFromID(blockHashId)
	require.NoError(t, err)
	require.NotNil(t, blockNumber)
	require.Equal(t, expectedNumber, *blockNumber)
}

func TestBlockBackendImplementation(t *testing.T) {
	backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
		runtime.OpaqueExtrinsic](t)

	blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
		*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

	c := New(backendMock, NewTestExecutor(t))

	expectedHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256("extrinsicsroot"),
		hash.H256("stateroot"),
		hash.H256("parent"),
		runtime.Digest{},
	)
	expectedHash := expectedHeader.Hash()
	expectedNumber := expectedHeader.Number()

	blockchainMock.EXPECT().Number(expectedHash).Return(&expectedNumber, nil)
	blockchainMock.EXPECT().Header(expectedHash).Return(&expectedHeader, nil)
	blockchainMock.EXPECT().Hash(expectedNumber).Return(&expectedHash, nil)

	expectedExtrinsics := []runtime.OpaqueExtrinsic{}
	blockchainMock.EXPECT().Body(expectedHash).Return(expectedExtrinsics, nil)

	expectedStatus := primivite_consensus_common.BlockStatusInChainWithState

	expectedIndexedExtrinsics := [][]byte{
		[]byte("extrinsic1"),
		[]byte("extrinsic2"),
	}

	blockchainMock.EXPECT().BlockIndexedBody(expectedHash).Return(expectedIndexedExtrinsics, nil)

	var expectedJustifications runtime.Justifications = nil
	blockchainMock.EXPECT().Justifications(expectedHash).Return(expectedJustifications, nil)

	expectedIndexedTransaction := []byte("transaction1")
	blockchainMock.EXPECT().IndexedTransaction(expectedHash).Return(expectedIndexedTransaction, nil)
	blockchainMock.EXPECT().HasIndexedTransaction(expectedHash).Return(true, nil)

	backendMock.EXPECT().RequiresFullSync().Return(true)
	backendMock.EXPECT().Blockchain().Return(blockchainMock)
	backendMock.EXPECT().HaveStateAt(expectedHash, expectedNumber).Return(true)

	// Get BlockBody
	extrinsics, err := c.BlockBody(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedExtrinsics, extrinsics)

	// Get BlockIndexedBody
	indexedBody, err := c.BlockIndexedBody(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedIndexedExtrinsics, indexedBody)

	// Get Block
	expectedBlock := generic.NewSignedBlock(
		generic.NewBlock[runtime.BlakeTwo256](expectedHeader, expectedExtrinsics), nil,
	)
	block, err := c.Block(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedBlock, block)

	// Get BlockStatus
	blockStatus, err := c.BlockStatus(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, blockStatus)

	// Get Justifications
	justifications, err := c.Justifications(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedJustifications, justifications)

	// Get BlockHash
	blockHash, err := c.BlockHash(expectedNumber)
	require.NoError(t, err)
	require.NotNil(t, blockHash)
	require.Equal(t, expectedHash, *blockHash)

	// Get IndexedTransaction
	indexedTransaction, err := c.IndexedTransaction(expectedHash)
	require.NoError(t, err)
	require.Equal(t, expectedIndexedTransaction, indexedTransaction)

	// HasIndexedTransactions
	has, err := c.HasIndexedTransaction(expectedHash)
	require.NoError(t, err)
	require.True(t, has)

	// RequiresFullSync
	requiresFullSync := c.RequiresFullSync()
	require.NoError(t, err)
	require.True(t, requiresFullSync)
}

func TestCheckBlock(t *testing.T) {
	badBlock := common.BlockCheckParams[hash.H256, uint64]{
		Number: 1,
		Hash:   hash.H256("bad_block"),
	}

	invalidForkBlock := common.BlockCheckParams[hash.H256, uint64]{
		Number: 2,
		Hash:   hash.H256("block_2_hash"),
	}

	c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))

	c.blockRules = BlockRules[hash.H256, uint64]{
		bad: map[hash.H256]struct{}{
			badBlock.Hash: {},
		},
		forks: map[uint64]hash.H256{
			invalidForkBlock.Number: hash.H256("invalid_fork"),
		},
	}

	t.Run("reject_known_bad_block", func(t *testing.T) {
		result, err := c.CheckBlock(badBlock)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultKnownBad{}, result)
	})

	t.Run("reject_block_from_invalid_fork", func(t *testing.T) {
		result, err := c.CheckBlock(invalidForkBlock)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultKnownBad{}, result)
	})

	t.Run("queued_block_already_in_chain", func(t *testing.T) {
		queuedBlock := common.BlockCheckParams[hash.H256, uint64]{
			Number: 3,
			Hash:   hash.H256("queued_block"),
		}

		c := New(NewTestBackend(t, db.BlocksPruningKeepFinalized{}, 0), NewTestExecutor(t))
		c.importingBlock = &queuedBlock.Hash

		result, err := c.CheckBlock(queuedBlock)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultAlreadyInChain{}, result)
	})

	t.Run("in_chain_with_state", func(t *testing.T) {
		importedBlockWithStatus := common.BlockCheckParams[hash.H256, uint64]{
			Number: 4,
			Hash:   hash.H256("status_imported"),
		}

		backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
			runtime.OpaqueExtrinsic](t)

		blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

		blockchainMock.EXPECT().Number(importedBlockWithStatus.Hash).Return(&importedBlockWithStatus.Number, nil)

		backendMock.EXPECT().Blockchain().Return(blockchainMock)
		backendMock.EXPECT().HaveStateAt(importedBlockWithStatus.Hash, importedBlockWithStatus.Number).Return(true)

		c := New(backendMock, NewTestExecutor(t))
		result, err := c.CheckBlock(importedBlockWithStatus)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultAlreadyInChain{}, result)
	})

	t.Run("pruned_block_not_import_existing", func(t *testing.T) {
		prunedBlock := common.BlockCheckParams[hash.H256, uint64]{
			Number:         5,
			Hash:           hash.H256("pruned_block"),
			ImportExisting: false,
		}

		backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
			runtime.OpaqueExtrinsic](t)

		blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

		blockchainMock.EXPECT().Number(prunedBlock.Hash).Return(&prunedBlock.Number, nil)

		backendMock.EXPECT().Blockchain().Return(blockchainMock)
		backendMock.EXPECT().HaveStateAt(prunedBlock.Hash, prunedBlock.Number).Return(false)

		c := New(backendMock, NewTestExecutor(t))

		result, err := c.CheckBlock(prunedBlock)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultAlreadyInChain{}, result)
	})

	t.Run("unknown_parent", func(t *testing.T) {
		blockUnknownParent := common.BlockCheckParams[hash.H256, uint64]{
			Number:     6,
			Hash:       hash.H256("ok_block"),
			ParentHash: hash.H256("unknown_block"),
		}

		backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
			runtime.OpaqueExtrinsic](t)

		blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

		blockchainMock.EXPECT().Number(blockUnknownParent.Hash).Return(nil, nil)
		blockchainMock.EXPECT().Number(blockUnknownParent.ParentHash).Return(nil, nil)

		backendMock.EXPECT().Blockchain().Return(blockchainMock)

		c := New(backendMock, NewTestExecutor(t))

		result, err := c.CheckBlock(blockUnknownParent)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultUnknownParent{}, result)
	})

	t.Run("parent_pruned", func(t *testing.T) {
		blockUnknownParent := common.BlockCheckParams[hash.H256, uint64]{
			Number:     6,
			Hash:       hash.H256("ok_block"),
			ParentHash: hash.H256("pruned_block"),
		}

		backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
			runtime.OpaqueExtrinsic](t)

		blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

		blockchainMock.EXPECT().Number(blockUnknownParent.Hash).Return(nil, nil)

		parentHash := blockUnknownParent.ParentHash
		parentNumber := blockUnknownParent.Number - 1

		blockchainMock.EXPECT().Number(parentHash).Return(&parentNumber, nil)

		backendMock.EXPECT().HaveStateAt(parentHash, parentNumber).Return(false)

		backendMock.EXPECT().Blockchain().Return(blockchainMock)

		c := New(backendMock, NewTestExecutor(t))

		result, err := c.CheckBlock(blockUnknownParent)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultMissingState{}, result)
	})

	t.Run("block_ok", func(t *testing.T) {
		blockUnknownParent := common.BlockCheckParams[hash.H256, uint64]{
			Number:     6,
			Hash:       hash.H256("ok_block"),
			ParentHash: hash.H256("pruned_block"),
		}

		backendMock := mocks.NewBackend[hash.H256, uint64, runtime.BlakeTwo256,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256],
			runtime.OpaqueExtrinsic](t)

		blockchainMock := mocks.NewBlockchainBackend[hash.H256, uint64,
			*generic.Header[uint64, hash.H256, runtime.BlakeTwo256], runtime.OpaqueExtrinsic](t)

		blockchainMock.EXPECT().Number(blockUnknownParent.Hash).Return(nil, nil)

		parentHash := blockUnknownParent.ParentHash
		parentNumber := blockUnknownParent.Number - 1

		blockchainMock.EXPECT().Number(parentHash).Return(&parentNumber, nil)
		backendMock.EXPECT().HaveStateAt(parentHash, parentNumber).Return(true)
		backendMock.EXPECT().Blockchain().Return(blockchainMock)

		c := New(backendMock, NewTestExecutor(t))

		result, err := c.CheckBlock(blockUnknownParent)
		require.NoError(t, err)
		require.Equal(t, common.ImportResultImported{IsNewBest: false}, result)
	})
}
