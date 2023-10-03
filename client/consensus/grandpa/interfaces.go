// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/client/api"
	"golang.org/x/exp/constraints"
)

// Telemetry TODO issue #3474
type Telemetry interface{}

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
	Insert(insert []api.KeyValue, deleted []api.Key) error
	// Get Query auxiliary data from key-Value store.
	Get(key api.Key) (*[]byte, error)
}

type HashI interface {
	constraints.Ordered
	IsEmpty() bool
}

type HeaderI[H HashI, N constraints.Unsigned] interface {
	ParentHash() H
	Hash() H
	Number() N
}

type HeaderBackend[H HashI, N constraints.Unsigned, Header HeaderI[H, N]] interface {
	// Header Get block header. Returns None if block is not found.
	Header(H) (*Header, error)
}
