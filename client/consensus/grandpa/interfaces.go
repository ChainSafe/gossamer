// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// Telemetry TODO issue #3474
type Telemetry interface{}

type Header[Hash constraints.Ordered, N constraints.Unsigned] interface {
	ParentHash() Hash
	Hash() Hash
	Number() N
}

type HeaderBackend[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]] interface {
	// Header Get block header. Returns None if block is not found.
	Header(Hash) (*H, error)
}
