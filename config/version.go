// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"runtime/debug"
)

// Sets the numeric Gossamer version here
const (
	VersionMajor = 0
	VersionMinor = 9
	VersionPatch = 0
	VersionMeta  = "unstable"
)

// GitCommit attempts to get a Git commit hash; empty string otherwise
var GitCommit = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}()

// Version holds a text representation of the Gossamer version
var Version = func() string {
	if VersionMeta != "stable" {
		return GetFullVersion()
	} else {
		return GetStableVersion()
	}
}()

// GetFullVersion gets a verbose, long version string, e.g., 0.9.0-unstable-e41617ba
func GetFullVersion() string {
	version := GetStableVersion() + "-" + VersionMeta
	if len(GitCommit) >= 8 {
		version += "-" + GitCommit[:8]
	}
	return version
}

// GetStableVersion gets a short, stable version string, e.g., 0.9.0
func GetStableVersion() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}
