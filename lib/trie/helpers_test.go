// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"errors"
)

type writeCall struct {
	written []byte
	n       int
	err     error
}

var errTest = errors.New("test error")
