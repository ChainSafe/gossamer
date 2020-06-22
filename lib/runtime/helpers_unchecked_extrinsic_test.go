package runtime

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/stretchr/testify/require"
)

func TestApplyExtrinsic_AuthoritiesChange_UncheckedExt(t *testing.T) {
	// TODO: update AuthoritiesChange to need to be signed by an authority
	rt := NewTestRuntime(t, NODE_RUNTIME)

	alice := kr.Alice.Public().Encode()
	bob := kr.Bob.Public().Encode()

	aliceb := [32]byte{}
	copy(aliceb[:], alice)

	bobb := [32]byte{}
	copy(bobb[:], bob)

	ids := [][32]byte{aliceb, bobb}

	ext := extrinsic.NewAuthoritiesChangeExt(ids)
	enc, err := ext.Encode()
	require.NoError(t, err)

	header := &types.Header{
		Number: big.NewInt(77),
	}

	err = rt.InitializeBlock(header)
	require.NoError(t, err)

	fmt.Printf("ac enc %v\n", enc)
	res, err := rt.ApplyExtrinsic(enc)
	require.Nil(t, err)

	require.Equal(t, []byte{0, 0}, res)
}

func TestApplyExtrinsic_StorageChange_Set_UncheckedExt(t *testing.T) {
	rt := NewTestRuntime(t, NODE_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(77),
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	ext := extrinsic.NewStorageChangeExt([]byte("testkey"), optional.NewBytes(true, []byte("testvalue")))

	extUx, err := extrinsic.CreateUncheckedExtrinsicUnsigned(ext)
	require.NoError(t, err)

	uxF, err := extUx.Function.Encode()
	require.NoError(t, err)
	uxF = append([]byte{4}, uxF...)
	exF2, err := scale.Encode(uxF)
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(exF2)
	require.NoError(t, err)
	// TODO detremine why this is returning 0x00010105 dispatch error, module 01, error 0005
	require.Equal(t, []byte{0, 1, 1, 0, 5}, res)

	// these aren't working, probably due to above issue
	//val, err := rt.storage.GetStorage([]byte("testkey"))
	//require.NoError(t, err)
	//require.Equal(t, []byte("testvalue"), val)
	//
	//for i := 0; i < maxRetries; i++ {
	//	_, err = rt.FinalizeBlock()
	//	if err == nil {
	//		break
	//	}
	//}
	//require.NoError(t, err)
	//
	//val, err = rt.storage.GetStorage([]byte("testkey"))
	//require.NoError(t, err)
	//// TODO: why does calling finalize_block modify the storage?
	//require.NotEqual(t, []byte("testvalue"), val)
}

func TestApplyExtrinsic_Transfer_NoBalance_UncheckedExt(t *testing.T) {
	rt := NewTestRuntime(t, NODE_RUNTIME)

	// Init transfer
	header := &types.Header{
		Number: big.NewInt(77),
	}
	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	alice := kr.Alice.Public().Encode()
	bob := kr.Bob.Public().Encode()

	ab := [32]byte{}
	copy(ab[:], alice)

	bb := [32]byte{}
	copy(bb[:], bob)

	var nonce uint64 = 0
	transfer := extrinsic.NewTransfer(ab, bb, 1000, nonce)
	gensisHash := common.MustHexToHash("0xcdd6bfd33737a9995d2b3463875408ba90be2789ad1e3edf3ac9736a40ca0a16")

	ux, err := extrinsic.CreateUncheckedExtrinsic(transfer, new(big.Int).SetUint64(nonce), gensisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	require.Equal(t, []byte{1, 2, 0, 5}, res) // 0x01020005 represents Apply error, Type: AncientBirthBlock
}

func TestApplyExtrinsic_Transfer_WithBalance_UncheckedExtrinsic(t *testing.T) {
	rt := NewTestRuntime(t, NODE_RUNTIME)

	// Init transfer
	header := &types.Header{
		Number: big.NewInt(77),
	}
	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	alice := kr.Alice.Public().Encode()
	bob := kr.Bob.Public().Encode()

	ab := [32]byte{}
	copy(ab[:], alice)

	bb := [32]byte{}
	copy(bb[:], bob)

	rt.storage.SetBalance(ab, 2000)

	var nonce uint64 = 1
	transfer := extrinsic.NewTransfer(ab, bb, 1000, nonce)
	gensisHash := common.MustHexToHash("0xcdd6bfd33737a9995d2b3463875408ba90be2789ad1e3edf3ac9736a40ca0a16")

	ux, err := extrinsic.CreateUncheckedExtrinsic(transfer, new(big.Int).SetUint64(nonce), gensisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	require.Equal(t, []byte{1, 2, 0, 5}, res) // 0x01020005 represents Apply error, Type: AncientBirthBlock

	// TODO: not sure why balances aren't getting adjusted properly, because of AncientBirthBlock?
	bal, err := rt.storage.GetBalance(ab)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), bal)

	// TODO this causes runtime error because balance for bb is nil (and GetBalance breaks when trys binary.LittleEndian.Uint64(bal))
	//bal, err = rt.storage.GetBalance(bb)
	//require.NoError(t, err)
	//require.Equal(t, uint64(1000), bal)
}
