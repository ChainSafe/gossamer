// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"errors"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/google/uuid"
)

const defaultBufferSize = 128

// GetImportedBlockNotifierChannel function to retrieve a imported block notifier channel
func (bs *BlockState) GetImportedBlockNotifierChannel() chan *types.Block {
	bs.importedLock.Lock()
	defer bs.importedLock.Unlock()

	ch := make(chan *types.Block, defaultBufferSize)
	bs.imported[ch] = struct{}{}
	return ch
}

// GetFinalisedNotifierChannel function to retrieve a finalised block notifier channel
func (bs *BlockState) GetFinalisedNotifierChannel() chan *types.FinalisationInfo {
	bs.finalisedLock.Lock()
	defer bs.finalisedLock.Unlock()

	ch := make(chan *types.FinalisationInfo, defaultBufferSize)
	bs.finalised[ch] = struct{}{}

	return ch
}

// FreeImportedBlockNotifierChannel to free imported block notifier channel
func (bs *BlockState) FreeImportedBlockNotifierChannel(ch chan *types.Block) {
	bs.importedLock.Lock()
	defer bs.importedLock.Unlock()
	delete(bs.imported, ch)
}

// FreeFinalisedNotifierChannel to free finalised notifier channel
func (bs *BlockState) FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo) {
	bs.finalisedLock.Lock()
	defer bs.finalisedLock.Unlock()

	delete(bs.finalised, ch)
}

func (bs *BlockState) notifyImported(block *types.Block) {
	bs.importedLock.RLock()
	defer bs.importedLock.RUnlock()

	if len(bs.imported) == 0 {
		return
	}

	logger.Trace("notifying imported block channels...")
	for ch := range bs.imported {
		go func(ch chan *types.Block) {
			select {
			case ch <- block:
			default:
			}
		}(ch)
	}
}

func (bs *BlockState) notifyFinalized(hash common.Hash, round, setID uint64) {
	bs.finalisedLock.RLock()
	defer bs.finalisedLock.RUnlock()

	if len(bs.finalised) == 0 {
		return
	}

	header, err := bs.GetHeader(hash)
	if err != nil {
		logger.Errorf("failed to get finalised header for hash %s: %s", hash, err)
		return
	}

	logger.Debug("notifying finalised block channels...")
	info := &types.FinalisationInfo{
		Header: *header,
		Round:  round,
		SetID:  setID,
	}

	for ch := range bs.finalised {
		go func(ch chan *types.FinalisationInfo) {
			select {
			case ch <- info:
			default:
			}
		}(ch)
	}
}

func (bs *BlockState) notifyRuntimeUpdated(version runtime.Version) {
	bs.runtimeUpdateSubscriptionsLock.RLock()
	defer bs.runtimeUpdateSubscriptionsLock.RUnlock()

	if len(bs.runtimeUpdateSubscriptions) == 0 {
		return
	}

	logger.Debug("notifying runtime updated channels...")
	var wg sync.WaitGroup
	wg.Add(len(bs.runtimeUpdateSubscriptions))
	for _, ch := range bs.runtimeUpdateSubscriptions {
		go func(ch chan<- runtime.Version) {
			defer wg.Done()
			ch <- version
		}(ch)
	}
	wg.Wait()
}

// RegisterRuntimeUpdatedChannel function to register chan that is notified when runtime version changes
func (bs *BlockState) RegisterRuntimeUpdatedChannel(ch chan<- runtime.Version) (uint32, error) {
	bs.runtimeUpdateSubscriptionsLock.Lock()
	defer bs.runtimeUpdateSubscriptionsLock.Unlock()

	if len(bs.runtimeUpdateSubscriptions) == 256 {
		return 0, errors.New("channel limit reached")
	}

	id := bs.generateID()

	bs.runtimeUpdateSubscriptions[id] = ch
	return id, nil
}

// UnregisterRuntimeUpdatedChannel function to unregister runtime updated channel
func (bs *BlockState) UnregisterRuntimeUpdatedChannel(id uint32) bool {
	bs.runtimeUpdateSubscriptionsLock.Lock()
	defer bs.runtimeUpdateSubscriptionsLock.Unlock()
	ch, ok := bs.runtimeUpdateSubscriptions[id]
	if ok {
		close(ch)
		delete(bs.runtimeUpdateSubscriptions, id)
		return true
	}
	return false
}

func (bs *BlockState) generateID() uint32 {
	var uid uuid.UUID
	for {
		uid = uuid.New()
		if bs.runtimeUpdateSubscriptions[uid.ID()] == nil {
			break
		}
	}
	return uid.ID()
}
