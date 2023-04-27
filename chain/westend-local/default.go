// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package westendlocal

import (
	cfg "github.com/ChainSafe/gossamer/config"
)

const (
	// defaultName is the default node name
	defaultName = "Westend"
	// defaultID is the default node ID
	defaultID = "westend_local"
	// defaultChainSpec is the default chain spec for the westend local node
	defaultChainSpec = "./chain/westend-local/westend-local-spec-raw.json"

	// DefaultBasePathAlice is the default basepath for the westend local alice node
	DefaultBasePathAlice = "~/.gossamer/westend-local-alice"
	// DefaultBasePathBob is the default basepath for the westend local bob node
	DefaultBasePathBob = "~/.gossamer/westend-local-bob"
	// DefaultBasePathCharlie is the default basepath for the westend local charlie node
	DefaultBasePathCharlie = "~/.gossamer/westend-local-charlie"
)

// DefaultConfig returns a westend local node configuration
func DefaultConfig() *cfg.Config {
	config := cfg.DefaultConfig()
	config.Name = defaultName
	config.ID = defaultID
	config.ChainSpec = defaultChainSpec

	return config
}

// DefaultAliceConfig returns a westend local node configuration
func DefaultAliceConfig() *cfg.Config {
	config := DefaultConfig()
	config.BasePath = DefaultBasePathAlice
	config.PrometheusPort = uint32(9856)
	config.Network.Port = 7001
	config.RPC.Port = 8545
	config.RPC.WSPort = 8546
	config.Pprof.ListeningAddress = "localhost:6060"

	return config
}

// DefaultBobConfig returns a westend local node configuration with bob as the authority
func DefaultBobConfig() *cfg.Config {
	config := DefaultConfig()
	config.BasePath = DefaultBasePathBob
	config.PrometheusPort = uint32(9866)
	config.Network.Port = 7011
	config.RPC.Port = 8555
	config.RPC.WSPort = 8556
	config.Pprof.ListeningAddress = "localhost:6070"

	return config
}

// DefaultCharlieConfig returns a westend local node configuration with charlie as the authority
func DefaultCharlieConfig() *cfg.Config {
	config := DefaultConfig()
	config.BasePath = DefaultBasePathCharlie
	config.PrometheusPort = uint32(9876)
	config.Network.Port = 7021
	config.RPC.Port = 8565
	config.RPC.WSPort = 8566
	config.Pprof.ListeningAddress = "localhost:6080"

	return config
}
