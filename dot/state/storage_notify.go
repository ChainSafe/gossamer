// Copyright 2020 ChainSafe Systems (ON) Corp.
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

	"github.com/ChainSafe/gossamer/lib/common"
)

// KeyValue struct to hold key value pairs
type KeyValue struct {
	Key   []byte
	Value []byte
}

//SubscriptionResult holds results of storage changes
type SubscriptionResult struct {
	Hash    common.Hash
	Changes []KeyValue
}

//StorageSubscription holds data for Subscription to Storage
type StorageSubscription struct {
	Filter   map[string][]byte
	Listener chan<- *SubscriptionResult
}

// RegisterStorageChangeChannel function to register storage change channels
func (s *StorageState) RegisterStorageChangeChannel(sub StorageSubscription) (byte, error) {
	s.changedLock.RLock()

	if len(s.subscriptions) == 256 {
		return 0, errors.New("storage subscriptions limit reached")
	}

	var id byte
	for {
		id = generateID()
		if s.subscriptions[id] == nil {
			break
		}
	}

	s.changedLock.RUnlock()

	s.changedLock.Lock()
	s.subscriptions[id] = &sub
	s.changedLock.Unlock()
	// notifyStorageSubscriptions here to send storage value of current state
	sr, err := s.blockState.BestBlockStateRoot()
	if err != nil {
		logger.Debug("error registering storage change channel", "error", err)
	}
	go func() {
		if err := s.notifyStorageSubscriptions(sr); err != nil {
			logger.Warn("failed to notify storage subscriptions", "error", err)
		}
	}()
	return id, nil
}

// UnregisterStorageChangeChannel removes the storage change notification channel with the given ID.
// A channel must be unregistered before closing it.
func (s *StorageState) UnregisterStorageChangeChannel(id byte) {
	s.changedLock.Lock()
	defer s.changedLock.Unlock()

	delete(s.subscriptions, id)
}
