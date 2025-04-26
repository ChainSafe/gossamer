// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	chainspec "github.com/ChainSafe/gossamer/internal/client/chain-spec"
	"github.com/ChainSafe/gossamer/internal/client/consensus"
	"github.com/ChainSafe/gossamer/internal/client/consensus/common"
	"github.com/ChainSafe/gossamer/internal/client/executor"
	"github.com/ChainSafe/gossamer/internal/log"
	primitives_api "github.com/ChainSafe/gossamer/internal/primitives/api"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	primivite_consensus_common "github.com/ChainSafe/gossamer/internal/primitives/consensus/common"
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "client"))

type BadBlocks[H runtime.Hash] map[H]struct{}

type prepareStorageChangesResult interface {
	isPrepareStorageChangesResult()
}

type (
	prepareStorageChangesResultDiscard struct {
		common.ImportResult
	}
	prepareStorageChangesResultImport struct {
		common.StorageChanges
	}
)

func (prepareStorageChangesResultDiscard) isPrepareStorageChangesResult() {}
func (prepareStorageChangesResultImport) isPrepareStorageChangesResult()  {}

// Used in importing a block, where additional changes are made after the runtime executed.
type PrePostHeaders[N runtime.Number, H runtime.Hash, Header runtime.Header[N, H]] interface {
	Post() Header
}

type (
	// they are the same: no post-runtime digest items.
	PrePostHeadersSame[N runtime.Number, H runtime.Hash, Header runtime.Header[N, H]] struct {
		Header Header
	}
	// different headers (pre, post).
	PrePostHeadersDifferent[N runtime.Number, H runtime.Hash, Header runtime.Header[N, H]] struct {
		PreHeader  Header
		PostHeader Header
	}
)

func (pph PrePostHeadersSame[N, H, Header]) Post() Header      { return pph.Header }
func (pph PrePostHeadersDifferent[N, H, Header]) Post() Header { return pph.PostHeader }

// Client configuration items.
type ClientConfig[N runtime.Number] struct {
	// Enable the offchain worker db.
	OffchainWorkerEnabled bool
	// If true, allows access from the runtime to write into offchain worker db.
	OffchainIndexingAPI bool
	// Path where WASM files exist to override the on-chain WASM.
	WasmRuntimeOverrides *string
	// Skip writing genesis state on first start.
	NoGenesis bool
	// Map of WASM runtime substitute starting at the child of the given block until the runtime
	// version doesn't match anymore.
	WasmRuntimeSubstitutes map[N][]byte
	// Enable recording of storage proofs during block import
	EnableImportProofRecording bool
}

// Client type that implements a number of client interfaces
type Client[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Executor ExecutorT,
	Header runtime.Header[N, H],
	RA primitives_api.ConstructRuntimeApi[
		N, E, H, Hasher,
		statemachine.Backend[H, Hasher],
		any,
		primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any],
	],
] struct {
	backend                         api.Backend[H, N, Hasher, Header, E]
	executor                        Executor
	storageNotifications            api.StorageNotifications[H]
	importNotificationChansMtx      sync.Mutex
	importNotificationChans         map[chan api.BlockImportNotification[H, N, Header]]any
	everyImportNotificationChansMtx sync.Mutex
	everyImportNotificationChans    map[chan api.BlockImportNotification[H, N, Header]]any
	finalityNotificationChansMtx    sync.Mutex
	finalityNotificationChans       map[chan api.FinalityNotification[H, N, Header]]any
	// Collects auxiliary operations to be performed atomically together with block import operations.
	importActionsMtx sync.Mutex
	importActions    []api.OnImportAction[H, N, Header]
	// Collects auxiliary operations to be performed atomically together with block finalization operations.
	finalityActionsMtx sync.Mutex
	finalityActions    []api.OnFinalityAction[H, N, Header]
	// Holds the block hash currently being imported.
	importingBlockMtx  sync.RWMutex
	importingBlock     *H
	unpinWorkerChan    chan<- api.UnpinWorkerMessage[H]
	blockRules         BlockRules[H, N]
	config             ClientConfig[N]
	runtimeConstructor RA
}

type ExecutorT interface {
	core.CodeExecutor
	executor.RuntimeVersionOf
}

