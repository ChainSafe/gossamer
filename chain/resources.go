// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package chain

import "embed"

// DefaultConfigTomlFiles is the embedded file system containing the default toml configurations.
//
//go:embed */*.toml
var DefaultConfigTomlFiles embed.FS
