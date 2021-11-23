// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"math/rand"
	"testing"
)

func generateRandBytes(size int) []byte {
	r := *rand.New(rand.NewSource(rand.Int63()))
	buf := make([]byte, r.Intn(size)+1)
	r.Read(buf)
	return buf
}

func generateRand(size int) [][]byte {
	rt := make([][]byte, size)
	r := *rand.New(rand.NewSource(rand.Int63()))
	for i := range rt {
		buf := make([]byte, r.Intn(379)+1)
		r.Read(buf)
		rt[i] = buf
	}
	return rt
}

func TestNewHasher(t *testing.T) {
	hasher := newHasher(false)
	defer hasher.returnToPool()

	_, err := hasher.hash.Write([]byte("noot"))
	if err != nil {
		t.Error(err)
	}

	sum := hasher.hash.Sum(nil)
	if sum == nil {
		t.Error("did not sum hash")
	}

	hasher.hash.Reset()
}

func TestHashLeaf(t *testing.T) {
	hasher := newHasher(false)
	defer hasher.returnToPool()

	n := &leaf{key: generateRandBytes(380), value: generateRandBytes(64)}
	h, err := hasher.Hash(n)
	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash leaf node: nil")
	}
}

func TestHashBranch(t *testing.T) {
	hasher := newHasher(false)
	defer hasher.returnToPool()

	n := &branch{key: generateRandBytes(380), value: generateRandBytes(380)}
	n.children[3] = &leaf{key: generateRandBytes(380), value: generateRandBytes(380)}
	h, err := hasher.Hash(n)
	if err != nil {
		t.Errorf("did not hash branch node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash branch node: nil")
	}
}

func TestHashShort(t *testing.T) {
	hasher := newHasher(false)
	defer hasher.returnToPool()

	n := &leaf{key: generateRandBytes(2), value: generateRandBytes(3)}
	expected, err := hasher.encode(n)
	if err != nil {
		t.Fatal(err)
	}

	h, err := hasher.Hash(n)
	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash leaf node: nil")
	} else if !bytes.Equal(h[:], expected) {
		t.Errorf("did not return encoded node padded to 32 bytes: got %s", h)
	}
}
