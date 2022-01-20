// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
)

func TestBuildIdentity(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()

	configA := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		BasePath: testDir,
	}

	err := configA.buildIdentity()
	require.NoError(t, err)

	configB := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		BasePath: testDir,
	}

	err = configB.buildIdentity()
	require.NoError(t, err)

	require.Equal(t, configA.privateKey, configB.privateKey)

	configC := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		RandSeed: 1,
	}

	err = configC.buildIdentity()
	require.NoError(t, err)

	configD := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		RandSeed: 2,
	}

	err = configD.buildIdentity()
	require.NoError(t, err)

	require.NotEqual(t, configC.privateKey, configD.privateKey)
}

// test build configuration method
func TestBuild(t *testing.T) {
	t.Parallel()

	testBasePath := t.TempDir()

	testBlockState := &state.BlockState{}
	testRandSeed := int64(1)

	cfg := &Config{
		logger:     log.New(log.SetWriter(io.Discard)),
		BlockState: testBlockState,
		BasePath:   testBasePath,
		RandSeed:   testRandSeed,
	}

	err := cfg.build()
	require.NoError(t, err)

	require.Equal(t, testBlockState, cfg.BlockState)
	require.Equal(t, testBasePath, cfg.BasePath)
	require.Equal(t, DefaultRoles, cfg.Roles)
	require.Equal(t, DefaultPort, cfg.Port)
	require.Equal(t, testRandSeed, cfg.RandSeed)
	require.Equal(t, DefaultBootnodes, cfg.Bootnodes)
	require.Equal(t, DefaultProtocolID, cfg.ProtocolID)
	require.Equal(t, false, cfg.NoBootstrap)
	require.Equal(t, false, cfg.NoMDNS)
}
