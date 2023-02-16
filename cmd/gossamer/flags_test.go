// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"io"
	"testing"

	"github.com/ChainSafe/gossamer/dot"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestFixFlagOrder tests the FixFlagOrder method
func TestFixFlagOrder(t *testing.T) {
	testCfg, testConfig := newTestConfigWithFile(t)
	genFile := dot.NewTestGenesisRawFile(t, testCfg)

	testApp := cli.NewApp()
	testApp.Writer = io.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
	}{
		{
			"Test gossamer --config --genesis --log --force --pruning --retain-blocks",
			[]string{"config", "genesis", "log", "force", "pruning", "retain-blocks"},
			[]interface{}{testConfig, genFile, "trace", true, "archive", uint32(512)},
		},
		{
			"Test gossamer --config --genesis --force --log --pruning --retain-blocks",
			[]string{"config", "genesis", "force", "log", "pruning", "retain-blocks"},
			[]interface{}{testConfig, genFile, true, "trace", "archive", uint32(512)},
		},
		{
			"Test gossamer --config --force --genesis --log ---pruning --retain-blocks",
			[]string{"config", "force", "genesis", "log", "pruning", "retain-blocks"},
			[]interface{}{testConfig, true, genFile, "trace", "archive", uint32(512)},
		},
		{
			"Test gossamer --force --config --genesis --log --pruning --retain-blocks",
			[]string{"force", "config", "genesis", "log", "pruning", "retain-blocks"},
			[]interface{}{true, testConfig, genFile, "trace", "archive", uint32(512)},
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.NoError(t, err)

			updatedInitAction := FixFlagOrder(initAction)
			err = updatedInitAction(ctx)
			require.NoError(t, err)

			updatedExportAction := FixFlagOrder(exportAction)
			err = updatedExportAction(ctx)
			require.NoError(t, err)
		})
	}
}
