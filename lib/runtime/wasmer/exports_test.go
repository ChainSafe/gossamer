package wasmer

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/trie"
	log "github.com/ChainSafe/log15"
	gtypes "github.com/centrifuge/go-substrate-rpc-client/v2/types"
	"github.com/stretchr/testify/require"
)

func TestInstance_Version_PolkadotRuntime(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		nil,
		5,
	)

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_KusamaRuntime(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/ksmcc/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0xb0006203c3a6e6bd2c6a17b1d4ae8ca49a31da0f4579da950b127774b44aef6b")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)

	expected := runtime.NewVersionData(
		[]byte("kusama"),
		[]byte("parity-kusama"),
		2,
		1020,
		0,
		nil,
		0,
	)

	// TODO: why does kusama seem to use the old runtime version format?
	version, err := instance.(*Instance).inst.Version()
	require.NoError(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func TestInstance_Version_NodeRuntime(t *testing.T) {
	expected := runtime.NewVersionData(
		[]byte("node"),
		[]byte("substrate-node"),
		10,
		260,
		0,
		nil,
		1,
	)

	instance := NewTestInstance(t, runtime.NODE_RUNTIME)

	version, err := instance.Version()
	require.Nil(t, err)

	t.Logf("SpecName: %s\n", version.SpecName())
	t.Logf("ImplName: %s\n", version.ImplName())
	t.Logf("AuthoringVersion: %d\n", version.AuthoringVersion())
	t.Logf("SpecVersion: %d\n", version.SpecVersion())
	t.Logf("ImplVersion: %d\n", version.ImplVersion())
	t.Logf("TransactionVersion: %d\n", version.TransactionVersion())

	require.Equal(t, 12, len(version.APIItems()))
	require.Equal(t, expected.SpecName(), version.SpecName())
	require.Equal(t, expected.ImplName(), version.ImplName())
	require.Equal(t, expected.AuthoringVersion(), version.AuthoringVersion())
	require.Equal(t, expected.SpecVersion(), version.SpecVersion())
	require.Equal(t, expected.ImplVersion(), version.ImplVersion())
	require.Equal(t, expected.TransactionVersion(), version.TransactionVersion())
}

func balanceKey(t *testing.T, pub []byte) []byte {
	h0, err := common.Twox128Hash([]byte("System"))
	require.NoError(t, err)
	h1, err := common.Twox128Hash([]byte("Account"))
	require.NoError(t, err)
	h2, err := common.Blake2b128(pub)
	require.NoError(t, err)
	return append(append(append(h0, h1...), h2...), pub...)
}

func TestNodeRuntime_ValidateTransaction(t *testing.T) {
	alicePub := common.MustHexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	aliceBalanceKey := balanceKey(t, alicePub)

	rt := NewTestInstance(t, runtime.NODE_RUNTIME)

	accInfo := types.AccountInfo{
		Nonce:    0,
		RefCount: 0,
		Data: struct {
			Free       common.Uint128
			Reserved   common.Uint128
			MiscFrozen common.Uint128
			FreeFrozen common.Uint128
		}{
			Free:       *common.Uint128FromBigInt(big.NewInt(1152921504606846976)),
			Reserved:   *common.Uint128FromBigInt(big.NewInt(0)),
			MiscFrozen: *common.Uint128FromBigInt(big.NewInt(0)),
			FreeFrozen: *common.Uint128FromBigInt(big.NewInt(0)),
		},
	}

	encBal, err := gtypes.EncodeToBytes(accInfo)
	require.NoError(t, err)

	err = rt.inst.ctx.Storage.Set(aliceBalanceKey, encBal)
	require.NoError(t, err)

	extBytes, err := common.HexToBytes("0x2d0284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d01ccacd0447dd220241dfb510e6e0554dff73899e79a068c58c7a149f568c71e046893a7e4726b5532af338b7780d0e9a83e9acc00e1610b02468405b2394769840000000600ff90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22e5c0")
	require.NoError(t, err)

	_ = buildBlock(t, rt)

	ext := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, extBytes...))

	_, err = rt.ValidateTransaction(ext)
	require.NoError(t, err)
}

