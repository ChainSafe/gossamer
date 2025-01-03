package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/client/api"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	statemachine "github.com/ChainSafe/gossamer/internal/primitives/state-machine"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

var logger = log.NewFromGlobal(log.AddContext("client", ""))

// / Callback invoked before committing the operations created during block import.
// / This gives the opportunity to perform auxiliary pre-commit actions and optionally
// / enqueue further storage write operations to be atomically performed on commit.
// pub type OnImportAction<Block> =
//
//	Box<dyn (Fn(&BlockImportNotification<Block>) -> AuxDataOperations) + Send>;
type OnImportAction[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] func(api.BlockImportNotification[H, N, Header]) api.AuxDataOperations

// / Callback invoked before committing the operations created during block finalization.
// / This gives the opportunity to perform auxiliary pre-commit actions and optionally
// / enqueue further storage write operations to be atomically performed on commit.
// pub type OnFinalityAction<Block> =
//
//	Box<dyn (Fn(&FinalityNotification<Block>) -> AuxDataOperations) + Send>;
type OnFinalityAction[
	H runtime.Hash,
	N runtime.Number,
	Header runtime.Header[N, H],
] func(api.FinalityNotification[H, N, Header]) api.AuxDataOperations

type Client[
	H runtime.Hash,
	Hasher runtime.Hasher[H],
	N runtime.Number,
	E runtime.Extrinsic,
	Header runtime.Header[N, H],
] struct {
	backend api.Backend[H, N, Hasher, Header, E]

	storageNotifications api.StorageNotifications[H]

	importNotificationChansMtx      sync.Mutex
	importNotificationChans         map[chan<- api.BlockImportNotification[H, N, Header]]any
	everyImportNotificationChansMtx sync.Mutex
	everyImportNotificationChans    map[chan<- api.BlockImportNotification[H, N, Header]]any

	finalityNotificationChansMtx sync.Mutex
	finalityNotificationChans    map[chan<- api.FinalityNotification[H, N, Header]]any

	// Collects auxiliary operations to be performed atomically together with
	// block import operations.
	// import_actions: Mutex<Vec<OnImportAction<Block>>>,
	importActionsMtx sync.Mutex
	importActions    []OnImportAction[H, N, Header]
	// Collects auxiliary operations to be performed atomically together with
	// block finalization operations.
	// finality_actions: Mutex<Vec<OnFinalityAction<Block>>>,
	finalityActionsMtx sync.Mutex
	finalityActions    []OnFinalityAction[H, N, Header]
	// Holds the block hash currently being imported. TODO: replace this with block queue.
	// importing_block: RwLock<Option<Block::Hash>>,
	importingBlockMtx sync.RWMutex
	importingBlock    *H

	unpinWorkerChan chan<- api.UnpinWorkerMessage[H]
}

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
		importNotificationChans:      make(map[chan<- api.BlockImportNotification[H, N, Header]]any),
		everyImportNotificationChans: make(map[chan<- api.BlockImportNotification[H, N, Header]]any),
		finalityNotificationChans:    make(map[chan<- api.FinalityNotification[H, N, Header]]any),
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
		return fmt.Errorf("unable to send message to Client.unpinWorkerChan")
	}
}

func (c *Client[H, Hasher, N, E, Header]) LockImportRun(f func(*api.ClientImportOperation[H, Hasher, N, Header, E]) error) error {
	var inner = func() error {
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
			importNotification *api.BlockImportNotification[H, N, Header]
			storageChanges     *struct {
				statemachine.StorageCollection
				statemachine.ChildStorageCollection
			}
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
					finalityNotification.Hash, err)
			} else {
				err := c.announcePin(api.AnnouncePin[H]{Hash: finalityNotification.Hash})
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

	err := inner()
	c.importingBlockMtx.Lock()
	c.importingBlock = nil
	c.importingBlockMtx.Unlock()
	return err
}

const notifyFinalizedTimeout = time.Duration(5 * time.Second)
const notifyBlockImportTimout = notifyFinalizedTimeout

func (c *Client[H, Hasher, N, E, Header]) notifyFinalized(notification *api.FinalityNotification[H, N, Header]) error {
	c.finalityNotificationChansMtx.Lock()
	defer c.finalityNotificationChansMtx.Unlock()

	if notification == nil {
		return nil
	}

	// telemetry!(
	// 	self.telemetry;
	// 	SUBSTRATE_INFO;
	// 	"notify.finalized";
	// 	"height" => format!("{}", notification.header.number()),
	// 	"best" => ?notification.hash,
	// );

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

func notifyChans[M any](msg M, chans map[chan<- M]any, timeout time.Duration) {
	wg := sync.WaitGroup{}
	for ch := range chans {
		wg.Add(1)
		go func(ch chan<- M) {
			defer wg.Done()
			ch <- msg
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
	storageChanges *struct {
		statemachine.StorageCollection
		statemachine.ChildStorageCollection
	},
) error {
	if notification != nil {
		return nil
	}

	var triggerStorageChangesNotification = func() {
		if storageChanges != nil {
			// TODO [ToDr] How to handle re-orgs? Should we re-emit all storage changes? (from substrate)
			changeset := make([]api.Change, len(storageChanges.StorageCollection))
			for i, kv := range storageChanges.StorageCollection {
				changeset[i] = api.Change{
					Key:   storage.StorageKey(kv.StorageKey),
					Value: storage.StorageData(kv.StorageValue),
				}
			}
			childChangeset := make([]api.ChildChange, len(storageChanges.ChildStorageCollection))
			for i, kc := range storageChanges.ChildStorageCollection {
				changeset := make([]api.Change, len(kc.StorageCollection))
				for i, kv := range kc.StorageCollection {
					changeset[i] = api.Change{
						Key:   storage.StorageKey(kv.StorageKey),
						Value: storage.StorageData(kv.StorageValue),
					}
				}
				childChangeset[i] = api.ChildChange{
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
		notifyChans(*notification, c.importNotificationChans, notifyBlockImportTimout)

		c.everyImportNotificationChansMtx.Lock()
		defer c.everyImportNotificationChansMtx.Unlock()
		notifyChans(*notification, c.everyImportNotificationChans, notifyBlockImportTimout)
	case api.RecentBlockImportNotificationAction:
		triggerStorageChangesNotification()
		c.importNotificationChansMtx.Lock()
		defer c.importNotificationChansMtx.Unlock()
		notifyChans(*notification, c.importNotificationChans, notifyBlockImportTimout)
	case api.EveryBlockImportNotificationAction:
		c.everyImportNotificationChansMtx.Lock()
		defer c.everyImportNotificationChansMtx.Unlock()
		notifyChans(*notification, c.everyImportNotificationChans, notifyBlockImportTimout)
	case api.NoneBlockImportNotificationAction:
		// This branch is unreachable in fact because the block import notification must be
		// Some(_) instead of None (it's already handled at the beginning of this function)
		// at this point.
	default:
		panic("unreachable")
	}

	return nil
}
