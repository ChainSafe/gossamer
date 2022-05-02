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
// The encoding format is documented in encode_doc.go.
func (n *Node) Encode(buffer Buffer) (err error) {
	if !n.Dirty && n.Encoding != nil {
		_, err = buffer.Write(n.Encoding)
		if err != nil {
			return fmt.Errorf("cannot write stored encoding to buffer: %w", err)
		}
		return nil
	}

	err = encodeHeader(n, buffer)
	if err != nil {
		return fmt.Errorf("cannot encode header: %w", err)
	}

	keyLE := codec.NibblesToKeyLE(n.Key)
	_, err = buffer.Write(keyLE)
	if err != nil {
		return fmt.Errorf("cannot write LE key to buffer: %w", err)
	}

	if n.Type() == Branch {
		childrenBitmap := common.Uint16ToBytes(n.ChildrenBitmap())
		_, err = buffer.Write(childrenBitmap)
		if err != nil {
			return fmt.Errorf("cannot write children bitmap to buffer: %w", err)
		}
	}

	// check value is not nil for branch nodes, even though
	// leaf nodes always have a non-nil value.
	if n.Type() == Leaf || n.Value != nil {
		// TODO remove `n.Type() == Leaf` and update tests
		encodedValue, err := scale.Marshal(n.Value) // TODO scale encoder to write to buffer
		if err != nil {
			return fmt.Errorf("cannot scale encode value: %w", err)
		}

		_, err = buffer.Write(encodedValue)
		if err != nil {
			return fmt.Errorf("cannot write scale encoded value to buffer: %w", err)
		}
	}

	if n.Type() == Branch {
		err = encodeChildrenOpportunisticParallel(n.Children, buffer)
		if err != nil {
			return fmt.Errorf("cannot encode children of branch: %w", err)
		}
	}

	if n.Type() == Leaf {
		// TODO cache this for branches too and update test cases.
		// TODO remove this copying since it defeats the purpose of `buffer`
		// and the sync.Pool.
		n.Encoding = make([]byte, buffer.Len())
		copy(n.Encoding, buffer.Bytes())
	}

	return nil
}
