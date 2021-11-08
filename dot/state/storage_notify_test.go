// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStorageState_RegisterStorageObserver(t *testing.T) {
	ss := newTestStorageState(t)

	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	mockfilter := map[string][]byte{}
	mockobs := &MockObserver{}

	mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult"))
	mockobs.On("GetID").Return(uint(10))
	mockobs.On("GetFilter").Return(mockfilter)

	ss.RegisterStorageObserver(mockobs)
	defer ss.UnregisterStorageObserver(mockobs)

	ts.Set([]byte("mackcom"), []byte("wuz here"))
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	expectedResult := &SubscriptionResult{
		Hash: ts.MustRoot(),
		Changes: []KeyValue{{
			Key:   []byte("mackcom"),
			Value: []byte("wuz here"),
		}},
	}

	time.Sleep(time.Millisecond)
	// called when register and called when store trie
	mockobs.AssertNumberOfCalls(t, "GetFilter", 2)
	mockobs.AssertNumberOfCalls(t, "Update", 1)
	mockobs.AssertCalled(t, "Update", expectedResult)
}

func TestStorageState_RegisterStorageObserver_Multi(t *testing.T) {
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	num := 5

	var mocks []*MockObserver

	for i := 0; i < num; i++ {
		mockfilter := map[string][]byte{}
		mockobs := &MockObserver{}

		mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult"))
		mockobs.On("GetID").Return(uint(10))
		mockobs.On("GetFilter").Return(mockfilter)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
		require.NoError(t, err)
	}

	key1 := []byte("key1")
	value1 := []byte("value1")

	ts.Set(key1, value1)

	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	expectedResult := &SubscriptionResult{
		Hash: ts.MustRoot(),
		Changes: []KeyValue{{
			Key:   key1,
			Value: value1,
		}},
	}

	for _, mockobs := range mocks {
		mockobs.AssertNumberOfCalls(t, "GetFilter", 2)
		mockobs.AssertNumberOfCalls(t, "Update", 1)
		mockobs.AssertCalled(t, "Update", expectedResult)
	}

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
		mockobs := &MockObserver{}
		mockobs.On("Update", mock.AnythingOfType("*state.SubscriptionResult"))
		mockobs.On("GetID").Return(uint(i))
		mockobs.On("GetFilter").Return(filter)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
	}

	ts.Set(key1, value1)
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	expectedResult := &SubscriptionResult{
		Hash: ts.MustRoot(),
		Changes: []KeyValue{{
			Key:   key1,
			Value: value1,
		}},
	}

	for _, mockobs := range mocks {
		mockobs.AssertNumberOfCalls(t, "GetFilter", len(filter)+3)
		mockobs.AssertNumberOfCalls(t, "Update", 1)
		mockobs.AssertCalled(t, "Update", expectedResult)
	}

	for _, observer := range mocks {
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
