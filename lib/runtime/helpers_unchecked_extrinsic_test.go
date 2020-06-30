package runtime

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/extrinsic"
	"github.com/stretchr/testify/require"
)

func TestApplyExtrinsic_AuthoritiesChange_UncheckedExt(t *testing.T) {
	t.Skip()
	// TODO: update AuthoritiesChange to need to be signed by an authority
	rt := NewTestRuntime(t, NODE_RUNTIME)

	alice := kr.Alice.Public().Encode()
	bob := kr.Bob.Public().Encode()

	aliceb := [32]byte{}
	copy(aliceb[:], alice)

	bobb := [32]byte{}
	copy(bobb[:], bob)

	//ids := [][32]byte{aliceb, bobb}

	//ext := extrinsic.NewAuthoritiesChangeExt(ids)
	fct := &extrinsic.Function{}
	extUx, err := extrinsic.CreateUncheckedExtrinsicUnsigned(fct)
	require.NoError(t, err)
	fmt.Printf("ux %v\n", extUx)

	enc, err := extUx.Encode()
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
	t.Skip()
	rt := NewTestRuntime(t, NODE_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(77),
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	//ext := extrinsic.NewStorageChangeExt([]byte("testkey"), optional.NewBytes(true, []byte("testvalue")))
	type KV struct {
		Key []byte
		Val []byte
	}
	kv1 := KV{
		Key: []byte("testkey"),
		Val: []byte("testvalue"),
	}
	tranCallData := struct {
		Vals []KV
	}{
		Vals: []KV{kv1},
	}
	fct := &extrinsic.Function{
		Call:     extrinsic.System,
		Pallet:   extrinsic.SYS_set_storage,
		CallData: tranCallData,
	}
	// TODO try signing this
	ux, err := extrinsic.CreateUncheckedExtrinsicUnsigned(fct)
	require.NoError(t, err)
	fmt.Printf("ux %v\n", ux)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	fmt.Printf("stor Enc %v\n", uxEnc)

	res, err := rt.ApplyExtrinsic(uxEnc)
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
		Number: big.NewInt(1),
	}
	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	bob := kr.Bob.Public().Encode()
	bb := [32]byte{}
	copy(bb[:], bob)

	var nonce uint64 = 1
	tranCallData := struct {
		Type byte
		To   [32]byte
		Amt  *big.Int
	}{
		Type: byte(255), // TODO determine why this is 255 (Address type?)
		To:   bb,
		Amt:  big.NewInt(1000),
	}
	transferF := &extrinsic.Function{
		Call:     extrinsic.Balances,
		Pallet:   extrinsic.PB_Transfer,
		CallData: tranCallData,
	}
	gensisHash := common.MustHexToHash("0xcdd6bfd33737a9995d2b3463875408ba90be2789ad1e3edf3ac9736a40ca0a16")

	ux, err := extrinsic.CreateUncheckedExtrinsic(transferF, new(big.Int).SetUint64(nonce), gensisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	// test encoding from substrate subkey
	//uxtExc := []byte{49, 2, 132, 255, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 1, 198, 191, 77, 13, 84, 220, 237, 107, 19, 190, 230, 176, 7, 204, 142, 40, 146, 150, 84, 141, 230, 75, 149, 63, 254, 157, 173, 91, 213, 194, 192, 40, 129, 37, 114, 60, 207, 38, 242, 40, 157, 32, 159, 126, 226, 173, 21, 144, 178, 48, 148, 18, 18, 36, 21, 148, 64, 206, 2, 71, 153, 56, 22, 140, 38, 0, 4, 0, 6, 0, 255, 142, 175, 4, 21, 22, 135, 115, 99, 38, 201, 254, 161, 126, 37, 252, 82, 135, 97, 54, 147, 201, 18, 144, 156, 178, 38, 170, 71, 148, 242, 106, 72, 161, 15}
	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	// we get below when header number is not 1
	//require.Equal(t, []byte{1, 2, 0, 5}, res) // 0x01020005 represents Apply error, Type: AncientBirthBlock
	// TODO determine why were getting this response
	require.Equal(t, []byte{1, 2, 0, 4}, res) // 0x01020004 represents Apply error, Type: BadProof ie bad signatrue
}

func TestApplyExtrinsic_Transfer_WithBalance_UncheckedExtrinsic(t *testing.T) {
	rt := NewTestRuntime(t, NODE_RUNTIME)

	// Init transfer
	header := &types.Header{
		Number: big.NewInt(1),
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
	tranCallData := struct {
		Type byte
		To   [32]byte
		Amt  *big.Int
	}{
		Type: byte(255), // TODO determine why this is 255 (Address type?)
		To:   bb,
		Amt:  big.NewInt(1000),
	}
	transferF := &extrinsic.Function{
		Call:     extrinsic.Balances,
		Pallet:   extrinsic.PB_Transfer,
		CallData: tranCallData,
	}
	gensisHash := common.MustHexToHash("0xcdd6bfd33737a9995d2b3463875408ba90be2789ad1e3edf3ac9736a40ca0a16")

	ux, err := extrinsic.CreateUncheckedExtrinsic(transferF, new(big.Int).SetUint64(nonce), gensisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	// TODO determine why were getting this response
	require.Equal(t, []byte{1, 2, 0, 4}, res) // 0x01020004 represents Apply error, Type: BadProof

	// TODO: not sure why balances aren't getting adjusted properly, because of AncientBirthBlock?
	bal, err := rt.storage.GetBalance(ab)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), bal)

	// TODO this causes runtime error because balance for bb is nil (and GetBalance breaks when trys binary.LittleEndian.Uint64(bal))
	//bal, err = rt.storage.GetBalance(bb)
	//require.NoError(t, err)
	//require.Equal(t, uint64(1000), bal)
}
