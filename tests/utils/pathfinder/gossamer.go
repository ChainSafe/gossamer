package pathfinder

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// GetGossamer returns the path to the Gossamer binary
// as <project root path>/bin/gossamer.
func GetGossamer(t *testing.T) (binPath string) {
	t.Helper()

	projectRootPath, err := utils.GetProjectRootPath()
	require.NoError(t, err, "cannot get project root path")
	return filepath.Join(projectRootPath, "bin/gossamer")
}
