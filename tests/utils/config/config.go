// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

const (
	// AliceKey is the key for Alice.
	AliceKey = "alice"
)

// LogGrandpa generates a grandpa config.
func LogGrandpa() (config cfg.Config) {
	config = Default()
	config.Log = &cfg.LogConfig{
		Core:    "crit",
		Network: "debug",
		Runtime: "crit",
		Babe:    "info",
		Grandpa: "debug",
	}
	return config
}

// NoBabe generates a no-babe config.
func NoBabe() (config cfg.Config) {
	config = Default()
	config.LogLevel = "info"
	config.Log = &cfg.LogConfig{
		Sync:    "debug",
		Network: "debug",
	}
	config.Core.BabeAuthority = false
	return config
}

// NoGrandpa generates an no-grandpa config.
func NoGrandpa() (config cfg.Config) {
	config = Default()
	config.Core.GrandpaAuthority = false
	config.Core.GrandpaInterval = 1
	return config
}

// NotAuthority generates an non-authority config.
func NotAuthority() (config cfg.Config) {
	config = Default()
	config.Core.Role = 1
	config.Core.BabeAuthority = false
	config.Core.GrandpaAuthority = false
	return config
}
