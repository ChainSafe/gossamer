// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"io"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/stretchr/testify/require"
)

// test buildIdentity method
func TestBuildIdentity(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

	configA := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		BasePath: testDir,
	}

	err := configA.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	configB := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		BasePath: testDir,
	}

	err = configB.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(configA.privateKey, configB.privateKey) {
		t.Error("Private keys should match")
	}

	configC := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		RandSeed: 1,
	}

	err = configC.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	configD := &Config{
		logger:   log.New(log.SetWriter(io.Discard)),
		RandSeed: 2,
	}

	err = configD.buildIdentity()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(configC.privateKey, configD.privateKey) {
		t.Error("Private keys should not match")
	}
}

// test build configuration method
func TestBuild(t *testing.T) {
	testBasePath := utils.NewTestBasePath(t, "node")
	defer utils.RemoveTestDir(t)

	testBlockState := &state.BlockState{}
	testRandSeed := int64(1)

	cfg := &Config{
		logger:     log.New(log.SetWriter(io.Discard)),
		BlockState: testBlockState,
		BasePath:   testBasePath,
		RandSeed:   testRandSeed,
	}

	err := cfg.build()
	if err != nil {
		t.Fatal(err)
	}

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
