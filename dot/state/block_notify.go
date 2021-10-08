// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"errors"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/google/uuid"
)

// DEFAULT_BUFFER_SIZE buffer size for channels
const DEFAULT_BUFFER_SIZE = 128

// GetImportedBlockNotifierChannel function to retrieve a imported block notifier channel
func (bs *BlockState) GetImportedBlockNotifierChannel() chan *types.Block {
	bs.importedLock.Lock()
	defer bs.importedLock.Unlock()

	ch := make(chan *types.Block, DEFAULT_BUFFER_SIZE)
	bs.imported[ch] = struct{}{}
	return ch
}

//nolint
// GetFinalisedNotifierChannel function to retrieve a finalized block notifier channel
func (bs *BlockState) GetFinalisedNotifierChannel() chan *types.FinalisationInfo {
	bs.finalisedLock.Lock()
	defer bs.finalisedLock.Unlock()

	ch := make(chan *types.FinalisationInfo, DEFAULT_BUFFER_SIZE)
	bs.finalised[ch] = struct{}{}

	return ch
}

// FreeImportedBlockNotifierChannel to free imported block notifier channel
func (bs *BlockState) FreeImportedBlockNotifierChannel(ch chan *types.Block) {
	bs.importedLock.Lock()
	defer bs.importedLock.Unlock()
	delete(bs.imported, ch)
}

//nolint
// FreeFinalisedNotifierChannel to free finalized notifier channel
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

	logger.Trace("notifying imported block chans...", "chans", bs.imported)
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
		logger.Error("failed to get finalised header", "hash", hash, "error", err)
		return
	}

	logger.Debug("notifying finalised block chans...", "chans", bs.finalised)
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

	logger.Debug("notifying runtime updated chans...", "chans", bs.runtimeUpdateSubscriptions)
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
