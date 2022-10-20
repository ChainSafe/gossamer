// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import "fmt"

var (
	ErrSubscriptionTransport = fmt.Errorf("subscriptions are not available on this transport")
	ErrStartBlockHashEmpty   = fmt.Errorf("the start block hash cannot be an empty value")
)
