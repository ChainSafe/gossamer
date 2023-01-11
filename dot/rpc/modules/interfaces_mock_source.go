// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import "encoding/json"

type telemetryClient interface {
	SendMessage(msg json.Marshaler)
}
