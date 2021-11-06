// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package modules

import "errors"

// ErrSubscriptionTransport error sent when trying to access websocket subscriptions via http
var ErrSubscriptionTransport = errors.New("subscriptions are not available on this transport")
