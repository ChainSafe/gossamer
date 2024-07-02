// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package columns

import "github.com/ChainSafe/gossamer/internal/primitives/database"

const (
	Meta      database.ColumnID = 0
	State     database.ColumnID = 1
	StateMeta database.ColumnID = 2
	// maps hashes to lookup keys and numbers to canon hashes.
	KeyLookup      database.ColumnID = 3
	Header         database.ColumnID = 4
	Body           database.ColumnID = 5
	Justifications database.ColumnID = 6
	Aux            database.ColumnID = 8
	// Offchain workers local storage
	Offchain database.ColumnID = 9
	// Transactions
	Transaction database.ColumnID = 11
	BodyIndex   database.ColumnID = 12
)
