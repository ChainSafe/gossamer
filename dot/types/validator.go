// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// ValidatorID represents a validator ID
type ValidatorID [sr25519.PublicKeyLength]byte

// ValidatorIndex represents a validator index
type ValidatorIndex uint32
