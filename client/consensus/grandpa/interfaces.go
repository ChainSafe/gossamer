// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"golang.org/x/exp/constraints"
)

// Telemetry TODO issue #3474
type Telemetry interface{}

//type HashI interface {
//	constraints.Ordered
//	IsEmpty() bool
//}

type Header[H constraints.Ordered, N constraints.Unsigned] interface {
	ParentHash() H
	Hash() H
	Number() N
}

type HeaderBackend[Hash constraints.Ordered, N constraints.Unsigned, H Header[Hash, N]] interface {
	// Header Get block header. Returns None if block is not found.
	Header(H) (*H, error)
}