// New is constructor for [Client]
func New[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Executor ExecutorT,
	Header runtime.Header[N, H],
	RA primitives_api.ConstructRuntimeApi[N, E, H, Hasher,
		statemachine.Backend[H, Hasher],
		any,
		primitives_api.ApiExt[N, E, H, Hasher, statemachine.Backend[H, Hasher], any],
	],
](
	backend api.Backend[H, N, Hasher, Header, E],
	config ClientConfig[N],
	executor Executor,
	runtimeConstructor RA,
) *Client[H, Hasher, N, E, Executor, Header, RA] {
	unpinWorkerChan := make(chan api.UnpinWorkerMessage[H])
	npw := newNotificationPinningWorker(unpinWorkerChan, backend)
	go npw.run()

	return &Client[H, Hasher, N, E, Executor, Header, RA]{
		backend:                      backend,
		executor:                     executor,
		storageNotifications:         api.NewStorageNotifications[H](),
		importNotificationChans:      make(map[chan api.BlockImportNotification[H, N, Header]]any),
		everyImportNotificationChans: make(map[chan api.BlockImportNotification[H, N, Header]]any),
		finalityNotificationChans:    make(map[chan api.FinalityNotification[H, N, Header]]any),
		unpinWorkerChan:              unpinWorkerChan,
		config:                       config,
		runtimeConstructor:           runtimeConstructor,
	}
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) announcePin(message api.AnnouncePin[H]) error {
	select {
	case c.unpinWorkerChan <- message:
		return nil
	default:
		return fmt.Errorf("unable to send AnnouncePin message to Client.unpinWorkerChan")
	}
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) unpin(message api.Unpin[H]) error {
	select {
	case c.unpinWorkerChan <- message:
		return nil
	default:
		return fmt.Errorf("unable to send Unpin message to Client.unpinWorkerChan")
	}
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) lockImportRun(
	f func(*api.ClientImportOperation[H, Hasher, N, Header, E]) (any, error),
) (any, error) {
	c.backend.GetImportLock().Lock()
	defer c.backend.GetImportLock().Unlock()

	blockImportOp, err := c.backend.BeginOperation()
	if err != nil {
		return nil, err
	}

	clientImportOp := api.ClientImportOperation[H, Hasher, N, Header, E]{
		Op: blockImportOp,
	}

	result, err := f(&clientImportOp)
	if err != nil {
		return nil, err
	}

	var finalityNotification *api.FinalityNotification[H, N, Header]
	if clientImportOp.NotifyFinalized != nil {
		finalityNotification = api.NewFinalityNotificationFromSummary(*clientImportOp.NotifyFinalized, c.unpin)
	}

	var (
		importNotification       *api.BlockImportNotification[H, N, Header]
		storageChanges           *api.StorageChanges
		importNotificationAction api.ImportNotificationAction
	)
	if clientImportOp.NotifyImported != nil {
		importNotification = api.NewBlockImportNotificationFromSummary(*clientImportOp.NotifyImported, c.unpin)
		storageChanges = clientImportOp.NotifyImported.StorageChanges
		importNotificationAction = clientImportOp.NotifyImported.ImportNotificationAction
	} else {
		importNotificationAction = api.NoneBlockImportNotificationAction
	}

	if finalityNotification != nil {
		c.finalityActionsMtx.Lock()
		defer c.finalityActionsMtx.Unlock()
		for _, action := range c.finalityActions {
			err := clientImportOp.Op.InsertAux(action(*finalityNotification))
			if err != nil {
				return nil, err
			}
		}
	}
	if importNotification != nil {
		c.importActionsMtx.Lock()
		defer c.importActionsMtx.Unlock()
		for _, action := range c.importActions {
			err := clientImportOp.Op.InsertAux(action(*importNotification))
			if err != nil {
				return nil, err
			}
		}
	}

	err = c.backend.CommitOperation(clientImportOp.Op)
	if err != nil {
		return nil, err
	}

	// We need to pin the block in the backend once
	// for each notification. Once all notifications are
	// dropped, the block will be unpinned automatically.
	if finalityNotification != nil {
		err := c.backend.PinBlock(finalityNotification.Hash)
		if err != nil {
			logger.Debugf("Unable to pin block for finality notification. hash: %s, Error: %v",
				finalityNotification.Hash, err)
		} else {
			err := c.announcePin(api.AnnouncePin[H]{Hash: finalityNotification.Hash})
			if err != nil {
				logger.Errorf("Unable to send AnnouncePin worker message for finality: %s", err)
			}
		}
	}

	if importNotification != nil {
		err := c.backend.PinBlock(importNotification.Hash)
		if err != nil {
			logger.Debugf("Unable to pin block for import notification. hash: %s, Error: %v",
				importNotification.Hash, err)
		} else {
			err := c.announcePin(api.AnnouncePin[H]{Hash: importNotification.Hash})
			if err != nil {
				logger.Errorf("Unable to send AnnouncePin worker message for import: %s", err)
			}
		}
	}

	err = c.notifyFinalized(finalityNotification)
	if err != nil {
		return nil, err
	}
	err = c.notifyImported(importNotification, importNotificationAction, storageChanges)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) LockImportRun(
	f func(*api.ClientImportOperation[H, Hasher, N, Header, E]) (any, error),
) (any, error) {
	result, err := c.lockImportRun(f)
	c.importingBlockMtx.Lock()
	c.importingBlock = nil
	c.importingBlockMtx.Unlock()
	return result, err
}

const notifyFinalizedTimeout = 5 * time.Second
const notifyBlockImportTimeout = notifyFinalizedTimeout

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) notifyFinalized(
	notification *api.FinalityNotification[H, N, Header],
) error {
	c.finalityNotificationChansMtx.Lock()
	defer c.finalityNotificationChansMtx.Unlock()

	if notification == nil {
		return nil
	}

	// TODO: telemetry is implemented here.  See substrate code:
	// https://github.com/paritytech/polkadot-sdk/blob/72fb8bd3cd4a5051bb855415b360657d7ce247fb/substrate/client/service/src/client/client.rs#L984

	wg := sync.WaitGroup{}
	for ch := range c.finalityNotificationChans {
		wg.Add(1)
		go func(ch chan<- api.FinalityNotification[H, N, Header]) {
			defer wg.Done()
			ch <- *notification
		}(ch)
	}
	done := make(chan any)
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		break
	case <-time.After(notifyFinalizedTimeout):
		break
	}

	return nil
}

