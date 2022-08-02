// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import "errors"

var (
	ErrSubscriptionTransport = errors.New("subscriptions are not available on this transport")
	ErrStartBlockHashEmpty   = errors.New("the start block hash cannot be an empty value")
)
