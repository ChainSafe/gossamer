package pvf

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestValidationWorkerPool_newValidationWorker(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		setupWorkerPool func(t *testing.T) *validationWorkerPool
		expectedWorkers []parachaintypes.ValidationCodeHash
	}{
		"add_one_worker": {
			setupWorkerPool: func(t *testing.T) *validationWorkerPool {
				pool := newValidationWorkerPool()
				pool.newValidationWorker(parachaintypes.ValidationCodeHash{1, 2, 3, 4})
				return pool
			},
			expectedWorkers: []parachaintypes.ValidationCodeHash{
				{1, 2, 3, 4},
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
