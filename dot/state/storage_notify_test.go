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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

type MockStorageObserver struct {
	id         uint
	filter     map[string][]byte
	lastUpdate *SubscriptionResult
	m          sync.RWMutex
}

func (m *MockStorageObserver) Update(change *SubscriptionResult) {
	m.m.Lock()
	m.lastUpdate = change
	m.m.Unlock()

}
func (m *MockStorageObserver) GetID() uint {
	return m.id
}
func (m *MockStorageObserver) GetFilter() map[string][]byte {
	return m.filter
}

func TestStorageState_RegisterStorageObserver(t *testing.T) {
	ss := newTestStorageState(t)

	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	observer := &MockStorageObserver{}
	ss.RegisterStorageObserver(observer)

	defer ss.UnregisterStorageObserver(observer)

	ts.Set([]byte("mackcom"), []byte("wuz here"))
	err = ss.StoreTrie(ts)
	require.NoError(t, err)

	expectedResult := &SubscriptionResult{
		Hash: ts.MustRoot(),
		Changes: []KeyValue{{
			Key:   []byte("mackcom"),
			Value: []byte("wuz here"),
		}},
	}
	time.Sleep(time.Millisecond)
	observer.m.RLock()
	defer observer.m.RUnlock()
	require.Equal(t, expectedResult, observer.lastUpdate)
}

func TestStorageState_RegisterStorageObserver_Multi(t *testing.T) {
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	num := 5

	var observers []*MockStorageObserver

	for i := 0; i < num; i++ {
		observer := &MockStorageObserver{
			id: uint(i),
		}
		observers = append(observers, observer)
		ss.RegisterStorageObserver(observer)
		require.NoError(t, err)
	}

	key1 := []byte("key1")
	value1 := []byte("value1")

	ts.Set(key1, value1)

	err = ss.StoreTrie(ts)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range observers {
		observer.m.RLock()
		require.NotNil(t, observer.lastUpdate.Hash)
		require.Equal(t, key1, observer.lastUpdate.Changes[0].Key)
		require.Equal(t, value1, observer.lastUpdate.Changes[0].Value)
		observer.m.RUnlock()
	}

	for _, observer := range observers {
		ss.UnregisterStorageObserver(observer)
	}
}

func TestStorageState_RegisterStorageObserver_Multi_Filter(t *testing.T) {
	t.Skip() // this seems to fail often on CI
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	key1 := []byte("key1")
	value1 := []byte("value1")

	num := 5
	var observers []*MockStorageObserver

	for i := 0; i < num; i++ {
		observer := &MockStorageObserver{
			id: uint(i),
			filter: map[string][]byte{
				common.BytesToHex(key1): {},
			},
		}
		observers = append(observers, observer)
		ss.RegisterStorageObserver(observer)
	}

	ts.Set(key1, value1)
	err = ss.StoreTrie(ts)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range observers {
		observer.m.RLock()
		require.NotNil(t, observer.lastUpdate.Hash)
		require.Equal(t, key1, observer.lastUpdate.Changes[0].Key)
		require.Equal(t, value1, observer.lastUpdate.Changes[0].Value)
		observer.m.RUnlock()
	}

	for _, observer := range observers {
		ss.UnregisterStorageObserver(observer)
	}
}

func Test_Example(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subscription example")
	}

	// this is a working example of how to use db.Subscribe taken from
	// https://github.com/dgraph-io/badger/blob/f50343ff404d8198df6dc83755ec2eab863d5ff2/db_test.go#L1939-L1948
	prefix := []byte{'a'}

	// This key should be printed, since it matches the prefix.
	aKey := []byte("a-key")
	aValue := []byte("a-value")

	// This key should not be printed.
	bKey := []byte("b-key")
	bValue := []byte("b-value")

	// Open the DB.
	dir, err := ioutil.TempDir("", "badger-test")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = os.RemoveAll(dir); err != nil {
			log.Fatal(err)
		}
	}()

	db := NewInMemoryDB(t)

	// Create the context here so we can cancel it after sending the writes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the WaitGroup to make sure we wait for the subscription to stop before continuing.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(kvs *chaindb.KVList) error {
			for _, kv := range kvs.Kv {
				fmt.Printf("%s is now set to %s\n", kv.Key, kv.Value)
			}
			return nil
		}
		if err = db.Subscribe(ctx, cb, prefix); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		log.Printf("subscription closed")
	}()

	// Write both keys, but only one should be printed in the Output.
	err = db.Put(aKey, aValue)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Put(bKey, bValue)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	log.Printf("stopping subscription")
	cancel()
	log.Printf("waiting for subscription to close")
	wg.Wait()
	// Output:
	// a-key is now set to a-value
}
