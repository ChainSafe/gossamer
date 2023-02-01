// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package gossamer

import "embed"

//go:embed chain/*/*.toml
// DefaultConfigTomlFiles resource files for default toml configurations
var DefaultConfigTomlFiles embed.FS
