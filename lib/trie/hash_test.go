// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateRandBytes(size int) []byte {
	buf := make([]byte, rand.Intn(size)+1)
	rand.Read(buf)
	return buf
}

func generateRand(size int) [][]byte {
	rt := make([][]byte, size)
	for i := range rt {
		buf := make([]byte, rand.Intn(379)+1)
		rand.Read(buf)
		rt[i] = buf
	}
	return rt
}

func TestHashLeaf(t *testing.T) {
	n := &leaf{key: generateRandBytes(380), value: generateRandBytes(64)}

	buffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, buffer, parallel)

	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if buffer.Len() == 0 {
		t.Errorf("did not hash leaf node: nil")
	}
}

func TestHashBranch(t *testing.T) {
	n := &branch{key: generateRandBytes(380), value: generateRandBytes(380)}
	n.children[3] = &leaf{key: generateRandBytes(380), value: generateRandBytes(380)}

	buffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, buffer, parallel)

	if err != nil {
		t.Errorf("did not hash branch node: %s", err)
	} else if buffer.Len() == 0 {
		t.Errorf("did not hash branch node: nil")
	}
}

func TestHashShort(t *testing.T) {
	n := &leaf{
		key:   generateRandBytes(2),
		value: generateRandBytes(3),
	}

	encodingBuffer := bytes.NewBuffer(nil)
	const parallel = false
	err := encodeNode(n, encodingBuffer, parallel)
	require.NoError(t, err)

	digestBuffer := bytes.NewBuffer(nil)
	err = hashNode(n, digestBuffer)
	require.NoError(t, err)
	assert.Equal(t, encodingBuffer.Bytes(), digestBuffer.Bytes())
}
