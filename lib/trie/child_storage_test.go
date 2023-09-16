// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"encoding/binary"
	"fmt"
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

	err = parentTrie.ClearFromChild(childKey, keyInChild)
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
	fmt.Printf("Empty parent trie %s\n", trieThatHoldsAChildTrie.MustHash().String())

	keyToChild := []byte("crowdloan")
	keyInChild := []byte("account-alice")

	contributed := uint64(1000)
	contributedWith := make([]byte, 8)
	binary.BigEndian.PutUint64(contributedWith, contributed)

	err := trieThatHoldsAChildTrie.PutIntoChild(keyToChild, keyInChild, contributedWith, V0)
	require.NoError(t, err)

	fmt.Printf("With Value: Parent Trie Hash %s\n", trieThatHoldsAChildTrie.MustHash().String())
	fmt.Printf("Child Trie Hash %s\n\n", trieThatHoldsAChildTrie.MustGetChild(keyToChild).MustHash().String())

	valueStored, err := trieThatHoldsAChildTrie.GetFromChild(keyToChild, keyInChild)
	require.NoError(t, err)
	require.Equal(t, contributed, binary.BigEndian.Uint64(valueStored))

	// clear child trie key value
	err = trieThatHoldsAChildTrie.ClearFromChild(keyToChild, keyInChild)
	require.NoError(t, err)

	fmt.Printf("After Clear Parent Trie Hash %s\n", trieThatHoldsAChildTrie.MustHash().String())
	//fmt.Printf("After Clear Child Trie Hash %s\n", trieThatHoldsAChildTrie.MustGetChild(keyToChild).MustHash().String())
}
