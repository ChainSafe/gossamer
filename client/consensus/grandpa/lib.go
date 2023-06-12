// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// AuthorityList A list of Grandpa authorities with associated weights.
// TODO migrate this type and associated functions to this package
type AuthorityList []types.Authority
