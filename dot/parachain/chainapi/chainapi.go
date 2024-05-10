// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package chainapi

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

type ChainAPIMessage[message any] struct {
	Message         message
	ResponseChannel chan any
}

type BlockHeader common.Hash

func GetNumberOfValidators() uint {
	// TODO: implement this, currently it's just a stub that should be replaced, see issue #3932
	return 10
}
