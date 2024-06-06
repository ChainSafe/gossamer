// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package db

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Functionality for reading and storing children hashes from db.

// Returns the hashes of the children blocks of the block with `parentHash`.
func readChildren[H comparable](
	db database.Database[hash.H256], column database.ColumnID, prefix []byte, parentHash H,
) ([]H, error) {
	// buf := prefix
	encoded := scale.MustMarshal(parentHash)
	key := append(prefix, encoded...)

	rawValOpt := db.Get(column, key)

	if rawValOpt == nil {
		return nil, nil
	}
	rawVal := rawValOpt

	var children []H
	err := scale.Unmarshal(rawVal, &children)
	if err != nil {
		return nil, fmt.Errorf("Error decoding children: %w", err)
	}

	return children, nil
}

// Insert the key-value pair (`parentHash`, `childrenHashes`) in the transaction.
// Any existing value is overwritten upon write.
func writeChildren[H comparable](
	tx *database.Transaction[hash.H256], column database.ColumnID, prefix []byte, parentHash H, childrenHashes []H,
) {
	encoded := scale.MustMarshal(parentHash)
	key := append(prefix, encoded...)
	tx.Set(column, key, scale.MustMarshal(childrenHashes))
}

// Prepare transaction to remove the children of `parent_hash`.
func removeChildren[H comparable](
	tx *database.Transaction[hash.H256], column database.ColumnID, prefix []byte, parentHash H,
) {
	encoded := scale.MustMarshal(parentHash)
	key := append(prefix, encoded...)
	tx.Remove(column, key)
}
