package newWasmer

import (
	"encoding/json"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
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
