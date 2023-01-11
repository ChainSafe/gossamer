// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import "github.com/ChainSafe/chaindb"

// blockStateDatabase is the database interface for the block state.
// Note this interface is not composed since that would require to
// move all sub-interfaces to this file and generate a lot more mocks.
type blockStateDatabase interface {
	Get(key []byte) (value []byte, err error)
	Put(key []byte, value []byte) error
	Del(key []byte) error
	Has(key []byte) (has bool, err error)
	NewBatch() chaindb.Batch
}
