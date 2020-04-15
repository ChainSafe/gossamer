// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// TestFixFlagOrder tests the FixFlagOrder method
func TestFixFlagOrder(t *testing.T) {
	testDir := utils.NewTestDir(t)
	testConfig := path.Join(testDir, "config.toml")

	defer utils.RemoveTestDir(t)

	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    bool // whether or not FixFlagOrder should succeed
	}{
		{
			"Test gossamer [subcommand] --config --force --verbosity",
			[]string{"config", "force", "verbosity"},
			[]interface{}{testConfig, true, "trace"},
			true,
		},
		{
			"Test gossamer [subcommand] --force --config --verbosity",
			[]string{"force", "config", "verbosity"},
			[]interface{}{true, testConfig, "trace"},
			true,
		},
    {
      "Test gossamer [subcommand] --force --config --verbosity",
      []string{"force", "config", "verbosity", "badflag"},
      []interface{}{true, testConfig, "trace", "badflag"},
      false,
    },
  }

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)

			fixedExportAction := FixFlagOrder(exportAction)

			err = fixedExportAction(ctx)
			require.Nil(t, err)
		})
	}
}
