package pvf

import (
	"os"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func createTestValidationCode(t *testing.T) parachaintypes.ValidationCode {
	// this wasm was achieved by building polkadot's adder test parachain
	runtimeFilePath := "./testdata/test_parachain_adder.wasm"
	validationCodeBytes, err := os.ReadFile(runtimeFilePath)
	require.NoError(t, err)

	return parachaintypes.ValidationCode(validationCodeBytes)

}

func TestValidationWorkerPool_newValidationWorker(t *testing.T) {
	t.Parallel()
	testValidationCode := createTestValidationCode(t)

	cases := map[string]struct {
		setupWorkerPool func(t *testing.T) *workerPool
		expectedWorkers []parachaintypes.ValidationCodeHash
	}{
		"add_one_invalid_worker": {
			setupWorkerPool: func(t *testing.T) *workerPool {
				pool := newValidationWorkerPool()
				_, err := pool.newValidationWorker(parachaintypes.ValidationCode{1, 2, 3, 4})
				require.Error(t, err)
				return pool
			},
			expectedWorkers: []parachaintypes.ValidationCodeHash{},
		},
		"add_one_valid_worker": {
			setupWorkerPool: func(t *testing.T) *workerPool {
				pool := newValidationWorkerPool()
				_, err := pool.newValidationWorker(testValidationCode)
				require.NoError(t, err)
				return pool
			},
			expectedWorkers: []parachaintypes.ValidationCodeHash{
				testValidationCode.Hash(),
			},
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			workerPool := tt.setupWorkerPool(t)
			defer workerPool.stop()

			require.ElementsMatch(t,
				maps.Keys(workerPool.workers),
				tt.expectedWorkers)
		})
	}
}
