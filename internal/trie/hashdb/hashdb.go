// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package hashdb

import "github.com/ChainSafe/gossamer/lib/common"

type Prefix struct {
	Data   []byte
	Padded *byte
}

type HashDB interface {
	Get(key []byte) (value []byte, err error)
	Insert(prefix Prefix, value []byte) common.Hash
}
