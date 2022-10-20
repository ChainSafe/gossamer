// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import "fmt"

// ErrInvalidResult is returned when decoding a Result type fails
var ErrInvalidResult = fmt.Errorf("decoding failed, invalid Result")
