// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package modules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

func TestSyncStateModule(t *testing.T) {
	fp := utils.GetWestendDevRawGenesisPath(t)

	data, err := os.ReadFile(filepath.Clean(fp))
	require.NoError(t, err)

	g := new(genesis.Genesis)
	err = json.Unmarshal(data, g)
	require.NoError(t, err)

	module := NewSyncStateModule(syncState{chainSpecification: g})

	req := GenSyncSpecRequest{
		Raw: true,
	}
	var res genesis.Genesis

	err = module.GenSyncSpec(nil, &req, &res)
	require.NoError(t, err)
}
