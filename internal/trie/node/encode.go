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

	// Only encode node value if the node is a leaf or
	// the node is a branch with a non empty value.
	if !nodeIsBranch || (nodeIsBranch && n.SubValue != nil) {
		encoder := scale.NewEncoder(buffer)
		err = encoder.Encode(n.SubValue)
		if err != nil {
			return fmt.Errorf("scale encoding value: %w", err)
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
