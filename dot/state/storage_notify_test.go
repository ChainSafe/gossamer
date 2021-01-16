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

func TestStorageState_RegisterStorageChangeChannel(t *testing.T) {
	ss := newTestStorageState(t)

	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	ch := make(chan *SubscriptionResult)
	sub := StorageSubscription{
		Filter:   make(map[string]bool),
		Listener: ch,
	}
	id, err := ss.RegisterStorageChangeChannel(sub)
	require.NoError(t, err)

	defer ss.UnregisterStorageChangeChannel(id)

	root, err := ts.Root()
	require.NoError(t, err)

	ts.Set([]byte("mackcom"), []byte("wuz here"))
	err = ss.StoreInDB(root)
	require.NoError(t, err)

	for i := 0; i < 1; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive storage change message")
		}
	}
}

func TestStorageState_RegisterStorageChangeChannel_Multi(t *testing.T) {
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	num := 5
	chs := make([]chan *SubscriptionResult, num)
	ids := make([]byte, num)

	for i := 0; i < num; i++ {
		chs[i] = make(chan *SubscriptionResult)
		sub := StorageSubscription{
			Listener: chs[i],
		}
		ids[i], err = ss.RegisterStorageChangeChannel(sub)
		require.NoError(t, err)
	}

	root, err := ts.Root()
	require.NoError(t, err)

	key1 := []byte("key1")
	value1 := []byte("value1")

	ts.Set(key1, value1)
	ts.Commit()

	err = ss.StoreTrie(root, ts)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(num)

	for i, ch := range chs {

		go func(i int, ch chan *SubscriptionResult) {
			select {
			case c := <-ch:
				require.NotNil(t, c.Hash)
				require.Equal(t, key1, c.Changes[0].Key)
				require.Equal(t, value1, c.Changes[0].Value)
				wg.Done()
			case <-time.After(testMessageTimeout):
				t.Error("did not receive storage change: ch=", i)
			}
		}(i, ch)

	}

	wg.Wait()

	for _, id := range ids {
		ss.UnregisterStorageChangeChannel(id)
	}
}

func TestStorageState_RegisterStorageChangeChannel_Multi_Filter(t *testing.T) {
	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	key1 := []byte("key1")
	value1 := []byte("value1")

	num := 5
	chs := make([]chan *SubscriptionResult, num)
	ids := make([]byte, num)
	subFilter := make(map[string]bool)
	subFilter[common.BytesToHex(key1)] = true

	for i := 0; i < num; i++ {
		chs[i] = make(chan *SubscriptionResult)
		sub := StorageSubscription{
			Filter:   subFilter,
			Listener: chs[i],
		}
		ids[i], err = ss.RegisterStorageChangeChannel(sub)
		require.NoError(t, err)
	}

	root, err := ts.Root()
	require.NoError(t, err)

	ts.Set(key1, value1)
	ts.Commit()

	err = ss.StoreTrie(root, ts)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(num)

	for i, ch := range chs {

		go func(i int, ch chan *SubscriptionResult) {
			select {
			case c := <-ch:
				require.NotNil(t, c.Hash)
				require.Equal(t, key1, c.Changes[0].Key)
				require.Equal(t, value1, c.Changes[0].Value)
				wg.Done()
			case <-time.After(testMessageTimeout):
				t.Error("did not receive storage change: ch=", i)
			}
		}(i, ch)

	}

	wg.Wait()

	for _, id := range ids {
		ss.UnregisterStorageChangeChannel(id)
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
