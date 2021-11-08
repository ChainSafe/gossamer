// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import "errors"

// ErrSubscriptionTransport error sent when trying to access websocket subscriptions via http
var ErrSubscriptionTransport = errors.New("subscriptions are not available on this transport")
