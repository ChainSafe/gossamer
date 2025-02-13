// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/blockchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "client"))

// Client type that implements a number of client interfaces
type Client[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	backend                         api.Backend[H, N, Hasher, Header, E]
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
	importingBlockMtx sync.RWMutex
	importingBlock    *H
	unpinWorkerChan   chan<- api.UnpinWorkerMessage[H]
}

// New is constructor for [Client]
func New[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
](
	backend api.Backend[H, N, Hasher, Header, E],
) *Client[H, Hasher, N, E, Header] {

	unpinWorkerChan := make(chan api.UnpinWorkerMessage[H])
	npw := newNotificationPinningWorker(unpinWorkerChan, backend)
	go npw.run()

	return &Client[H, Hasher, N, E, Header]{
		backend:                      backend,
		storageNotifications:         api.NewStorageNotifications[H](),
		importNotificationChans:      make(map[chan api.BlockImportNotification[H, N, Header]]any),
		everyImportNotificationChans: make(map[chan api.BlockImportNotification[H, N, Header]]any),
		finalityNotificationChans:    make(map[chan api.FinalityNotification[H, N, Header]]any),
		unpinWorkerChan:              unpinWorkerChan,
	}
}

func (c *Client[H, Hasher, N, E, Header]) announcePin(message api.AnnouncePin[H]) error {
	select {
	case c.unpinWorkerChan <- message:
		return nil
	default:
		return fmt.Errorf("unable to send AnnouncePin message to Client.unpinWorkerChan")
	}
}

func (c *Client[H, Hasher, N, E, Header]) unpin(message api.Unpin[H]) error {
	select {
	case c.unpinWorkerChan <- message:
		return nil
	default:
		return fmt.Errorf("unable to send Unpin message to Client.unpinWorkerChan")
	}
}

func (c *Client[H, Hasher, N, E, Header]) lockImportRun(
	f func(*api.ClientImportOperation[H, Hasher, N, Header, E]) error,
) error {
	c.backend.GetImportLock().Lock()
	defer c.backend.GetImportLock().Unlock()

	blockImportOp, err := c.backend.BeginOperation()
	if err != nil {
		return err
	}

	clientImportOp := api.ClientImportOperation[H, Hasher, N, Header, E]{
		Op: blockImportOp,
	}

	err = f(&clientImportOp)
	if err != nil {
		return err
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
				return err
			}
		}
	}
	if importNotification != nil {
		c.importActionsMtx.Lock()
		defer c.importActionsMtx.Unlock()
		for _, action := range c.importActions {
			err := clientImportOp.Op.InsertAux(action(*importNotification))
			if err != nil {
				return err
			}
		}
	}

	err = c.backend.CommitOperation(clientImportOp.Op)
	if err != nil {
		return err
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
		return err
	}
	err = c.notifyImported(importNotification, importNotificationAction, storageChanges)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client[H, Hasher, N, E, Header]) LockImportRun(
	f func(*api.ClientImportOperation[H, Hasher, N, Header, E]) error,
) error {
	err := c.lockImportRun(f)
	c.importingBlockMtx.Lock()
	c.importingBlock = nil
	c.importingBlockMtx.Unlock()
	return err
}

const notifyFinalizedTimeout = 5 * time.Second
const notifyBlockImportTimeout = notifyFinalizedTimeout

