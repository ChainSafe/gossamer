// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package api

type Key []byte

type KeyValue struct {
	Key   Key
	Value []byte
}

// AuxStore is part of the substrate backend.
// Provides access to an auxiliary database.
//
// This is a simple global database not aware of forks. Can be used for storing auxiliary
// information like total block weight/difficulty for fork resolution purposes as a common use
// case.
type AuxStore interface {
	// Insert auxiliary data into key-Value store.
	//
	// Deletions occur after insertions.
	Insert(insert []KeyValue, deleted []Key) error
	// Get Query auxiliary data from key-Value store.
	Get(key Key) (*[]byte, error)
}
