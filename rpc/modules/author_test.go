package modules

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/state"
)

func TestAuthorModule_Pending(t *testing.T) {
	txQueue := state.NewTransactionQueue()
	auth := NewAuthorModule(nil, txQueue)

	res := new(PendingExtrinsicsResponse)
	auth.PendingExtrinsics(nil, nil, res)

	if !reflect.DeepEqual(*res, PendingExtrinsicsResponse([][]byte{})) {
		t.Errorf("Fail: expected: %+v got: %+v\n", res, &[][]byte{})
	}
}