func (c *Client[H, Hasher, N, E, Header]) notifyFinalized(notification *api.FinalityNotification[H, N, Header]) error {
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

func (c *Client[H, Hasher, N, E, Header]) notifyImported(
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

func (c *Client[H, Hasher, N, E, Header]) RegisterImportAction(op api.OnImportAction[H, N, Header]) {
	c.importActionsMtx.Lock()
	defer c.importActionsMtx.Unlock()
	c.importActions = append(c.importActions, op)
}

func (c *Client[H, Hasher, N, E, Header]) RegisterFinalityAction(op api.OnFinalityAction[H, N, Header]) {
	c.finalityActionsMtx.Lock()
	defer c.finalityActionsMtx.Unlock()
	c.finalityActions = append(c.finalityActions, op)
}

func (c *Client[H, Hasher, N, E, Header]) RegisterImportNotificationStream() api.ImportNotifications[H, N, Header] {
	ch := make(chan api.BlockImportNotification[H, N, Header])
	c.importNotificationChansMtx.Lock()
	defer c.importNotificationChansMtx.Unlock()
	c.importNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Header]) UnregisterImportNotificationStream(
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

func (c *Client[H, _, N, E, Header]) RegisterEveryImportNotificationStream() api.ImportNotifications[H, N, Header] {
	ch := make(chan api.BlockImportNotification[H, N, Header])
	c.everyImportNotificationChansMtx.Lock()
	defer c.everyImportNotificationChansMtx.Unlock()
	c.everyImportNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Header]) UnregisterEveryImportNotificationStream(
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

func (c *Client[H, _, N, E, Header]) RegisterFinalityNotificationStream() api.FinalityNotifications[H, N, Header] {
	ch := make(chan api.FinalityNotification[H, N, Header])
	c.finalityNotificationChansMtx.Lock()
	defer c.finalityNotificationChansMtx.Unlock()
	c.finalityNotificationChans[ch] = nil
	return ch
}

func (c *Client[H, Hasher, N, E, Header]) UnregisterFinalityNotificationStream(
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

func (c *Client[H, Hasher, N, E, Header]) StorageChangesNotificationStream(
	filterKeys []storage.StorageKey,
	childFilterKeys []api.ChildFilterKeys,
) api.StorageEventStream[H] {
	return c.storageNotifications.Listen(filterKeys, childFilterKeys)
}

// HeaderBackend implementation for Client

func (c *Client[H, Hasher, N, E, Header]) Header(hash H) (*Header, error) {
	return c.backend.Blockchain().Header(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Body(hash H) ([]E, error) {
	return c.backend.Blockchain().Body(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Info() blockchain.Info[H, N] {
	return c.backend.Blockchain().Info()
}

func (c *Client[H, Hasher, N, E, Header]) Status(hash H) (blockchain.BlockStatus, error) {
	return c.backend.Blockchain().Status(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Number(hash H) (*N, error) {
	return c.backend.Blockchain().Number(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Hash(number N) (*H, error) {
	return c.backend.Blockchain().Hash(number)
}

func (c *Client[H, Hasher, N, E, Header]) BlockHashFromID(id generic.BlockID) (*H, error) {
	return c.backend.Blockchain().BlockHashFromID(id)
}

func (c *Client[H, Hasher, N, E, Header]) BlockNumberFromID(id generic.BlockID) (*N, error) {
	return c.backend.Blockchain().BlockNumberFromID(id)
}

// BlockBackend implementation for Client

func (c *Client[H, Hasher, N, E, Header]) BlockBody(hash H) ([]E, error) {
	return c.backend.Blockchain().Body(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Block(hash H) (*generic.SignedBlock[N, H, Hasher, E], error) {
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
			generic.NewBlock[N, H, Hasher](*header, body), justifications,
		), nil
	}

	return nil, nil
}

func (c *Client[H, Hasher, N, E, Header]) BlockStatus(hash H) (blockchain.BlockStatus, error) {
	return c.backend.Blockchain().Status(hash)
}

func (c *Client[H, Hasher, N, E, Header]) Justifications(hash H) (runtime.Justifications, error) {
	return c.backend.Blockchain().Justifications(hash)
}

func (c *Client[H, Hasher, N, E, Header]) BlockHash(number N) (*H, error) {
	return c.backend.Blockchain().Hash(number)
}

func (c *Client[H, Hasher, N, E, Header]) IndexedTransaction(hash H) ([]byte, error) {
	return c.backend.Blockchain().IndexedTransaction(hash)
}

func (c *Client[H, Hasher, N, E, Header]) HasIndexedTransaction(hash H) (bool, error) {
	return c.backend.Blockchain().HasIndexedTransaction(hash)
}

func (c *Client[H, Hasher, N, E, Header]) BlockIndexedBody(hash H) ([][]byte, error) {
	return c.backend.Blockchain().BlockIndexedBody(hash)
}

func (c *Client[H, Hasher, N, E, Header]) RequiresFullSync() bool {
	return c.backend.RequiresFullSync()
}
