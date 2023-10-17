// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

// KeyValue struct to hold key value pairs
type KeyValue struct {
	Key   []byte
	Value []byte
}

func (kv KeyValue) String() string {
	return fmt.Sprintf("{Key: 0x%x, BTree: 0x%x}", kv.Key, kv.Value)
}

// SubscriptionResult holds results of storage changes
type SubscriptionResult struct {
	Hash    common.Hash
	Changes []KeyValue
}

// String serialises the subscription result changes
// to human readable strings.
func (s SubscriptionResult) String() string {
	changes := make([]string, len(s.Changes))
	for i := range s.Changes {
		changes[i] = s.Changes[i].String()
	}
	return "[" + strings.Join(changes, ", ") + "]"
}

// Observer interface defines functions needed for observers, Observer Design Pattern
type Observer interface {
	Update(result *SubscriptionResult)
	GetID() uint
	GetFilter() map[string][]byte
}

// RegisterStorageObserver to add abserver to notification list
func (s *StorageState) RegisterStorageObserver(o Observer) {
	s.observerListMutex.Lock()
	defer s.observerListMutex.Unlock()
	s.observerList = append(s.observerList, o)

	// notifyObserver here to send storage value of current state
	sr, err := s.blockState.BestBlockStateRoot()
	if err != nil {
		logger.Debugf("error registering storage change channel: %s", err)
		return
	}
	go func() {
		if err := s.notifyObserver(sr, o); err != nil {
			logger.Warnf("failed to notify storage subscriptions: %s", err)
		}
	}()

}

// UnregisterStorageObserver removes observer from notification list
func (s *StorageState) UnregisterStorageObserver(o Observer) {
	s.observerListMutex.Lock()
	defer s.observerListMutex.Unlock()
	s.observerList = s.removeFromSlice(s.observerList, o)
}

func (s *StorageState) notifyAll(root common.Hash) {
	s.observerListMutex.RLock()
	defer s.observerListMutex.RUnlock()
	for _, observer := range s.observerList {
		err := s.notifyObserver(root, observer)
		if err != nil {
			logger.Warnf("failed to notify storage subscriptions: %s", err)
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
		logger.Tracef("update observer, changes are %v", subRes.Changes)
		go func() {
			o.Update(subRes)
		}()
	}

	return nil
}

func (s *StorageState) removeFromSlice(observerList []Observer, observerToRemove Observer) []Observer {
	observerListLength := len(observerList)
	for i, observer := range observerList {
		if observerToRemove.GetID() == observer.GetID() {
			observerList[i] = observerList[observerListLength-1]
			return observerList[:observerListLength-1]
		}
	}
	return observerList
}
