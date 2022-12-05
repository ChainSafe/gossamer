// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStorageState_RegisterStorageObserver(t *testing.T) {
	ss := newTestStorageState(t)

	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	mockfilter := map[string][]byte{}
	mockobs := NewMockObserver(t)

	mockobs.On("GetID").Return(uint(10)).Times(2)

	var fireAndForgetMockCallsWG sync.WaitGroup

	fireAndForgetMockCallsWG.Add(2)
	getFilterCall := mockobs.On("GetFilter").Return(mockfilter).Times(2)
	getFilterCall.RunFn = func(args mock.Arguments) {
		fireAndForgetMockCallsWG.Done()
	}

	fireAndForgetMockCallsWG.Add(1)
	lastMockCall := mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult")).
		Return(map[string][]byte{}).Once()
	lastMockCall.RunFn = func(args mock.Arguments) {
		fireAndForgetMockCallsWG.Done()
	}

	ss.RegisterStorageObserver(mockobs)
	defer ss.UnregisterStorageObserver(mockobs)

	ts.Put([]byte("mackcom"), []byte("wuz here"))
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	// We need to wait since GetFilter and Update are called
	// in fire and forget goroutines. Not ideal, but it's out of scope
	// to refactor the production code in this commit.
	fireAndForgetMockCallsWG.Wait()
}

func TestStorageState_RegisterStorageObserver_Multi(t *testing.T) {
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	num := 5

	var mocks []*MockObserver

	for i := 0; i < num; i++ {
		mockfilter := map[string][]byte{}
		mockobs := NewMockObserver(t)

		mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult"))
		mockobs.On("GetID").Return(uint(10))
		mockobs.On("GetFilter").Return(mockfilter).Times(2)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
		require.NoError(t, err)
	}

	key1 := []byte("key1")
	value1 := []byte("value1")

	ts.Put(key1, value1)

	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range mocks {
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
	var mocks []*MockObserver
	filter := map[string][]byte{
		common.BytesToHex(key1): {},
	}

	for i := 0; i < num; i++ {
		mockobs := NewMockObserver(t)
		mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult"))
		mockobs.On("GetID").Return(uint(i))
		mockobs.On("GetFilter").Return(filter).Times(len(filter) + 3)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
	}

	ts.Put(key1, value1)
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range mocks {
		ss.UnregisterStorageObserver(observer)
	}
}

func Test_Example(t *testing.T) {
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
		if err := db.Subscribe(ctx, cb, prefix); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		log.Printf("subscription closed")
	}()

	// Write both keys, but only one should be printed in the Output.
	err := db.Put(aKey, aValue)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Put(bKey, bValue)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("stopping subscription")
	cancel()
	log.Printf("waiting for subscription to close")
	wg.Wait()
	// Output:
	// a-key is now set to a-value
}
