package modules

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Was made with @polkadot/api on https://github.com/danforbes/polkadot-js-scripts/tree/create-signed-tx
const validEncodedExtrinsic = "0xd1018400d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01bc2b6e35929aabd5b8bc4e5b0168c9bee59e2bb9d6098769f6683ecf73e44c776652d947a270d59f3d37eb9f9c8c17ec1b4cc473f2f9928ffdeef0f3abd43e85d502000000012844616e20466f72626573"

func TestPaymentQueryInfo(t *testing.T) {
	state := newTestStateService(t)
	mod := &PaymentModule{
		blockAPI: state.Block,
	}

	var req PaymentQueryInfoRequest
	req.Ext = validEncodedExtrinsic
	req.Hash = nil

	var res uint
	err := mod.QueryInfo(nil, &req, &res)
	require.NoError(t, err)
}
