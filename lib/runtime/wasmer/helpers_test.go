// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
)

func genesisFromRawJSON(t *testing.T, jsonFilepath string) (gen genesis.Genesis) {
	t.Helper()

	fp, err := filepath.Abs(jsonFilepath)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Clean(fp))
	require.NoError(t, err)

	err = json.Unmarshal(data, &gen)
	require.NoError(t, err)

	return gen
}
