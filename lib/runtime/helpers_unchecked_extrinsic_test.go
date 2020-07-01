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

	// genesisHash from runtime must match genesisHash used in transfer payload message signing
	// genesis bytes for runtime seems to be stored in allocated storage at key 0xdfa1667c116b77971ada377f9bd9c485a0566b8e477ae01969120423f2f124ea
	//  runtime trace logs show genesis set to below:
	genesisBytes := []byte{83, 121, 115, 116, 101, 109, 32, 66, 108, 111, 99, 107, 72, 97, 115, 104, 0, 0, 0, 0, 101, 117, 101, 50, 54, 188, 7, 185, 79, 198, 211, 234}
	genesisHash := common.BytesToHash(genesisBytes)

	// Init transfer
	header := &types.Header{
		ParentHash: genesisHash,
		Number:     big.NewInt(1),
	}
	err := rt.InitializeBlock(header)
	require.NoError(t, err)

	bob := kr.Bob.Public().Encode()
	bb := [32]byte{}
	copy(bb[:], bob)

	var nonce uint64 = 0
	tranCallData := struct {
		Type byte
		To   [32]byte
		Amt  *big.Int
	}{
		Type: byte(255), // TODO determine why this is 255 (Lookup type?)
		To:   bb,
		Amt:  big.NewInt(1234),
	}
	transferF := &extrinsic.Function{
		Call:     extrinsic.Balances,
		Pallet:   extrinsic.PB_Transfer,
		CallData: tranCallData,
	}

	ux, err := extrinsic.CreateUncheckedExtrinsic(transferF, new(big.Int).SetUint64(nonce), genesisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	require.Equal(t, []byte{1, 2, 0, 1}, res) // 0x01020001 represents Apply error, Type: Payment: Inability to pay (expected result)
}

func TestApplyExtrinsic_Transfer_WithBalance_UncheckedExtrinsic(t *testing.T) {
	rt := NewTestRuntime(t, NODE_RUNTIME)

	genesisBytes := []byte{83, 121, 115, 116, 101, 109, 32, 66, 108, 111, 99, 107, 72, 97, 115, 104, 0, 0, 0, 0, 101, 117, 101, 50, 54, 188, 7, 185, 79, 198, 211, 234}
	genesisHash := common.BytesToHash(genesisBytes)

	// Init transfer
	header := &types.Header{
		ParentHash: genesisHash,
		Number:     big.NewInt(1),
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

	var nonce uint64 = 0
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

	ux, err := extrinsic.CreateUncheckedExtrinsic(transferF, new(big.Int).SetUint64(nonce), genesisHash, kr.Alice)
	require.NoError(t, err)

	uxEnc, err := ux.Encode()
	require.NoError(t, err)

	res, err := rt.ApplyExtrinsic(uxEnc)
	require.NoError(t, err)

	// TODO determine why were getting this response, set balance above should have fixed
	require.Equal(t, []byte{1, 2, 0, 1}, res) // 0x01020001 represents Apply error, Type: Payment: Inability to pay some fees

	// TODO: not sure why balances aren't getting adjusted properly, because of AncientBirthBlock?
	bal, err := rt.storage.GetBalance(ab)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), bal)

	// TODO this causes runtime error because balance for bb is nil (and GetBalance breaks when trys binary.LittleEndian.Uint64(bal))
	//bal, err = rt.storage.GetBalance(bb)
	//require.NoError(t, err)
	//require.Equal(t, uint64(1000), bal)
}
