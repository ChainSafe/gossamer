// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPutAndGetChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie)
	if err != nil {
		t.Fatal(err)
	}

	childTrieRes, err := parentTrie.GetChild(childKey)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(childTrie, childTrieRes) {
		t.Fatalf("Fail: got %v expected %v", childTrieRes, childTrie)
	}
}

func TestPutAndGetFromChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie)
	if err != nil {
		t.Fatal(err)
	}

	testKey := []byte("child_key")
	testValue := []byte("child_value")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue)
	if err != nil {
		t.Fatal(err)
	}

	valueRes, err := parentTrie.GetFromChild(childKey, testKey)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(valueRes, testValue) {
		t.Fatalf("Fail: got %x expected %x", valueRes, testValue)
	}

	testKey = []byte("child_key_again")
	testValue = []byte("child_value_again")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue)
	if err != nil {
		t.Fatal(err)
	}

	valueRes, err = parentTrie.GetFromChild(childKey, testKey)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(valueRes, testValue) {
		t.Fatalf("Fail: got %x expected %x", valueRes, testValue)
	}
}

func TestChildTrieHashAfterClear(t *testing.T) {
	trieThatHoldsAChildTrie := NewEmptyTrie()
	originalEmptyHash := trieThatHoldsAChildTrie.MustHash()

	keyToChild := []byte("crowdloan")
	keyInChild := []byte("account-alice")
	contributed := uint64(1000)
	contributedWith := make([]byte, 8)
	binary.BigEndian.PutUint64(contributedWith, contributed)

	err := trieThatHoldsAChildTrie.PutIntoChild(keyToChild, keyInChild, contributedWith)
	require.NoError(t, err)

	// the parent trie hash SHOULT NOT BE EQUAL to the original
	// empty hash since it contains a value
	require.NotEqual(t, originalEmptyHash, trieThatHoldsAChildTrie.MustHash())

	// ensure the value is inside the child trie
	valueStored, err := trieThatHoldsAChildTrie.GetFromChild(keyToChild, keyInChild)
	require.NoError(t, err)
	require.Equal(t, contributed, binary.BigEndian.Uint64(valueStored))

	// clear child trie key value
	err = trieThatHoldsAChildTrie.ClearFromChild(keyToChild, keyInChild)
	require.NoError(t, err)

	// the parent trie hash SHOULD BE EQUAL to the original
	// empty hash since now it does not have any other value in it
	require.Equal(t, originalEmptyHash, trieThatHoldsAChildTrie.MustHash())

}
