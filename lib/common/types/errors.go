// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package types

import "errors"

// ErrInvalidResult is returned when decoding a Result type fails
var ErrInvalidResult = errors.New("decoding failed, invalid Result")
