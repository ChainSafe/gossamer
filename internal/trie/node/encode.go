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
	// See https://spec.polkadot.network/#defn-node-subvalue
	// See https://github.com/paritytech/substrate/blob/a7ba55d3c54b9957c242f659e02f5b5a0f47b3ff/primitives/trie/src/node_codec.rs#L123
	if !nodeIsBranch || (nodeIsBranch && n.SubValue != nil) {
		encoder := scale.NewEncoder(buffer)
		// Note: scale encoding `[]byte(nil)` and `[]byte{}` result in the same `[]byte{0}`,
		// that's why it's ok to encode a nil value for leaf nodes.
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
