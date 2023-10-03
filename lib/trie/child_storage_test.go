// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutAndGetChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie, V0)
	assert.NoError(t, err)

	childTrieRes, err := parentTrie.GetChild(childKey)
	assert.NoError(t, err)

	assert.Equal(t, childTrie, childTrieRes)
}

func TestPutAndDeleteChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie, V0)
	assert.NoError(t, err)

	err = parentTrie.DeleteChild(childKey)
	assert.NoError(t, err)

	_, err = parentTrie.GetChild(childKey)
	assert.ErrorContains(t, err, "child trie does not exist at key")
}

func TestPutAndClearFromChild(t *testing.T) {
	childKey := []byte("default")
	keyInChild := []byte{0x01, 0x35}
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie, V0)
	assert.NoError(t, err)

	err = parentTrie.ClearFromChild(childKey, keyInChild, V0)
	assert.NoError(t, err)

	childTrie, err = parentTrie.GetChild(childKey)
	assert.NoError(t, err)

	value := childTrie.Get(keyInChild)
	assert.Equal(t, []uint8(nil), value)
}

func TestPutAndGetFromChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie()
	parentTrie := NewEmptyTrie()

	err := parentTrie.SetChild(childKey, childTrie, V0)
	assert.NoError(t, err)

	testKey := []byte("child_key")
	testValue := []byte("child_value")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue, V0)
	assert.NoError(t, err)

	valueRes, err := parentTrie.GetFromChild(childKey, testKey)
	assert.NoError(t, err)

	assert.Equal(t, valueRes, testValue)

	testKey = []byte("child_key_again")
	testValue = []byte("child_value_again")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue, V0)
	assert.NoError(t, err)

	valueRes, err = parentTrie.GetFromChild(childKey, testKey)
	assert.NoError(t, err)

	assert.Equal(t, valueRes, testValue)
}

func TestChildTrieHashAfterClear(t *testing.T) {
	trieThatHoldsAChildTrie := NewEmptyTrie()
	originalEmptyHash := trieThatHoldsAChildTrie.MustHash()

	keyToChild := []byte("crowdloan")
	keyInChild := []byte("account-alice")
	contributed := uint64(1000)
	contributedWith := make([]byte, 8)
	binary.BigEndian.PutUint64(contributedWith, contributed)

	err := trieThatHoldsAChildTrie.PutIntoChild(keyToChild, keyInChild, contributedWith, V0)
	require.NoError(t, err)

	// the parent trie hash SHOULT NOT BE EQUAL to the original
	// empty hash since it contains a value
	require.NotEqual(t, originalEmptyHash, trieThatHoldsAChildTrie.MustHash())

	// ensure the value is inside the child trie
	valueStored, err := trieThatHoldsAChildTrie.GetFromChild(keyToChild, keyInChild)
	require.NoError(t, err)
	require.Equal(t, contributed, binary.BigEndian.Uint64(valueStored))

	// clear child trie key value
	err = trieThatHoldsAChildTrie.ClearFromChild(keyToChild, keyInChild, V0)
	require.NoError(t, err)

	// the parent trie hash SHOULD BE EQUAL to the original
	// empty hash since now it does not have any other value in it
	require.Equal(t, originalEmptyHash, trieThatHoldsAChildTrie.MustHash())

}
