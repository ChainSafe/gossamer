// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package inmemory

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/pkg/trie/codec"
	"github.com/stretchr/testify/require"
)

func TestInMemoryTrieIterator(t *testing.T) {
	tt := NewEmptyTrie()

	tt.Put([]byte("some_other_storage:XCC:ZZZ"), []byte("0x10"))
	tt.Put([]byte("yet_another_storage:BLABLA:YYY:JJJ"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:AAA"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:CCC"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:DDD"), []byte("0x10"))
	tt.Put([]byte("account_storage:JJK:EEE"), []byte("0x10"))

	iter := NewInMemoryTrieIterator(WithTrie(tt))
	require.Equal(t, []byte("account_storage:ABC:AAA"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Equal(t, []byte("account_storage:ABC:CCC"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Equal(t, []byte("account_storage:ABC:DDD"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Equal(t, []byte("account_storage:JJK:EEE"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Equal(t, []byte("some_other_storage:XCC:ZZZ"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Equal(t, []byte("yet_another_storage:BLABLA:YYY:JJJ"), codec.NibblesToKeyLE((iter.NextEntry().Key)))
	require.Nil(t, iter.NextEntry())
}

func TestInMemoryIteratorGetAllKeysWithPrefix(t *testing.T) {
	tt := NewEmptyTrie()

	tt.Put([]byte("services_storage:serviceA:19090"), []byte("0x10"))
	tt.Put([]byte("services_storage:serviceB:22222"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:AAA"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:CCC"), []byte("0x10"))
	tt.Put([]byte("account_storage:ABC:DDD"), []byte("0x10"))
	tt.Put([]byte("account_storage:JJK:EEE"), []byte("0x10"))

	prefix := []byte("account_storage")
	iter := tt.PrefixedIter(prefix)

	keys := make([][]byte, 0)
	for key := iter.NextKey(); bytes.HasPrefix(key, prefix); key = iter.NextKey() {
		keys = append(keys, key)
	}

	expectedKeys := [][]byte{
		[]byte("account_storage:ABC:AAA"),
		[]byte("account_storage:ABC:CCC"),
		[]byte("account_storage:ABC:DDD"),
		[]byte("account_storage:JJK:EEE"),
	}

	require.Equal(t, expectedKeys, keys)
}
