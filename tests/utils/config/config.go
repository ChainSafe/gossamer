// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"github.com/ChainSafe/gossamer/dot/config/toml"
)

// LogGrandpa generates a grandpa config.
func LogGrandpa() (cfg toml.Config) {
	cfg = Default()
	cfg.Log = toml.LogConfig{
		CoreLvl:           "crit",
		NetworkLvl:        "debug",
		RuntimeLvl:        "crit",
		BlockProducerLvl:  "info",
		FinalityGadgetLvl: "debug",
	}
	return cfg
}

// NoBabe generates a no-babe config.
func NoBabe() (cfg toml.Config) {
	cfg = Default()
	cfg.Global.LogLvl = "info"
	cfg.Log = toml.LogConfig{
		SyncLvl:    "debug",
		NetworkLvl: "debug",
	}
	cfg.Core.BabeAuthority = false
	return cfg
}

// NoGrandpa generates an no-grandpa config.
func NoGrandpa() (cfg toml.Config) {
	cfg = Default()
	cfg.Core.GrandpaAuthority = false
	cfg.Core.BABELead = true
	cfg.Core.GrandpaInterval = 1
	return cfg
}

// NotAuthority generates an non-authority config.
func NotAuthority() (cfg toml.Config) {
	cfg = Default()
	cfg.Core.Roles = 1
	cfg.Core.BabeAuthority = false
	cfg.Core.GrandpaAuthority = false
	return cfg
}
