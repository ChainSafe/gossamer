// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"sync"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
<<<<<<< HEAD
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/dgraph-io/badger/v4/pb"
=======
>>>>>>> 344461dfca4d7cc341379cc778f4eeabd411e4cb
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestStorageState_RegisterStorageObserver(t *testing.T) {
	ctrl := gomock.NewController(t)

	ss := newTestStorageState(t)

	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	mockfilter := map[string][]byte{}
	mockobs := NewMockObserver(ctrl)

	mockobs.EXPECT().GetID().Return(uint(10)).Times(2)

	var fireAndForgetMockCallsWG sync.WaitGroup

	fireAndForgetMockCallsWG.Add(2)
	mockobs.EXPECT().GetFilter().DoAndReturn(func() map[string][]byte {
		defer fireAndForgetMockCallsWG.Done()
		return mockfilter
	}).Times(2)

	fireAndForgetMockCallsWG.Add(1)
	mockobs.EXPECT().Update(gomock.Any()).
		DoAndReturn(func(r *SubscriptionResult) map[string][]byte {
			defer fireAndForgetMockCallsWG.Done()
			return map[string][]byte{}
		})

	ss.RegisterStorageObserver(mockobs)
	defer ss.UnregisterStorageObserver(mockobs)

	ts.Put([]byte("mackcom"), []byte("wuz here"), trie.V0)
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	// We need to wait since GetFilter and Update are called
	// in fire and forget goroutines. Not ideal, but it's out of scope
	// to refactor the production code in this commit.
	fireAndForgetMockCallsWG.Wait()
}

func TestStorageState_RegisterStorageObserver_Multi(t *testing.T) {
	ctrl := gomock.NewController(t)

	ss := newTestStorageState(t)
	ts, err := ss.TrieState(nil)
	require.NoError(t, err)

	num := 5

	var mocks []*MockObserver

	for i := 0; i < num; i++ {
		mockfilter := map[string][]byte{}
		mockobs := NewMockObserver(ctrl)

		mockobs.EXPECT().Update(gomock.Any())
		mockobs.EXPECT().GetID().Return(uint(10)).Times(2)
		mockobs.EXPECT().GetFilter().Return(mockfilter).Times(2)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
		require.NoError(t, err)
	}

	key1 := []byte("key1")
	value1 := []byte("value1")

	ts.Put(key1, value1, trie.V0)

	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range mocks {
		ss.UnregisterStorageObserver(observer)
	}
}

func TestStorageState_RegisterStorageObserver_Multi_Filter(t *testing.T) {
	t.Skip() // this seems to fail often on CI\

	ctrl := gomock.NewController(t)
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
		mockobs := NewMockObserver(ctrl)
		mockobs.EXPECT().Update(gomock.Any())
		mockobs.EXPECT().GetID().Return(uint(i))
		mockobs.EXPECT().GetFilter().Return(filter).Times(len(filter) + 3)

		mocks = append(mocks, mockobs)
		ss.RegisterStorageObserver(mockobs)
	}

	ts.Put(key1, value1, trie.V0)
	err = ss.StoreTrie(ts, nil)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	for _, observer := range mocks {
		ss.UnregisterStorageObserver(observer)
	}
}
