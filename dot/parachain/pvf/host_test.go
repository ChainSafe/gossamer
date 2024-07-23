package pvf

import (
	"fmt"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
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
	v.Start()
	v.workerPool.newValidationWorker(parachaintypes.ValidationCodeHash{1, 2, 3, 4})

	resCh := make(chan *ValidationTaskResult)

	requestMsg := &ValidationTask{
		WorkerID: &parachaintypes.ValidationCodeHash{1, 2, 3, 4},
		ResultCh: resCh,
	}

	v.Validate(requestMsg)

	res := <-resCh
	fmt.Printf("Validation result: %v", res)
}
