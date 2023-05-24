// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/trie/codec"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Encode encodes the node to the buffer given.
// The encoding format is documented in the README.md
// of this package, and specified in the Polkadot spec at
// https://spec.polkadot.network/#sect-state-storage
func (n *Node) Encode(buffer Buffer) (err error) {
	err = encodeHeader(n, buffer)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	if n == nil {
		// only encode the empty variant byte header
		return nil
	}

	keyLE := codec.NibblesToKeyLE(n.PartialKey)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	kind := n.Kind()
	nodeIsBranch := kind == Branch
	if nodeIsBranch {
		childrenBitmap := common.Uint16ToBytes(n.ChildrenBitmap())
		_, err = buffer.Write(childrenBitmap)
		if err != nil {
			return fmt.Errorf("cannot write children bitmap to buffer: %w", err)
		}
	}

	// Only encode node storage value if the node has a storage value,
	// even if it is empty. Do not encode if the branch is without value.
	// Note leaves and branches with value cannot have a `nil` storage value.
	if n.StorageValue != nil {
		encoder := scale.NewEncoder(buffer)
		err = encoder.Encode(n.StorageValue)
		if err != nil {
			return fmt.Errorf("scale encoding storage value: %w", err)
		}
	}

	if nodeIsBranch {
		err = encodeChildrenOpportunisticParallel(n.Children, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode children of branch: %w", err)
		}
	}

	return nil
}
