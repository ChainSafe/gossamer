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

// Attempts to get a Git commit hash
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

// Holds a text representation of the Gossamer version
var Version = func() string {
	if VersionMeta != "stable" {
		return getFullVersion()
	} else {
		return getStableVersion()
	}
}()

// Gets a verbose, long version string, e.g., 0.9.0-unstable-e41617ba
func getFullVersion() string {
	version := getStableVersion() + "-" + VersionMeta
	if len(GitCommit) >= 8 {
		version += "-" + GitCommit[:8]
	}
	return version
}

// Gets a short, stable version string, e.g., 0.9.0
func getStableVersion() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}
