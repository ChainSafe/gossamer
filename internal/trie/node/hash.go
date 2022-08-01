// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"

	"github.com/ChainSafe/gossamer/internal/trie/pools"
	"github.com/ChainSafe/gossamer/lib/common"
)

// EncodeAndHash returns the encoding of the node and
// the Merkle value of the node.
func (n *Node) EncodeAndHash() (encoding, hash []byte, err error) {
	if !n.Dirty && n.Encoding != nil && n.HashDigest != nil {
		return n.Encoding, n.HashDigest, nil
	}

	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = n.Encode(buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	n.Encoding = make([]byte, len(bufferBytes))
	copy(n.Encoding, bufferBytes)
	encoding = n.Encoding // no need to copy

	if buffer.Len() < 32 {
		n.HashDigest = make([]byte, len(bufferBytes))
		copy(n.HashDigest, bufferBytes)
		hash = n.HashDigest // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	n.HashDigest = hashArray[:]
	hash = n.HashDigest // no need to copy

	return encoding, hash, nil
}

// EncodeAndHashRoot returns the encoding of the root node and
// the Merkle value of the root node (the hash of its encoding).
func (n *Node) EncodeAndHashRoot() (encoding, hash []byte, err error) {
	if !n.Dirty && n.Encoding != nil && n.HashDigest != nil {
		return n.Encoding, n.HashDigest, nil
	}

	buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
	buffer.Reset()
	defer pools.EncodingBuffers.Put(buffer)

	err = n.Encode(buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	n.Encoding = make([]byte, len(bufferBytes))
	copy(n.Encoding, bufferBytes)
	encoding = n.Encoding // no need to copy

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	n.HashDigest = hashArray[:]
	hash = n.HashDigest // no need to copy

	return encoding, hash, nil
}
