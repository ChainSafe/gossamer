package runtime

import (
	"path/filepath"
	"testing"

	"github.com/ChainSafe/gossamer/keystore"
	"github.com/ChainSafe/gossamer/tests"
	"github.com/stretchr/testify/require"
)

// NewTestRuntime will create a new runtime (polkadot/test)
func NewTestRuntime(t *testing.T, targetRuntime string) *Runtime {
	testRuntimeFilePath, testRuntimeURL := tests.GetAbsolutePath(tests.TESTS_FP), tests.TEST_WASM_URL

	// If target runtime is polkadot, re-assign vars
	if targetRuntime == tests.POLKADOT_RUNTIME {
		testRuntimeFilePath, testRuntimeURL = tests.GetAbsolutePath(tests.POLKADOT_RUNTIME_FP), tests.POLKADOT_RUNTIME_URL
	}

	_, err := tests.GetRuntimeBlob(testRuntimeFilePath, testRuntimeURL)
	require.Nil(t, err, "Fail: could not get runtime", "targetRuntime", targetRuntime)

	rs := tests.NewTestRuntimeStorage(nil)

	fp, err := filepath.Abs(testRuntimeFilePath)
	require.Nil(t, err, "could not create testRuntimeFilePath", "targetRuntime", targetRuntime)

	r, err := NewRuntimeFromFile(fp, rs, keystore.NewKeystore())
	require.Nil(t, err, "Got error when trying to create new VM", "targetRuntime", targetRuntime)
	require.NotNil(t, r, "Could not create new VM instance", "targetRuntime", targetRuntime)

	return r
}
