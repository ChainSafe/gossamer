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
	"fmt"
	"reflect"

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

//go:generate mockery --name Observer --structname MockObserver --case underscore --inpackage

// Observer interface defines functions needed for observers, Observer Design Pattern
type Observer interface {
	Update(result *SubscriptionResult)
	GetID() uint
	GetFilter() map[string][]byte
}

// RegisterStorageObserver to add abserver to notification list
func (s *StorageState) RegisterStorageObserver(o Observer) {
	s.observerList = append(s.observerList, o)

	// notifyObserver here to send storage value of current state
	sr, err := s.blockState.BestBlockStateRoot()
	if err != nil {
		logger.Debug("error registering storage change channel", "error", err)
		return
	}
	go func() {
		if err := s.notifyObserver(sr, o); err != nil {
			logger.Warn("failed to notify storage subscriptions", "error", err)
		}
	}()

}

// UnregisterStorageObserver removes observer from notification list
func (s *StorageState) UnregisterStorageObserver(o Observer) {
	s.observerList = s.removeFromSlice(s.observerList, o)
}

func (s *StorageState) notifyAll(root common.Hash) {
	s.changedLock.RLock()
	defer s.changedLock.RUnlock()
	for _, observer := range s.observerList {
		err := s.notifyObserver(root, observer)
		if err != nil {
			logger.Warn("failed to notify storage subscriptions", "error", err)
		}
	}
}

func (s *StorageState) notifyObserver(root common.Hash, o Observer) error {
	t, err := s.TrieState(&root)
	if err != nil {
		return err
	}

	if t == nil {
		return errTrieDoesNotExist(root)
	}

	subRes := &SubscriptionResult{
		Hash: root,
	}
	if len(o.GetFilter()) == 0 {
		// no filter, so send all changes
		ent := t.TrieEntries()
		for k, v := range ent {
			if k != ":code" {
				// currently we're ignoring :code since this is a lot of data
				kv := &KeyValue{
					Key:   common.MustHexToBytes(fmt.Sprintf("0x%x", k)),
					Value: v,
				}
				subRes.Changes = append(subRes.Changes, *kv)
			}
		}
	} else {
		// filter result to include only interested keys
		for k, cachedValue := range o.GetFilter() {
			value := t.Get(common.MustHexToBytes(k))
			if !reflect.DeepEqual(cachedValue, value) {
				kv := &KeyValue{
					Key:   common.MustHexToBytes(k),
					Value: value,
				}
				subRes.Changes = append(subRes.Changes, *kv)
				o.GetFilter()[k] = value
			}
		}
	}

	if len(subRes.Changes) > 0 {
		logger.Trace("update observer", "changes", subRes.Changes)
		go func() {
			o.Update(subRes)
		}()
	}

	return nil
}

func (s *StorageState) removeFromSlice(observerList []Observer, observerToRemove Observer) []Observer {
	s.changedLock.Lock()
	defer s.changedLock.Unlock()
	observerListLength := len(observerList)
	for i, observer := range observerList {
		if observerToRemove.GetID() == observer.GetID() {
			observerList[i] = observerList[observerListLength-1]
			return observerList[:observerListLength-1]
		}
	}
	return observerList
}
