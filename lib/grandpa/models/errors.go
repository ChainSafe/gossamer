// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package models

import "errors"

var (
	ErrBlockDoesNotExist = errors.New("block does not exist")
	ErrVoterNotFound     = errors.New("voter is not in voter set")
)
