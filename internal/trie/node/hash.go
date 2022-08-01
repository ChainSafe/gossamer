// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"
	"hash"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/pools"
)

// MerkleValue writes the Merkle value from the encoding of a non-root
// node to the writer given.
// If the encoding is less or equal to 32 bytes, the Merkle value is the encoding.
// Otherwise, the Merkle value is the Blake2b hash digest of the encoding.
func MerkleValue(encoding []byte, writer io.Writer) (err error) {
	if len(encoding) < 32 {
		_, err = writer.Write(encoding)
		if err != nil {
			return fmt.Errorf("writing encoding: %w", err)
		}
		return nil
	}

	hasher := pools.Hashers.Get().(hash.Hash)
	hasher.Reset()
	defer pools.Hashers.Put(hasher)

	_, err = hasher.Write(encoding)
	if err != nil {
		return fmt.Errorf("hashing encoding: %w", err)
	}

	digest := hasher.Sum(nil)
	_, err = writer.Write(digest)
	if err != nil {
		return fmt.Errorf("writing digest: %w", err)
	}

	return nil
}

// MerkleValueRoot writes the Merkle value for the root of the trie
// to the writer given as argument.
// The Merkle value is the Blake2b hash of the encoding of the root node.
func MerkleValueRoot(rootEncoding []byte, writer io.Writer) (err error) {
	hasher := pools.Hashers.Get().(hash.Hash)
	hasher.Reset()
	defer pools.Hashers.Put(hasher)

	_, err = hasher.Write(rootEncoding)
	if err != nil {
		return fmt.Errorf("hashing encoding: %w", err)
	}

	digest := hasher.Sum(nil)
	_, err = writer.Write(digest)
	if err != nil {
		return fmt.Errorf("writing digest: %w", err)
	}

	return nil
}

// CalculateMerkleValue returns the Merkle value of the node.
// If the node encoding is less or equal to 32 bytes,
// the encoding is the Merkle value.
// Otherwise, the Blake2b hash digest of the encoding
// is returned as the Merkle value.
func (n *Node) CalculateMerkleValue(isRoot bool) (merkleValue []byte, err error) {
	if !n.Dirty && n.MerkleValue != nil {
		return n.MerkleValue, nil
	}

	if isRoot {
		_, merkleValue, err = n.EncodeAndHashRoot()
		if err != nil {
			return nil, fmt.Errorf("encoding and hashing root node: %w", err)
		}
		return merkleValue, nil
	}

	_, merkleValue, err = n.EncodeAndHash()
	if err != nil {
		return nil, fmt.Errorf("encoding and hashing node: %w", err)
	}
	return merkleValue, nil
}

// EncodeAndHash returns the encoding of the node and the
// Merkle value of the node. See the `MerkleValue` method for
// more details on the value of the Merkle value.
// TODO change this function to write to an encoding writer
// and a merkle value writer, such that buffer sync pools can be used
// by the caller.
func (n *Node) EncodeAndHash() (encoding, merkleValue []byte, err error) {
	if !n.Dirty && n.Encoding != nil && n.MerkleValue != nil {
		return n.Encoding, n.MerkleValue, nil
	}

	if n.Dirty || n.Encoding == nil {
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()
		defer pools.EncodingBuffers.Put(buffer)

		err = n.Encode(buffer)
		if err != nil {
			return nil, nil, fmt.Errorf("encoding node: %w", err)
		}

		bufferBytes := buffer.Bytes()

		// TODO remove this copying since it defeats the purpose of `buffer`
		// and the sync.Pool.
		n.Encoding = make([]byte, len(bufferBytes))
		copy(n.Encoding, bufferBytes)
	}
	encoding = n.Encoding // no need to copy

	const maxMerkleValueSize = 32
	merkleValueBuffer := bytes.NewBuffer(make([]byte, 0, maxMerkleValueSize))
	err = MerkleValue(encoding, merkleValueBuffer)
	if err != nil {
		return nil, nil, fmt.Errorf("merkle value: %w", err)
	}
	merkleValue = merkleValueBuffer.Bytes()
	n.MerkleValue = merkleValue // no need to copy

	return encoding, merkleValue, nil
}

// EncodeAndHashRoot returns the encoding of the node and the
// Merkle value of the node. See the `MerkleValueRoot` method
// for more details on the value of the Merkle value.
// TODO change this function to write to an encoding writer
// and a merkle value writer, such that buffer sync pools can be used
// by the caller.
func (n *Node) EncodeAndHashRoot() (encoding, merkleValue []byte, err error) {
	const rootMerkleValueLength = 32
	if !n.Dirty && n.Encoding != nil && len(n.MerkleValue) == rootMerkleValueLength {
		return n.Encoding, n.MerkleValue, nil
	}

	if n.Dirty || n.Encoding == nil {
		buffer := pools.EncodingBuffers.Get().(*bytes.Buffer)
		buffer.Reset()
		defer pools.EncodingBuffers.Put(buffer)

		err = n.Encode(buffer)
		if err != nil {
			return nil, nil, fmt.Errorf("encoding node: %w", err)
		}

		bufferBytes := buffer.Bytes()

		// TODO remove this copying since it defeats the purpose of `buffer`
		// and the sync.Pool.
		n.Encoding = make([]byte, len(bufferBytes))
		copy(n.Encoding, bufferBytes)
	}
	encoding = n.Encoding // no need to copy

	const merkleValueSize = 32
	merkleValueBuffer := bytes.NewBuffer(make([]byte, 0, merkleValueSize))
	err = MerkleValueRoot(encoding, merkleValueBuffer)
	if err != nil {
		return nil, nil, fmt.Errorf("merkle value: %w", err)
	}
	merkleValue = merkleValueBuffer.Bytes()
	n.MerkleValue = merkleValue // no need to copy

	return encoding, merkleValue, nil
}
