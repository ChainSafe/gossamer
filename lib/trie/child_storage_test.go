// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestPutAndGetChild(t *testing.T) {
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)

	childKey := []byte("default")
	childTrie := buildSmallTrie(metrics)
	parentTrie := NewEmptyTrie(metrics)

	metrics.EXPECT().NodesAdd(uint32(1))
	err := parentTrie.PutChild(childKey, childTrie)
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
	ctrl := gomock.NewController(t)

	metrics := NewMockMetrics(ctrl)

	childKey := []byte("default")
	childTrie := buildSmallTrie(metrics)
	parentTrie := NewEmptyTrie(metrics)

	metrics.EXPECT().NodesAdd(uint32(1))

	err := parentTrie.PutChild(childKey, childTrie)
	if err != nil {
		t.Fatal(err)
	}

	testKey := []byte("child_key")
	testValue := []byte("child_value")
	metrics.EXPECT().NodesAdd(uint32(1))
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

	metrics.EXPECT().NodesAdd(uint32(1))
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
