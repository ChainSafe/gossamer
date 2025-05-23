// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package notification

import "github.com/ChainSafe/gossamer/internal/client/utils/pubsub"

type Payload struct{}

var _ pubsub.Registry[struct{}, func() (Payload, error), Payload] = &registry[Payload]{}
