package pvf

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"testing"
)

func Test_validationHost_start(t *testing.T) {
	type fields struct {
		workerPool *validationWorkerPool
	}
	tests := map[string]struct {
		name   string
		fields fields
	}{
		"test": {
			name: "test",
		},
	}
	for tname, tt := range tests {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			v := &ValidationHost{
				workerPool: tt.fields.workerPool,
			}
			v.Start()
		})
	}
}

func TestValidationHost(t *testing.T) {
	v := NewValidationHost()
	v.workerPool.newValidationWorker(parachaintypes.ValidationCodeHash{1, 2, 3, 4})
	v.Validate(parachaintypes.ValidationCodeHash{1, 2, 3, 4})
}
