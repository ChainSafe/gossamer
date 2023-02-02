// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only
package internal

import "embed"

//go:embed chain/*/*.toml
// DefaultConfigTomlFiles is the embedded file system containing the default toml configurations.
var DefaultConfigTomlFiles embed.FS
