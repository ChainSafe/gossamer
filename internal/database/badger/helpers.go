// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/internal/database"
	badger "github.com/dgraph-io/badger/v3"
)

func ptrTo[T any](value T) *T { return &value }

func newPrefixedKey(prefix, key []byte) (prefixedKey []byte) {
	// WARNING: Do not use:
	// return append(prefix, key...)
	// since the prefix might have a capacity larger than its length,
	// and that would produce data corruption on prefixed keys pointing
	// to the prefix underlying memory array.
	prefixedKey = make([]byte, 0, len(prefix)+len(key))
	prefixedKey = append(prefixedKey, prefix...)
	prefixedKey = append(prefixedKey, key...)
	return prefixedKey
}

// transformError transforms a badger error into a database error
// eventually, for errors defined in the parent database package.
func transformError(badgerErr error) (err error) {
	if errors.Is(badgerErr, badger.ErrDBClosed) {
		return fmt.Errorf("%w", database.ErrClosed)
	}
	return badgerErr
}
