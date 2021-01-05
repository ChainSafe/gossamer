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
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/dgraph-io/badger/v2"
	"github.com/golang/snappy"
	"github.com/stretchr/testify/require"
)

func TestStorageState_RegisterStorageChangeChannel(t *testing.T) {
	ss := newTestStorageState(t)

	ch := make(chan *KeyValue, 3)
	id, err := ss.RegisterStorageChangeChannel(ch)
	require.NoError(t, err)

	defer ss.UnregisterStorageChangeChannel(id)

	// three storage change events
	ss.setStorage(nil, []byte("mackcom"), []byte("wuz here"))
	ss.setStorage(nil, []byte("key1"), []byte("value1"))
	ss.setStorage(nil, []byte("key1"), []byte("value2"))

	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(testMessageTimeout):
			t.Fatal("did not receive storage change message")
		}
	}
}

func TestStorageState_RegisterStorageChangeChannel_Multi(t *testing.T) {
	ss := newTestStorageState(t)

	num := 5
	chs := make([]chan *KeyValue, num)
	ids := make([]byte, num)

	var err error
	for i := 0; i < num; i++ {
		chs[i] = make(chan *KeyValue)
		ids[i], err = ss.RegisterStorageChangeChannel(chs[i])
		require.NoError(t, err)
	}

	key1 := []byte("key1")
	ss.setStorage(nil, key1, []byte("value1"))

	var wg sync.WaitGroup
	wg.Add(num)

	for i, ch := range chs {

		go func(i int, ch chan *KeyValue) {
			select {
			case c := <-ch:
				require.Equal(t, key1, c.Key)
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

func ExampleDB_Subscribe() {
	// this is a working example of how to use db.Subscribe taken from
	// https://github.com/dgraph-io/badger/blob/f50343ff404d8198df6dc83755ec2eab863d5ff2/db_test.go#L1939-L1948
	// Note, this works as expected, however when I change db to use chaindb (ChainSafe's badger fork)
	// it fails (See TestSubscribe_A, B and C).  The issue seems to be related to snappy.Encode
	// of keys when they are stored when using chaindb.
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
	db, err := badger.Open(badger.DefaultOptions(dir))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create the context here so we can cancel it after sending the writes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the WaitGroup to make sure we wait for the subscription to stop before continuing.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(kvs *badger.KVList) error {
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
	err = db.Update(func(txn *badger.Txn) error { return txn.Set(aKey, aValue) })
	if err != nil {
		log.Fatal(err)
	}
	err = db.Update(func(txn *badger.Txn) error { return txn.Set(bKey, bValue) })
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

func Test_Subscribe_A(t *testing.T) {
	prefix := []byte{}

	// Both these keys should be printed since prefix is empty
	aKey := []byte("a-key")
	aValue := []byte("a-value")

	bKey := []byte("b-key")
	bValue := []byte("b-value")

	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(testDatadirPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(testDatadirPath)
	})

	// create TrieState to handle storage
	s, err := NewTrieState(db, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Create the context here so we can cancel it after sending the writes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the WaitGroup to make sure we wait for the subscription to stop before continuing.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(kvs *badger.KVList) error {
			for _, kv := range kvs.Kv {
				log.Printf("%s is now set to %s\n", kv.Key, kv.Value)
			}
			return nil
		}
		if err = db.Subscribe(ctx, cb, prefix); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		log.Printf("subscription closed")
	}()

	// Write both keys, both should be printed since prefix is empty
	err = s.Set(aKey, aValue)
	require.NoError(t, err)

	err = s.Set(bKey, bValue)
	require.NoError(t, err)

	// print results of key lookup (to confirm they were stored)
	res, err := s.Get(aKey)
	require.NoError(t, err)
	log.Printf("a-key stored value %s\n", res)

	res, err = s.Get(bKey)
	require.NoError(t, err)
	log.Printf("b-key stored value %s\n", res)

	log.Printf("stopping subscription")
	cancel()
	log.Printf("waiting for subscription to close")
	wg.Wait()
}

func Test_Subscribe_B(t *testing.T) {
	prefix := []byte{'a'}

	// This key should be printed, since it matches the prefix.
	aKey := []byte("a-key")
	aValue := []byte("a-value")

	// This key should not be printed.
	bKey := []byte("b-key")
	bValue := []byte("b-value")

	//  NOTE: none of the keys are printed because when they are stored the key (and value)
	// are snappy.Encode, but the subscribe prefix is not See Test_Substribe_C for encode prefix test

	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(testDatadirPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(testDatadirPath)
	})

	// create TrieState to handle storage
	s, err := NewTrieState(db, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Create the context here so we can cancel it after sending the writes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the WaitGroup to make sure we wait for the subscription to stop before continuing.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(kvs *badger.KVList) error {
			for _, kv := range kvs.Kv {
				log.Printf("%s is now set to %s\n", kv.Key, kv.Value)
			}
			return nil
		}
		if err = db.Subscribe(ctx, cb, prefix); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		log.Printf("subscription closed")
	}()

	// Write both keys, both should be printed since prefix is empty
	err = s.Set(aKey, aValue)
	require.NoError(t, err)

	err = s.Set(bKey, bValue)
	require.NoError(t, err)

	// print results of key lookup (to confirm they were stored)
	res, err := s.Get(aKey)
	require.NoError(t, err)
	log.Printf("a-key stored value %s\n", res)

	res, err = s.Get(bKey)
	require.NoError(t, err)
	log.Printf("b-key stored value %s\n", res)

	log.Printf("stopping subscription")
	cancel()
	log.Printf("waiting for subscription to close")
	wg.Wait()
}

func Test_Subscribe_C(t *testing.T) {
	prefix := snappy.Encode(nil, []byte("a"))

	// This key should be printed, since it matches the prefix.
	aKey := []byte("a-key")
	aValue := []byte("a-value")

	// This key should not be printed.
	bKey := []byte("b-key")
	bValue := []byte("b-value")

	//  NOTE: none of the keys are printed because when they are stored the key (and value)
	// are snappy.Encode, even tho we snappy.Encode the prefix it still doen't print these because
	// of the way snappy.Encode works (see Test_SnappyEncode)

	testDatadirPath, err := ioutil.TempDir("/tmp", "test-datadir-*")
	require.NoError(t, err)

	db, err := chaindb.NewBadgerDB(testDatadirPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(testDatadirPath)
	})

	// create TrieState to handle storage
	s, err := NewTrieState(db, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Create the context here so we can cancel it after sending the writes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the WaitGroup to make sure we wait for the subscription to stop before continuing.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb := func(kvs *badger.KVList) error {
			for _, kv := range kvs.Kv {
				log.Printf("%s is now set to %s\n", kv.Key, kv.Value)
			}
			return nil
		}
		if err = db.Subscribe(ctx, cb, prefix); err != nil && err != context.Canceled {
			log.Fatal(err)
		}
		log.Printf("subscription closed")
	}()

	// Write both keys, both should be printed since prefix is empty
	err = s.Set(aKey, aValue)
	require.NoError(t, err)

	err = s.Set(bKey, bValue)
	require.NoError(t, err)

	// print results of key lookup (to confirm they were stored)
	res, err := s.Get(aKey)
	require.NoError(t, err)
	log.Printf("a-key stored value %s\n", res)

	res, err = s.Get(bKey)
	require.NoError(t, err)
	log.Printf("b-key stored value %s\n", res)

	log.Printf("stopping subscription")
	cancel()
	log.Printf("waiting for subscription to close")
	wg.Wait()
}

func Test_SnappyEncode(t *testing.T) {
	encA := snappy.Encode(nil, []byte("a"))
	fmt.Printf("encode     a: %v\n", encA)

	encAkey := snappy.Encode(nil, []byte("a-key"))
	fmt.Printf("encode a-key: %v\n", encAkey)

	// this is why db.Subscribe is not working as expected, since the keys are snappy.Encode
	// when stored the prefix changes, so when I subscribe with prefix a, the system is looking
	// for key changes that start with a (97), but all the keys were stored with a different
	// prefix since they were encoded.
}
