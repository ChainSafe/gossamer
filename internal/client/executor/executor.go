// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package executor

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core"
	"github.com/ChainSafe/gossamer/internal/primitives/externalities"
	"github.com/ChainSafe/gossamer/internal/primitives/version"
)

type RuntimeVersionOf interface {
	RuntimeVersion(
		externalities externalities.Externalities,
		runtimeCode core.RuntimeCode,
	) (version.RuntimeVersion, error)
}
