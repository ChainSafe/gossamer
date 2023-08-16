// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// Authority represents a grandpa authority
type Authority struct {
	Key    PublicKey
	Weight uint64
}
