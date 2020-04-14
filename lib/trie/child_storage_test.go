// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package trie

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPutAndGetChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie(t)
	parentTrie := NewEmptyTrie()

	err := parentTrie.PutChild(childKey, childTrie)
	require.Nil(t, err)

	childTrieRes, err := parentTrie.GetChild(childKey)
	require.Nil(t, err)

	if !reflect.DeepEqual(childTrie, childTrieRes) {
		t.Fatalf("Fail: got %v expected %v", childTrieRes, childTrie)
	}
}

func TestPutAndGetFromChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie(t)
	parentTrie := NewEmptyTrie()

	err := parentTrie.PutChild(childKey, childTrie)
	require.Nil(t, err)

	testKey := []byte("child_key")
	testValue := []byte("child_value")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue)
	require.Nil(t, err)

	valueRes, err := parentTrie.GetFromChild(childKey, testKey)
	require.Nil(t, err)

	if !bytes.Equal(valueRes, testValue) {
		t.Fatalf("Fail: got %x expected %x", valueRes, testValue)
	}

	testKey = []byte("child_key_again")
	testValue = []byte("child_value_again")
	err = parentTrie.PutIntoChild(childKey, testKey, testValue)
	require.Nil(t, err)

	valueRes, err = parentTrie.GetFromChild(childKey, testKey)
	require.Nil(t, err)

	if !bytes.Equal(valueRes, testValue) {
		t.Fatalf("Fail: got %x expected %x", valueRes, testValue)
	}
}
