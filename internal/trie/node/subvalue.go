// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
)

// StorageValueEqual returns true if the node storage value is equal to the
// storage value given as argument. In particular, it returns false
// if one storage value is nil and the other storage value is the empty slice.
func (n Node) StorageValueEqual(storageValue []byte, database Getter) (equal bool, err error) {
	if len(storageValue) == 0 && len(n.StorageValue) == 0 {
		return (storageValue == nil && n.StorageValue == nil) ||
			(storageValue != nil && n.StorageValue != nil), nil
	}

	nodeStorageValue, err := n.GetStorageValue(database)
	if err != nil {
		return false, fmt.Errorf("getting node storage value from database: %w", err)
	}

	return bytes.Equal(nodeStorageValue, storageValue), nil
}

func (n *Node) GetStorageValue(database Getter) (storageValue []byte, err error) {
	if n.StorageValueInlined {
		return n.StorageValue, nil
	}

	subValueDigest := n.StorageValue
	storageValue, err = database.Get(subValueDigest)
	if err != nil {
		return nil, fmt.Errorf("getting storage value from database: %w", err)
	}

	return storageValue, nil
}

func (n *Node) SetStorageValue(storageValue []byte, database Putter,
	deltas DeltaSubValueRecorder) (err error) {
	if len(storageValue) <= 32 {
		n.registerDeletedSubValue(deltas)
		n.StorageValueInlined = true
		n.StorageValue = storageValue
		return nil
	}

	// Note we don't use a buffer from the pool since setter is an interace
	// and we want to avoid dealing with set methods keeping the key in memory
	// which would get overridden if we use a buffer from the pool.
	const digestSize = 32
	digestBuffer := bytes.NewBuffer(make([]byte, 0, digestSize))

	err = hashByteSlice(storageValue, digestBuffer)
	if err != nil {
		return fmt.Errorf("computing storage value hash digest: %w", err)
	}

	err = database.Put(digestBuffer.Bytes(), storageValue)
	if err != nil {
		return fmt.Errorf("setting subvalue: %w", err)
	}

	n.registerDeletedSubValue(deltas)
	n.StorageValueInlined = false
	n.StorageValue = digestBuffer.Bytes()
	return nil
}

// registerDeletedSubValue sets the storage value of the node in the map
// of deleted storage value hashes if the node has its storage value NOT inlined.
func (n *Node) registerDeletedSubValue(deltas DeltaSubValueRecorder) {
	if n.StorageValueInlined {
		return
	}

	subValueHashDigest := common.NewHash(n.StorageValue)
	// Note we treat storage value hashes the same as node hashes for pruning.
	deltas.RecordDeleted(subValueHashDigest)
}
