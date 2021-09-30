package modules

import (
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/sync"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v3/signature"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/stretchr/testify/require"
)

func TestPaymentQueryInfo(t *testing.T) {
	state := newTestStateService(t)
	mod := &PaymentModule{
		blockAPI: state.Block,
	}

	rt, err := state.Block.GetRuntime(nil)
	require.NoError(t, err)

	b, err := state.Block.BestBlock()
	require.NoError(t, err)

	createTestingBlock(t, state.Block, rt, state.Block.GenesisHash(), b.Header)

	b, err = state.Block.BestBlock()
	require.NoError(t, err)

	ext, err := b.Body.AsEncodedExtrinsics()
	require.NoError(t, err)

	fmt.Println(ext)

	var req PaymentQueryInfoRequest
	req.Ext = common.BytesToHex(ext[0])
	req.Hash = b.Header.Hash()

	var res uint
	err = mod.QueryInfo(nil, &req, &res)
	require.NoError(t, err)
}

func createTestingBlock(t *testing.T, bs *state.BlockState, rt runtime.Instance, genhash common.Hash, parent types.Header) {
	t.Helper()

	ext := createExtrinsic(t, rt, genhash, uint64(1))
	b := sync.BuildBlock(t, rt, &parent, ext)
	bs.StoreRuntime(b.Header.Hash(), rt)
	err := bs.AddBlock(b)
	require.NoError(t, err)

}

func createExtrinsic(t *testing.T, rt runtime.Instance, genHash common.Hash, nonce uint64) types.Extrinsic {
	t.Helper()
	rawMeta, err := rt.Metadata()
	require.NoError(t, err)

	var decoded []byte
	err = scale.Unmarshal(rawMeta, &decoded)
	require.NoError(t, err)

	meta := &ctypes.Metadata{}
	err = ctypes.DecodeFromBytes(decoded, meta)
	require.NoError(t, err)

	rv, err := rt.Version()
	require.NoError(t, err)

	c, err := ctypes.NewCall(meta, "System.remark", []byte{0xab, 0xcd})
	require.NoError(t, err)

	ext := ctypes.NewExtrinsic(c)
	o := ctypes.SignatureOptions{
		BlockHash:          ctypes.Hash(genHash),
		Era:                ctypes.ExtrinsicEra{IsImmortalEra: false},
		GenesisHash:        ctypes.Hash(genHash),
		Nonce:              ctypes.NewUCompactFromUInt(nonce),
		SpecVersion:        ctypes.U32(rv.SpecVersion()),
		Tip:                ctypes.NewUCompactFromUInt(0),
		TransactionVersion: ctypes.U32(rv.TransactionVersion()),
	}

	// Sign the transaction using Alice's key
	err = ext.Sign(signature.TestKeyringPairAlice, o)
	require.NoError(t, err)

	extEnc, err := ctypes.EncodeToHexString(ext)
	require.NoError(t, err)

	extBytes := types.Extrinsic(common.MustHexToBytes(extEnc))
	return extBytes
}
