package pvf

import (
	"testing"
)

func Test_validationHost_start(t *testing.T) {
	type fields struct {
		workerPool *workerPool
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
			v := &Host{
				workerPool: tt.fields.workerPool,
			}
			v.Start()
		})
	}
}
