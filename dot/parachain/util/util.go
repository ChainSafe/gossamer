// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"context"
	"fmt"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
)

// SigningKeyAndIndex finds the first key we can sign with from the given set of validators,
// if any, and returns it along with the validator index.
func SigningKeyAndIndex(
	validators []parachaintypes.ValidatorID,
	ks keystore.Keystore,
) (*parachaintypes.ValidatorID, parachaintypes.ValidatorIndex) {
	for i, validator := range validators {
		publicKey, _ := sr25519.NewPublicKey(validator[:])
		keypair := ks.GetKeypair(publicKey)

		if keypair != nil {
			return &validator, parachaintypes.ValidatorIndex(i)
		}
	}
	return nil, 0
}