func notifyChans[M any](msg M, chans map[chan M]any, timeout time.Duration) {
	wg := sync.WaitGroup{}
	for ch := range chans {
		wg.Add(1)
		go func(ch chan M) {
			defer wg.Done()
			select {
			case ch <- msg:
			default:
				// cleanup chan if not able to send
				close(ch)
				delete(chans, ch)
			}
		}(ch)
	}
	done := make(chan any)
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		break
	case <-time.After(timeout):
		break
	}
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) notifyImported(
	notification *api.BlockImportNotification[H, N, Header],
	importNotificationAction api.ImportNotificationAction,
	storageChanges *api.StorageChanges,
) error {
	if notification == nil {
		return nil
	}

	var triggerStorageChangesNotification = func() {
		if storageChanges != nil {
			// TODO [ToDr] How to handle re-orgs? Should we re-emit all storage changes? (from substrate)
			changeset := make([]api.StorageChange, len(storageChanges.StorageCollection))
			for i, kv := range storageChanges.StorageCollection {
				changeset[i] = api.StorageChange{
					StorageKey:  storage.StorageKey(kv.StorageKey),
					StorageData: storage.StorageData(kv.StorageValue),
				}
			}
			childChangeset := make([]api.StorageChildChange, len(storageChanges.ChildStorageCollection))
			for i, kc := range storageChanges.ChildStorageCollection {
				changeset := make([]api.StorageChange, len(kc.StorageCollection))
				for i, kv := range kc.StorageCollection {
					changeset[i] = api.StorageChange{
						StorageKey:  storage.StorageKey(kv.StorageKey),
						StorageData: storage.StorageData(kv.StorageValue),
					}
				}
				childChangeset[i] = api.StorageChildChange{
					StorageKey: storage.StorageKey(kc.StorageKey),
					ChangeSet:  changeset,
				}
			}
			c.storageNotifications.Trigger(
				notification.Hash,
				changeset,
				childChangeset,
			)
		}
	}

	switch importNotificationAction {
	case api.BothBlockImportNotificationAction:
		triggerStorageChangesNotification()
		c.importNotificationChansMtx.Lock()
		defer c.importNotificationChansMtx.Unlock()
		notifyChans(*notification, c.importNotificationChans, notifyBlockImportTimeout)

		c.everyImportNotificationChansMtx.Lock()
		defer c.everyImportNotificationChansMtx.Unlock()
		notifyChans(*notification, c.everyImportNotificationChans, notifyBlockImportTimeout)
	case api.RecentBlockImportNotificationAction:
		triggerStorageChangesNotification()
		c.importNotificationChansMtx.Lock()
		defer c.importNotificationChansMtx.Unlock()
		notifyChans(*notification, c.importNotificationChans, notifyBlockImportTimeout)
	case api.EveryBlockImportNotificationAction:
		c.everyImportNotificationChansMtx.Lock()
		defer c.everyImportNotificationChansMtx.Unlock()
		notifyChans(*notification, c.everyImportNotificationChans, notifyBlockImportTimeout)
	case api.NoneBlockImportNotificationAction:
		// This branch is unreachable in fact because the block import notification must be
		// not nil (it's already handled at the beginning of this function) at this point.
	default:
		panic("unreachable")
	}

	return nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RegisterImportAction(op api.OnImportAction[H, N, Header]) {
	c.importActionsMtx.Lock()
	defer c.importActionsMtx.Unlock()
	c.importActions = append(c.importActions, op)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RegisterFinalityAction(op api.OnFinalityAction[H, N, Header]) {
	c.finalityActionsMtx.Lock()
	defer c.finalityActionsMtx.Unlock()
	c.finalityActions = append(c.finalityActions, op)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RegisterImportNotificationStream() api.ImportNotifications[
	H, N, Header] {
	ch := make(chan api.BlockImportNotification[H, N, Header])
	c.importNotificationChansMtx.Lock()
	defer c.importNotificationChansMtx.Unlock()
	c.importNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) UnregisterImportNotificationStream(
	ch api.ImportNotifications[H, N, Header],
) {
	c.importNotificationChansMtx.Lock()
	defer c.importNotificationChansMtx.Unlock()
	_, ok := c.importNotificationChans[ch]
	if ok {
		close(ch)
	}
	delete(c.importNotificationChans, ch)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RegisterEveryImportNotificationStream() api.ImportNotifications[
	H, N, Header] {
	ch := make(chan api.BlockImportNotification[H, N, Header])
	c.everyImportNotificationChansMtx.Lock()
	defer c.everyImportNotificationChansMtx.Unlock()
	c.everyImportNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) UnregisterEveryImportNotificationStream(
	ch api.ImportNotifications[H, N, Header],
) {
	c.everyImportNotificationChansMtx.Lock()
	defer c.everyImportNotificationChansMtx.Unlock()
	_, ok := c.everyImportNotificationChans[ch]
	if ok {
		close(ch)
	}
	delete(c.everyImportNotificationChans, ch)
}

func (c *Client[H, _, N, E, Executor, Header, RA]) RegisterFinalityNotificationStream() api.FinalityNotifications[
	H, N, Header] {
	ch := make(chan api.FinalityNotification[H, N, Header])
	c.finalityNotificationChansMtx.Lock()
	defer c.finalityNotificationChansMtx.Unlock()
	c.finalityNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) UnregisterFinalityNotificationStream(
	ch api.FinalityNotifications[H, N, Header],
) {
	c.finalityNotificationChansMtx.Lock()
	defer c.finalityNotificationChansMtx.Unlock()
	_, ok := c.finalityNotificationChans[ch]
	if ok {
		close(ch)
	}
	delete(c.finalityNotificationChans, ch)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) StorageChangesNotificationStream(
	filterKeys []storage.StorageKey,
	childFilterKeys []api.ChildFilterKeys,
) api.StorageEventStream[H] {
	return c.storageNotifications.Listen(filterKeys, childFilterKeys)
}

// HeaderBackend implementation for Client

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Header(hash H) (*Header, error) {
	return c.backend.Blockchain().Header(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Body(hash H) ([]E, error) {
	return c.backend.Blockchain().Body(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Info() blockchain.Info[H, N] {
	return c.backend.Blockchain().Info()
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Status(hash H) (blockchain.BlockStatus, error) {
	return c.backend.Blockchain().Status(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Number(hash H) (*N, error) {
	return c.backend.Blockchain().Number(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Hash(number N) (*H, error) {
	return c.backend.Blockchain().Hash(number)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockHashFromID(id generic.BlockID) (*H, error) {
	return c.backend.Blockchain().BlockHashFromID(id)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockNumberFromID(id generic.BlockID) (*N, error) {
	return c.backend.Blockchain().BlockNumberFromID(id)
}

// BlockBackend implementation for Client

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockBody(hash H) ([]E, error) {
	return c.backend.Blockchain().Body(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Block(hash H) (*generic.SignedBlock[N, H, Hasher, E], error) {
	header, err := c.Header(hash)
	if err != nil {
		return nil, err
	}

	body, err := c.Body(hash)
	if err != nil {
		return nil, err
	}

	justifications, err := c.Justifications(hash)
	if err != nil {
		return nil, err
	}

	if header != nil && body != nil {
		return generic.NewSignedBlock(
			generic.NewBlock[Hasher](*header, body), justifications,
		), nil
	}

	return nil, nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockStatus(hash H) (
	primivite_consensus_common.BlockStatus, error,
) {
	c.importingBlockMtx.RLock()
	if c.importingBlock != nil && *c.importingBlock == hash {
		return primivite_consensus_common.BlockStatusQueued, nil
	}
	c.importingBlockMtx.RUnlock()

	number, err := c.backend.Blockchain().Number(hash)
	if err != nil {
		return primivite_consensus_common.BlockStatusUnknown, err
	}

	if number == nil {
		return primivite_consensus_common.BlockStatusUnknown, nil
	}

	if c.backend.HaveStateAt(hash, *number) {
		return primivite_consensus_common.BlockStatusInChainWithState, nil
	} else {
		return primivite_consensus_common.BlockStatusInChainPruned, nil
	}
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Justifications(hash H) (runtime.Justifications, error) {
	return c.backend.Blockchain().Justifications(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockHash(number N) (*H, error) {
	return c.backend.Blockchain().Hash(number)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) IndexedTransaction(hash H) ([]byte, error) {
	return c.backend.Blockchain().IndexedTransaction(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) HasIndexedTransaction(hash H) (bool, error) {
	return c.backend.Blockchain().HasIndexedTransaction(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) BlockIndexedBody(hash H) ([][]byte, error) {
	return c.backend.Blockchain().BlockIndexedBody(hash)
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RequiresFullSync() bool {
	return c.backend.RequiresFullSync()
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) Children(parent H) ([]H, error) {
	return c.backend.Blockchain().Children(parent)
}

// Note: This is an async function so the plan is to ensure we do call it in a goroutine
func (c *Client[H, Hasher, N, E, Executor, Header, RA]) CheckBlock(block common.BlockCheckParams[H, N]) (
	common.ImportResult, error,
) {
	// Check the block against white and black lists if any are defined
	// (i.e. fork blocks and bad blocks respectively)
	switch lookupResult := c.blockRules.Lookup(block.Number, block.Hash).(type) {
	case LookupResultKnownBad:
		logger.Tracef("Rejecting known bad block: #%d %v", block.Number, block.Hash)
		return common.ImportResultKnownBad{}, nil
	case LookupResultExpected[H]:
		logger.Tracef(
			"Rejecting block from known invalid fork. Got %v, expected: %v at height %d",
			block.Hash,
			lookupResult.hash,
			block.Number,
		)
		return common.ImportResultKnownBad{}, nil
	case LookupResultNotSpecial:
		//do nothing
	}

	// Own status must be checked first. If the block and ancestry is pruned
	// this function must return [common.ImportResultAlreadyInChain] rather than [common.ImportResultMissingState]
	blockStatus, err := c.BlockStatus(block.Hash)
	if err != nil {
		return nil, err
	}

	switch blockStatus {
	case primivite_consensus_common.BlockStatusInChainWithState, primivite_consensus_common.BlockStatusQueued:
		return common.ImportResultAlreadyInChain{}, nil
	case primivite_consensus_common.BlockStatusInChainPruned:
		if !block.ImportExisting {
			return common.ImportResultAlreadyInChain{}, nil
		}
	case primivite_consensus_common.BlockStatusUnknown:
		// do nothing
	default:
		panic("unreachable")
	}

	parentStatus, err := c.BlockStatus(block.ParentHash)
	if err != nil {
		return nil, err
	}

	switch parentStatus {
	case primivite_consensus_common.BlockStatusInChainWithState, primivite_consensus_common.BlockStatusQueued:
		// do nothing
	case primivite_consensus_common.BlockStatusUnknown:
		if !block.AllowMissingParent {
			return common.ImportResultUnknownParent{}, nil
		}
	case primivite_consensus_common.BlockStatusInChainPruned:
		if !block.AllowMissingParent {
			return common.ImportResultMissingState{}, nil
		}
	default:
		panic("unreachable")
	}

	return common.ImportResultImported{
		IsNewBest: false,
	}, nil
}

// Note: This is an async function so the plan is to ensure we do call it in a goroutine
func (c *Client[H, Hasher, N, E, Executor, Header, RA]) ImportBlock(
	block *common.BlockImportParams[H, N, E, Header],
) (common.ImportResult, error) {
	prepareStorageResult, err := c.prepareBlockStorageChanges(block)
	if err != nil {
		return nil, err
	}

	var storageChanges common.StorageChanges

	switch r := prepareStorageResult.(type) {
	case prepareStorageChangesResultDiscard:
		return r.ImportResult, nil
	case prepareStorageChangesResultImport:
		storageChanges = r.StorageChanges
	}

	importResult, err := c.LockImportRun(func(
		clientImportOp *api.ClientImportOperation[H, Hasher, N, Header, E],
	) (any, error) {
		result, err := c.applyBlock(clientImportOp, *block, storageChanges)
		return result, err
	})

	if err != nil {
		logger.Warnf("Block import error: %s", err)
		return nil, err
	}

	return importResult.(common.ImportResult), nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) prepareBlockStorageChanges(
	importBlock *common.BlockImportParams[H, N, E, Header],
) (prepareStorageChangesResult, error) {
	parentHash := importBlock.Header.ParentHash()
	stateAction := importBlock.StateAction
	importBlock.StateAction = common.StateActionSkip{}

	var enactState bool
	var storageChanges common.StorageChanges

	status, err := c.BlockStatus(parentHash)
	if err != nil {
		return nil, err
	}

	if status == primivite_consensus_common.BlockStatusInChainPruned {
		switch action := stateAction.(type) {
		case common.StateActionApplyChanges:
			if _, ok := action.StorageChanges.(common.Changes[H, Hasher]); ok {
				return prepareStorageChangesResultDiscard{common.ImportResultMissingState{}}, nil
			}
		case common.StateActionExecute:
			return prepareStorageChangesResultDiscard{common.ImportResultMissingState{}}, nil
		case common.StateActionExecuteIfPossible:
			enactState = false
			storageChanges = nil
		}
	} else if action, ok := stateAction.(common.StateActionApplyChanges); ok {
		enactState = true
		storageChanges = action.StorageChanges
	} else if status == primivite_consensus_common.BlockStatusUnknown {
		return prepareStorageChangesResultDiscard{common.ImportResultUnknownParent{}}, nil
	} else if _, ok := stateAction.(common.StateActionSkip); ok {
		enactState = false
		storageChanges = nil
	} else if _, ok := stateAction.(common.StateActionExecute); ok {
		enactState = true
		storageChanges = nil
	} else if _, ok := stateAction.(common.StateActionExecuteIfPossible); ok {
		enactState = true
		storageChanges = nil
	}

	var storageChangesToApply common.StorageChanges

	if enactState && storageChanges != nil {
		// we have storage changes and should enact the state, so we don't need to do anything here
		storageChangesToApply = storageChanges
	} else if enactState && storageChanges == nil && importBlock.Body != nil {
		// We should enact state, but don't have any storage changes, so we need to execute the block
		runtimeApi := c.RuntimeApi()

		runtimeApi.SetCallContext(core.CallContextOnchain)
		if c.config.EnableImportProofRecording {
			runtimeApi.RecordProof()
			recorder := runtimeApi.ProofRecorder()
			if recorder == nil {
				panic("Proof recording is enabled in the line above; qed.")
			}

			runtimeApi.RegisterExtension(recorder)
		}

		err := runtimeApi.ExecuteBlock(parentHash, generic.NewBlock[Hasher](importBlock.Header, *importBlock.Body))
		if err != nil {
			return nil, err
		}

		state, err := c.backend.StateAt(parentHash)
		if err != nil {
			return nil, err
		}

		genStorageChanges, err := runtimeApi.IntoStorageChanges(state, parentHash)
		if err != nil {
			return nil, err
		}

		if importBlock.Header.StateRoot() != genStorageChanges.TransactionStorageRoot {
			return nil, blockchain.ErrInvalidStateRoot
		}

		storageChangesToApply = common.Changes[H, Hasher](genStorageChanges)
	} else if enactState && storageChanges == nil && importBlock.Body == nil {
		// No block body, no storage changes
		storageChangesToApply = nil
	} else if !enactState {
		// We should not enact the state, so we set the storage changes to None
		storageChangesToApply = nil
	}

	return prepareStorageChangesResultImport{storageChangesToApply}, nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) applyBlock(
	operation *api.ClientImportOperation[H, Hasher, N, Header, E],
	importBlock common.BlockImportParams[H, N, E, Header],
	storageChanges common.StorageChanges,
) (common.ImportResult, error) {
	if len(importBlock.Intermediates) > 0 {
		return nil, blockchain.ErrIncompletePipeline
	}

	if importBlock.ForkChoice == nil {
		return nil, blockchain.ErrIncompletePipeline
	}

	var importHeaders PrePostHeaders[N, H, Header]

	if len(importBlock.PostDigests) == 0 {
		importHeaders = PrePostHeadersSame[N, H, Header]{importBlock.Header}
	} else {
		postHeader := importBlock.Header.Clone()
		for _, item := range importBlock.PostDigests {
			postHeader.DigestMut().Push(item)
		}
		importHeaders = PrePostHeadersDifferent[N, H, Header]{importBlock.Header, importBlock.Header}
	}

	hash := importHeaders.Post().Hash()

	c.importingBlockMtx.Lock()
	c.importingBlock = &hash
	c.importingBlockMtx.Unlock()

	operation.Op.SetCreateGap(importBlock.CreateGap)

	result, err := c.executeAndImportBlock(
		operation,
		importBlock.Origin,
		hash,
		importHeaders,
		importBlock.Justifications,
		importBlock.Body,
		importBlock.IndexedBody,
		storageChanges,
		importBlock.Finalized,
		importBlock.Auxiliary,
		importBlock.ForkChoice,
		importBlock.ImportExisting,
	)
	if err != nil {
		return nil, err
	}

	// TODO: telemetry is implemented here.  See substrate code:
	// https://github.com/paritytech/polkadot-sdk/blob/f5de39196e8c30de4bc47a2d46b1a0fe1e9aaee0/substrate/client/service/src/client/client.rs#L527-L536

	return result, nil
}

//gocyclo:ignore
func (c *Client[H, Hasher, N, E, Executor, Header, RA]) executeAndImportBlock(
	operation *api.ClientImportOperation[H, Hasher, N, Header, E],
	origin consensus.BlockOrigin,
	hash H,
	importHeaders PrePostHeaders[N, H, Header],
	justifications *runtime.Justifications,
	body *[]E,
	indexedBody [][]byte,
	storageChanges common.StorageChanges,
	finalized bool,
	aux api.AuxDataOperations,
	forkchoice common.ForkChoiceStrategy,
	importExisting bool,
) (common.ImportResult, error) {
	parentHash := importHeaders.Post().ParentHash()
	status, err := c.backend.Blockchain().Status(hash)
	if err != nil {
		return nil, err
	}

	parentStatus, err := c.backend.Blockchain().Status(parentHash)
	if err != nil {
		return nil, err
	}

	parentExists := parentStatus != blockchain.BlockStatusUnknown

	if !importExisting && status == blockchain.BlockStatusInChain {
		return common.ImportResultAlreadyInChain{}, nil
	}

	info := c.backend.Blockchain().Info()
	gapBlock := info.BlockGap != nil && info.BlockGap.Start == importHeaders.Post().Number()

	// the block is lower than our last finalized block so it must revert
	// finality, refusing import.
	if status == blockchain.BlockStatusUnknown && importHeaders.Post().Number() < info.FinalizedNumber && !gapBlock {
		return nil, blockchain.ErrNotInFinalizedChain
	}

	// this is a fairly arbitrary choice of where to draw the line on making notifications,
	// but the general goal is to only make notifications when we are already fully synced
	// and get a new chain head.
	makeNotifications := origin == consensus.NetworkBroadcastBlockOrigin ||
		origin == consensus.OwnBlockOrigin ||
		origin == consensus.ConsensusBroadcastBlockOrigin

	var finalStorageChanges *api.StorageChanges

	if storageChanges != nil {
		switch sc := (storageChanges).(type) {
		case common.Changes[H, Hasher]:
			err := c.backend.BeginStateOperation(operation.Op, parentHash)
			if err != nil {
				return nil, err
			}

			if c.config.OffchainIndexingAPI {
				err := operation.Op.UpdateOffchainStorage(sc.OffchainStorageChanges)
				if err != nil {
					return nil, err
				}
			}

			err = operation.Op.UpdateDBStorage(sc.Transaction)
			if err != nil {
				return nil, err
			}

			err = operation.Op.UpdateStorage(sc.MainStorageChanges, sc.ChildStorageChanges)
			if err != nil {
				return nil, err
			}

			err = operation.Op.UpdateTransactionIndex(sc.TransactionIndexChanges)
			if err != nil {
				return nil, err
			}

			finalStorageChanges = &api.StorageChanges{
				StorageCollection:      sc.MainStorageChanges,
				ChildStorageCollection: sc.ChildStorageChanges,
			}
		case common.Import[H]:
			strg := storage.Storage{}
			for _, state := range sc.State {
				if len(state.ParentStorageKeys) == 0 && len(state.StateRoot) == 0 {
					for _, entry := range state.KeyValues {
						strg.Top.Set(string(entry.StorageKey), entry.StorageValue)
					}
				} else {
					for _, parentStorage := range state.ParentStorageKeys {
						var storageKey []byte
						prefixedStorageKey := storage.PrefixedStorageKey(parentStorage)
						if childType := storage.NewChildTypeFromPrefixedKey(prefixedStorageKey); childType != nil {
							storageKey = childType.Key
						} else {
							return nil, blockchain.ErrInvalidChildStorageKey
						}

						entry, has := strg.ChildrenDefault[string(storageKey)]
						if !has {
							entry = storage.StorageChild{
								Data:      btree.Map[string, []byte]{},
								ChildInfo: storage.NewDefaultChildInfo(storageKey),
							}
						}

						for _, kv := range state.KeyValues {
							entry.Data.Set(string(kv.StorageKey), kv.StorageValue)
						}
						strg.ChildrenDefault[string(storageKey)] = entry
					}

					// This is use by fast sync for runtime version to be resolvable from
					// changes.
					stateVersion, err := chainspec.ResolveStateVersionFromWasm[Hasher](strg, c.executor)
					if err != nil {
						return nil, err
					}

					stateRoot, err := operation.Op.ResetStorage(strg, stateVersion)
					if err != nil {
						return nil, err
					}

					if stateRoot != importHeaders.Post().StateRoot() {
						// State root mismatch when importing state. This should not happen in
						// safe fast sync mode, but may happen in unsafe mode.
						logger.Warn("Error importing state: State root mismatch.")
						return nil, blockchain.ErrInvalidStateRoot
					}
				}
			}
		}
	}

	// Ensure parent chain is finalized to maintain invariant that finality is called sequentially.
	if finalized && parentExists && info.FinalizedHash != parentHash {
		err := c.applyFinalityWithBlockHash(operation, parentHash, nil, info, makeNotifications)
		if err != nil {
			return nil, err
		}
	}

	var isNewBest bool

	if !gapBlock && finalized {
		switch fc := forkchoice.(type) {
		case common.LongestChain:
			isNewBest = importHeaders.Post().Number() > info.BestNumber
		case common.Custom:
			isNewBest = bool(fc)
		default:
			panic("unreachable")
		}
	}

	var leafState api.NewBlockState

	if finalized {
		leafState = api.NewBlockStateFinal
	} else if isNewBest {
		leafState = api.NewBlockStateBest
	} else {
		leafState = api.NewBlockStateNormal
	}

	var treeRoute *blockchain.TreeRoute[H, N]

	if isNewBest && info.BestHash != parentHash && parentExists {
		routeFromBest, err := blockchain.NewTreeRoute(c.backend.Blockchain(), info.BestHash, parentHash)
		if err != nil {
			return nil, err
		}
		treeRoute = &routeFromBest
	}

	logger.Tracef(
		"Imported %v, (#%d), best=%v, origin=%v",
		hash,
		importHeaders.Post().Number(),
		isNewBest,
		origin,
	)

	err = operation.Op.SetBlockData(
		importHeaders.Post().Clone().(Header),
		*body,
		indexedBody,
		*justifications,
		leafState,
	)

	if err != nil {
		return nil, err
	}

	err = operation.Op.InsertAux(aux)
	if err != nil {
		return nil, err
	}

	c.everyImportNotificationChansMtx.Lock()
	shouldNotifyEveryBlock := len(c.everyImportNotificationChans) > 0
	c.everyImportNotificationChansMtx.Unlock()

	// Notify when we are already synced to the tip of the chain
	// or if this import triggers a re-org
	shouldNotifyRecentBlock := makeNotifications || treeRoute != nil

	if shouldNotifyEveryBlock || shouldNotifyRecentBlock {
		header := importHeaders.Post()
		if finalized && shouldNotifyRecentBlock {
			var summary api.FinalizeSummary[H, N, Header]
			if operation.NotifyFinalized == nil {
				summary = api.FinalizeSummary[H, N, Header]{
					Header:     header.Clone().(Header),
					Finalized:  []H{parentHash},
					StaleHeads: []H{},
				}
			} else {
				summary = *operation.NotifyFinalized
				operation.NotifyFinalized = nil
				summary.Header = header.Clone().(Header)
				summary.Finalized = append(summary.Finalized, parentHash)
			}

			if parentExists {
				// Add to the stale list all heads that are branching from parent besides our
				// current `head`.
				leaves, err := c.backend.Blockchain().Leaves()
				if err != nil {
					return nil, err
				}

				for _, head := range leaves {
					if head != parentHash {
						routeFromParent, err := blockchain.NewTreeRoute(c.backend.Blockchain(), parentHash, head)
						if err != nil {
							return nil, err
						}

						if len(routeFromParent.Retracted()) == 0 {
							summary.StaleHeads = append(summary.StaleHeads, head)
						}
					}
				}
			}

			*operation.NotifyFinalized = summary
		}

		var importNotificationAction api.ImportNotificationAction
		if shouldNotifyEveryBlock {
			if shouldNotifyRecentBlock {
				importNotificationAction = api.BothBlockImportNotificationAction
			} else {
				importNotificationAction = api.EveryBlockImportNotificationAction
			}
		} else {
			importNotificationAction = api.RecentBlockImportNotificationAction
		}

		operation.NotifyImported = &api.ImportSummary[H, N, Header]{
			Hash:                     hash,
			Origin:                   origin,
			Header:                   header,
			IsNewBest:                isNewBest,
			StorageChanges:           finalStorageChanges,
			TreeRoute:                treeRoute,
			ImportNotificationAction: importNotificationAction,
		}

	}

	return common.ImportResultImported{IsNewBest: isNewBest}, nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) applyFinalityWithBlockHash(
	operation *api.ClientImportOperation[H, Hasher, N, Header, E],
	hash H,
	justification *runtime.Justification,
	info blockchain.Info[H, N],
	notify bool,
) error {
	if hash == info.FinalizedHash {
		logger.Warnf(
			"Possible safety violation: attempted to re-finalize last finalized block %v",
			hash,
		)
		return nil
	}

	// Find tree route from last finalized to given block.
	routeFromFinalized, err := blockchain.NewTreeRoute(c.backend.Blockchain(), info.FinalizedHash, hash)
	if err != nil {
		return err
	}

	if len(routeFromFinalized.Retracted()) > 0 {
		retracted := routeFromFinalized.Retracted()[0]

		logger.Warnf("Safety violation: attempted to revert finalized block %v "+
			"which is not in the same chain as last finalized %v",
			retracted, info.FinalizedHash)

		return blockchain.ErrNotInFinalizedChain
	}

	// We may need to coercively update the best block if there is more than one
	// leaf or if the finalized block number is greater than last best number recorded
	// by the backend. This last condition may apply in case of consensus implementations
	// not always checking this condition.
	blockNumber, err := c.backend.Blockchain().Number(hash)
	if err != nil {
		return fmt.Errorf("Failed to get header for hash %v", hash)
	}

	leaves, err := c.backend.Blockchain().Leaves()
	if err != nil {
		return err
	}

	if len(leaves) > 1 || info.BestNumber < *blockNumber {
		routeFromBest, err := blockchain.NewTreeRoute(c.backend.Blockchain(), info.BestHash, hash)
		if err != nil {
			return err
		}

		// If the block is not a direct ancestor of the current best chain,
		// then some other block is the common ancestor.
		if routeFromBest.CommonBlock().Hash != hash {
			// NOTE: we're setting the finalized block as best block, this might
			// be slightly inaccurate since we might have a "better" block
			// further along this chain, but since best chain selection logic is
			// plugable we cannot make a better choice here. usages that need
			// an accurate "best" block need to go through `SelectChain`
			// instead.
			if err := operation.Op.MarkHead(hash); err != nil {
				return err
			}
		}
	}

	enacted := routeFromFinalized.Enacted()
	if len(enacted) == 0 {
		panic("no enacted blocks")
	}

	for _, finalizeNew := range enacted[:len(enacted)-1] {
		if err := operation.Op.MarkFinalized(finalizeNew.Hash, nil); err != nil {
			return err
		}
	}

	if enacted[len(enacted)-1].Hash != hash {
		panic("finalized block is not the last enacted block")
	}
	if err := operation.Op.MarkFinalized(hash, justification); err != nil {
		return err
	}

	if notify {
		var finalized []H
		for _, elem := range routeFromFinalized.Enacted() {
			finalized = append(finalized, elem.Hash)
		}

		lastFinalized := routeFromFinalized.Last()

		if lastFinalized == nil {
			panic("the block to finalize is always the latest block in the route to the finalized block; qed")
		}

		blockNumber := lastFinalized.Number

		// The stale heads are the leaves that will be displaced after the
		// block is finalized.
		var staleHeads []H
		displacedLeaves, err := c.backend.Blockchain().DisplacedLeavesAfterFinalizing(hash, blockNumber)
		if err != nil {
			return err
		}

		staleHeads = displacedLeaves.Hashes()

		header, err := c.backend.Blockchain().Header(hash)
		if err != nil {
			return err
		}

		operation.NotifyFinalized = &api.FinalizeSummary[H, N, Header]{
			Header:     *header,
			Finalized:  finalized,
			StaleHeads: staleHeads,
		}

	}

	return nil
}

func (c *Client[H, Hasher, N, E, Executor, Header, RA]) RuntimeApi() primitives_api.ApiExt[
	N, E, H, Hasher,
	statemachine.Backend[H, Hasher],
	any,
] {
	return c.runtimeConstructor.ConstructRuntimeApi()
}
