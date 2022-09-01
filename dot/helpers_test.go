package dot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
)

func writeGenesisToTestJSON(t *testing.T, genesis genesis.Genesis) (filename string) {
	jsonData, err := json.Marshal(genesis)
	require.NoError(t, err)
	filename = filepath.Join(t.TempDir(), "genesis-test")
	err = os.WriteFile(filename, jsonData, os.ModePerm)
	require.NoError(t, err)
	return filename
}