func TestInstance_GrandpaAuthorities_NodeRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	tt.Put(runtime.GrandpaAuthoritiesKey, value)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []*types.Authority{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	require.Equal(t, expected, auths)
}

func TestInstance_GrandpaAuthorities_PolkadotRuntime(t *testing.T) {
	tt := trie.NewEmptyTrie()

	value, err := common.HexToBytes("0x0108eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	tt.Put(runtime.GrandpaAuthoritiesKey, value)

	rt := NewTestInstanceWithTrie(t, runtime.POLKADOT_RUNTIME, tt, log.LvlTrace)

	auths, err := rt.GrandpaAuthorities()
	require.NoError(t, err)

	authABytes, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authBBytes, _ := common.HexToBytes("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	authA, _ := ed25519.NewPublicKey(authABytes)
	authB, _ := ed25519.NewPublicKey(authBBytes)

	expected := []*types.Authority{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	require.Equal(t, expected, auths)
}

func TestInstance_BabeConfiguration_NodeRuntime_NoAuthorities(t *testing.T) {
	rt := NewTestInstance(t, runtime.NODE_RUNTIME)
	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: nil,
		Randomness:         [32]byte{},
		SecondarySlots:     1,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_BabeConfiguration_NodeRuntime_WithAuthorities(t *testing.T) {
	tt := trie.NewEmptyTrie()

	rvalue, err := common.HexToHash("0x01")
	require.NoError(t, err)
	tt.Put(runtime.BABERandomnessKey(), rvalue[:])

	avalue, err := common.HexToBytes("0x08eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640100000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	require.NoError(t, err)

	tt.Put(runtime.BABEAuthoritiesKey(), avalue)

	rt := NewTestInstanceWithTrie(t, runtime.NODE_RUNTIME, tt, log.LvlTrace)

	cfg, err := rt.BabeConfiguration()
	require.NoError(t, err)

	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	expectedAuthData := []*types.AuthorityRaw{
		{Key: authA, Weight: 1},
		{Key: authB, Weight: 1},
	}

	expected := &types.BabeConfiguration{
		SlotDuration:       3000,
		EpochLength:        200,
		C1:                 1,
		C2:                 4,
		GenesisAuthorities: expectedAuthData,
		Randomness:         [32]byte{1},
		SecondarySlots:     1,
	}

	require.Equal(t, expected, cfg)
}

func TestInstance_InitializeBlock_NodeRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.NODE_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: types.Digest{},
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func TestInstance_InitializeBlock_PolkadotRuntime(t *testing.T) {
	rt := NewTestInstance(t, runtime.POLKADOT_RUNTIME)

	header := &types.Header{
		Number: big.NewInt(1),
		Digest: types.Digest{},
	}

	err := rt.InitializeBlock(header)
	require.NoError(t, err)
}

func buildBlock(t *testing.T, instance runtime.Instance) *types.Block {
	header := &types.Header{
		ParentHash: trie.EmptyHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{},
	}

	err := instance.InitializeBlock(header)
	require.NoError(t, err)

	idata := types.NewInherentsData()
	err = idata.SetInt64Inherent(types.Timstap0, uint64(time.Now().Unix()))
	require.NoError(t, err)

	err = idata.SetInt64Inherent(types.Babeslot, 1)
	require.NoError(t, err)

	err = idata.SetBigIntInherent(types.Finalnum, big.NewInt(0))
	require.NoError(t, err)

	ienc, err := idata.Encode()
	require.NoError(t, err)

	// Call BlockBuilder_inherent_extrinsics which returns the inherents as extrinsics
	inherentExts, err := instance.InherentExtrinsics(ienc)
	require.NoError(t, err)

	// decode inherent extrinsics
	exts, err := scale.Decode(inherentExts, [][]byte{})
	require.NoError(t, err)

	// apply each inherent extrinsic
	for _, ext := range exts.([][]byte) {
		in, err := scale.Encode(ext) //nolint
		require.NoError(t, err)

		ret, err := instance.ApplyExtrinsic(append([]byte{1}, in...))
		require.NoError(t, err, in)
		require.Equal(t, ret, []byte{0, 0})
	}

	res, err := instance.FinalizeBlock()
	require.NoError(t, err)

	res.Number = header.Number

	babeDigest := types.NewBabePrimaryPreDigest(0, 1, [32]byte{}, [64]byte{})
	data := babeDigest.Encode()
	preDigest := types.NewBABEPreRuntimeDigest(data)
	res.Digest = types.Digest{preDigest}

	expected := &types.Header{
		ParentHash: header.ParentHash,
		Number:     big.NewInt(1),
		Digest:     types.Digest{preDigest},
	}

	require.Equal(t, expected.ParentHash, res.ParentHash)
	require.Equal(t, expected.Number, res.Number)
	require.Equal(t, expected.Digest, res.Digest)
	require.NotEqual(t, common.Hash{}, res.StateRoot)
	require.NotEqual(t, common.Hash{}, res.ExtrinsicsRoot)
	require.NotEqual(t, trie.EmptyHash, res.StateRoot)

	return &types.Block{
		Header: res,
		Body:   types.NewBody(inherentExts),
	}
}

func TestInstance_FinalizeBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	buildBlock(t, instance)
}

func TestInstance_ExecuteBlock_NodeRuntime(t *testing.T) {
	instance := NewTestInstance(t, runtime.NODE_RUNTIME)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_GossamerRuntime(t *testing.T) {
	t.Skip() // TODO: fix timestamping issue
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime(t *testing.T) {
	DefaultTestLogLvl = 0

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime_PolkadotBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/polkadot/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0x29d0d972cd27cbc511e9589fcb7a4506d5eb6a9e8df205f00472e5ab354a4e17")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 5

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 3, 0, 11, 80, 149, 160, 81, 114, 1, 16, 4, 20, 0, 0}
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	// digest data received from querying polkadot node
	digestBytes := common.MustHexToBytes("0x0c0642414245b501010000000093decc0f00000000362ed8d6055645487fe42e9c8640be651f70a3a2a03658046b2b43f021665704501af9b1ca6e974c257e3d26609b5f68b5b0a1da53f7f252bbe5d94948c39705c98ffa4b869dd44ac29528e3723d619cc7edf1d3f7b7a57a957f6a7e9bdb270a044241424549040118fa3437b10f6e7af8f31362df3a179b991a8c56313d1bcd6307a4d0c734c1ae310100000000000000d2419bc8835493ac89eb09d5985281f5dff4bc6c7a7ea988fd23af05f301580a0100000000000000ccb6bef60defc30724545d57440394ed1c71ea7ee6d880ed0e79871a05b5e40601000000000000005e67b64cf07d4d258a47df63835121423551712844f5b67de68e36bb9a21e12701000000000000006236877b05370265640c133fec07e64d7ca823db1dc56f2d3584b3d7c0f1615801000000000000006c52d02d95c30aa567fda284acf25025ca7470f0b0c516ddf94475a1807c4d250100000000000000000000000000000000000000000000000000000000000000000000000000000005424142450101d468680c844b19194d4dfbdc6697a35bf2b494bda2c5a6961d4d4eacfbf74574379ba0d97b5bb650c2e8670a63791a727943bcb699dc7a228bdb9e0a98c9d089")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)

	// polkadot block 1, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x91b171bb158e2d3848fa23a9f1c25182fb8e20313b2c1eb49219da7a70ce90c3"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xc56fcd6e7a757926ace3e1ecff9b4010fc78b90d459202a339266a7f6360002f"),
			ExtrinsicsRoot: common.MustHexToHash("0x9a87f6af64ef97aff2d31bebfdd59f8fe2ef6019278b634b2515a38f1c4c2420"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, _ = instance.ExecuteBlock(block) // TODO: complete
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/ksmcc/genesis-raw.json")
	require.NoError(t, err)

	genTrie, err := genesis.NewTrieFromGenesis(gen)
	require.NoError(t, err)

	expectedGenesisRoot := common.MustHexToHash("0xb0006203c3a6e6bd2c6a17b1d4ae8ca49a31da0f4579da950b127774b44aef6b")
	require.Equal(t, expectedGenesisRoot, genTrie.MustHash())

	// set state to genesis state
	genState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = genState
	cfg.LogLvl = 4

	instance, err := NewRuntimeFromGenesis(gen, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := []byte{8, 40, 4, 2, 0, 11, 144, 17, 14, 179, 110, 1, 16, 4, 20, 0, 0}
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x0c0642414245340201000000ef55a50f00000000044241424549040118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000054241424501014625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)

	// kusama block 1, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0xb0a8d493285c2df73290dfb7e61f870f17b41801197a149ca93654499ea3dafe"),
			Number:         big.NewInt(1),
			StateRoot:      common.MustHexToHash("0xfabb0c6e92d29e8bb2167f3c6fb0ddeb956a4278a3cf853661af74a076fc9cb7"),
			ExtrinsicsRoot: common.MustHexToHash("0xa35fb7f7616f5c979d48222b3d2fa7cb2331ef73954726714d91ca945cc34fd8"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock3784(t *testing.T) {
	gossTrie3783 := newTrieFromPairs(t, "../../../block3783_gossamer.out")
	expectedRoot := common.MustHexToHash("0x948338bc0976aee78879d559a1f42385407e5a481b05a91d2a9386aa7507e7a0")
	require.Equal(t, expectedRoot, gossTrie3783.MustHash())

	// set state to genesis state
	state3783, err := storage.NewTrieState(gossTrie3783)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state3783
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(gossTrie3783, cfg)
	require.NoError(t, err)

	// block data is received from querying a polkadot node
	body := common.MustHexToBytes("0x10280402000bb00d69b46e0114040900193b10041400009101041300eaaec5728cd6ea9160ff92a49bb45972c532d2163241746134726aaa5b2f72129d8650715320f23765c6306503669f69bf684b188dea73b1e247dd1dd166513b1c13daa387c35f24ac918d2fa772b73cffd20204a8875e48a1b11bb3229deb7f00")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 4, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245340203000000bd64a50f0000000005424142450101bc0d6850dba8d32ea1dbe26cb4ac56da6cca662c7cc642dc8eed32d2bddd65029f0721436eafeebdf9b4f17d1673c6bc6c3c51fe3dda3121a5fc60c657a5808b")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)

	// kusama block 3784, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x4843b4aa38cf2e3e2f6fae401b98dd705bed668a82dd3751dc38f1601c814ca8"),
			Number:         big.NewInt(3784),
			StateRoot:      common.MustHexToHash("0xac44cc18ec22f0f3fca39dfe8725c0383af1c982a833e081fbb2540e46eb09a5"),
			ExtrinsicsRoot: common.MustHexToHash("0x52b7d4852fc648cb8f908901e1e36269593c25050c31718454bca74b69115d12"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func newTrieFromPairs(t *testing.T, filename string) *trie.Trie {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	rpcPairs := make(map[string]interface{})
	err = json.Unmarshal(data, &rpcPairs)
	require.NoError(t, err)
	pairs := rpcPairs["result"].([]interface{})

	entries := make(map[string]string)
	for _, pair := range pairs {
		pairArr := pair.([]interface{})
		entries[pairArr[0].(string)] = pairArr[1].(string)
	}

	tr := trie.NewEmptyTrie()
	err = tr.LoadFromMap(entries)
	require.NoError(t, err)
	return tr
}
