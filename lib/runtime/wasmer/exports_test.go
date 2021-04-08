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
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/kusama/genesis.json")
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
	version, err := instance.(*Instance).Version()
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

func balanceKey(t *testing.T, pub []byte) []byte { //nolint
	h0, err := common.Twox128Hash([]byte("System"))
	require.NoError(t, err)
	h1, err := common.Twox128Hash([]byte("Account"))
	require.NoError(t, err)
	h2, err := common.Blake2b128(pub)
	require.NoError(t, err)
	return append(append(append(h0, h1...), h2...), pub...)
}

func TestNodeRuntime_ValidateTransaction(t *testing.T) {
	t.Skip("fixing next_key breaks this... :(")
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

	rt.ctx.Storage.Set(aliceBalanceKey, encBal)

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
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
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

func TestInstance_ApplyExtrinsic_GossamerRuntime(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/gssmr/genesis.json")
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

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(genTrie)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	//initialize block header
	parentHash := common.MustHexToHash("0x35a28a7dbaf0ba07d1485b0f3da7757e3880509edc8c31d0850cb6dd6219361d")
	header, err := types.NewHeader(parentHash, big.NewInt(1), common.Hash{}, common.Hash{}, types.NewEmptyDigest())
	require.NoError(t, err)
	err = instance.InitializeBlock(header)
	require.NoError(t, err)

	ext := types.Extrinsic(common.MustHexToBytes("0x410284ffd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d015a3e258da3ea20581b68fe1264a35d1f62d6a0debb1a44e836375eb9921ba33e3d0f265f2da33c9ca4e10490b03918300be902fcb229f806c9cf99af4cc10f8c0000000600ff8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a480b00c465f14670"))

	res, err := instance.ApplyExtrinsic(ext)
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0}, res)
}

func TestInstance_ExecuteBlock_PolkadotRuntime(t *testing.T) {
	DefaultTestLogLvl = 0

	instance := NewTestInstance(t, runtime.POLKADOT_RUNTIME)
	block := buildBlock(t, instance)

	// reset state back to parent state before executing
	parentState, err := storage.NewTrieState(nil)
	require.NoError(t, err)
	instance.SetContextStorage(parentState)

	block.Header.Digest = types.Digest{}
	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotRuntime_PolkadotBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/polkadot/genesis.json")
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

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1(t *testing.T) {
	gen, err := genesis.NewGenesisFromJSONRaw("../../../chain/kusama/genesis.json")
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
	gossTrie3783 := newTrieFromPairs(t, "../test_data/kusama/block3783.out")
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

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock901442(t *testing.T) {
	ksmTrie901441 := newTrieFromPairs(t, "../test_data/kusama/block901441.out")
	expectedRoot := common.MustHexToHash("0x3a2ef7ee032f5810160bb8f3ffe3e3377bb6f2769ee9f79a5425973347acd504")
	require.Equal(t, expectedRoot, ksmTrie901441.MustHash())

	// set state to genesis state
	state901441, err := storage.NewTrieState(ksmTrie901441)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state901441
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie901441, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280402000b207eb80a70011c040900fa0437001004140000")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 3, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245340244000000aeffb30f00000000054241424501011cbef2a084a774c34d9990c7bfc6b4d2d5e9f5b59feca792cd2bb89a890c2a6f09668b5e8224879f007f49f299d25fbb3c0f30d94fb8055e07fa8a4ed10f8083")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest))

	// kusama block 901442, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x68d9c5f75225f09d7ce493eff8aabac7bae8b65cb81a2fd532a99fbb8c663931"),
			Number:         big.NewInt(901442),
			StateRoot:      common.MustHexToHash("0x6ea065f850894c5b58cb1a73ec887e56842851943641149c57cea357cae4f596"),
			ExtrinsicsRoot: common.MustHexToHash("0x13483a4c148fff5f072e86b5af52bf031556514e9c87ea19f9e31e7b13c0c414"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1377831(t *testing.T) {
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block1377830.out")
	expectedRoot := common.MustHexToHash("0xe4de6fecda9e9e35f937d159665cf984bc1a68048b6c78912de0aeb6bd7f7e99")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x08280402000b60c241c070011004140000")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245b50101020000008abebb0f00000000045553c32a949242580161bcc35d7c3e492e66defdcf4525d7a338039590012f42660acabf1952a2d5d01725601705404d6ac671507a6aa2cf09840afbdfbb006f48062dae16c56b8dc5c6ea6ffba854b7e8f46e153e98c238cbe7bbb1556f0b0542414245010136914c6832dd5ba811a975a3b654d76a1ec81684f4b03d115ce2e694feadc96411930438fde4beb008c5f8e26cfa2f5b554fa3814b5b73d31f348446fd4fd688")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest))

	// kusama block 1377831, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0xca387b3cc045e8848277069d8794cbf077b08218c0b55f74d81dd750b14e768c"),
			Number:         big.NewInt(1377831),
			StateRoot:      common.MustHexToHash("0x7e5569e652c4b1a3cecfcf5e5e64a97fe55071d34bab51e25626ec20cae05a02"),
			ExtrinsicsRoot: common.MustHexToHash("0x7f3ea0ed63b4053d9b75e7ee3e5b3f6ce916e8f59b7b6c5e966b7a56ea0a563a"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock1482003(t *testing.T) {
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block1482002.out")
	expectedRoot := common.MustHexToHash("0x09f9ca28df0560c2291aa16b56e15e07d1e1927088f51356d522722aa90ca7cb")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280402000b10c3e3e570011c04090042745a001004140000")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 3, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x100642414245b50101320000009759bd0f000000009e210440b679b326821b2fca4245b00fbdf8805d3aaf61cf1bf503394effef513cba364ecbebaa6529ac9660ab4c063bea175e9d6ca0685d5df4418fef6d8f07c27a0f957275efc0ba4e50bfbd6e1384fe84dcfda5785e80341213d23fa3600104424142450d7101d10204b57800309efc8132f8a177637557c9c2d9be6a4ba1e31e1b8c32e1e699ee170100000000000000964bbef8761a9505e0cd68956f96929cc6fb56937903d1681e73aed2a9659619010000000000000002b987ef285b8918c77eb19d35859a92b6074f595119246a95f05b5aed5b3a5f0100000000000000bcea83362dcd04d5b701459ac0a9cb9ed9ffbb6199bcce4022129dce2442ba4e0100000000000000162257d27926f8d958a9b763e36899d0efd245e218ea72c29b9233094d8a4f25010000000000000000dc83b2e0ebf20defdc1feefe506cf1d72c17022f318b033ff5889756ccd962010000000000000042e274ddcb6310dc1ba6196dacc48aa484fbf8d1229277255f24c331ad8ab6200100000000000000fa110af82bc8810ef86db861c0590432dd987293f7b8d237706b9a765bcc5e6301000000000000003ce145a3a3cf58cb62b7ee01b3c25cc52a6b648590ff66c5e90bc1b0e64a04380100000000000000da6380d552c6fce2effa16055b935d40b000f42e9a7a448a50ba273b7a67977e010000000000000038f1ca1d81d8566cbc6d6aa6caa86032f144799c0c121aa8cbb6a2cbc1c4da4501000000000000008ea2f21d319cd3306bfc46590c6d06498e6884d90f42682e4f7a104ba1b668020100000000000000eac271b2814c397e8bdcac88c9a364355ba86119de6f1d221d6558bd2c68fc0f0100000000000000c2e8b87c8ee12b9c2f412824e560d9757a6df8b94e9958dde2a03a4455804d7701000000000000004a15c3e9ad2addb360bccc771dac1498848b9d4c99229616d0aeb70d4c8d965c010000000000000060d180c03896ded51c26326c10ff7d550e2f7ebca27b80980ef5305245caa84e0100000000000000d6402926eff84f7793792b280b612246e9ec5ea12cf9341af2d6426ecf884a51010000000000000012fd99c560cdceef73f67478e8fdf160daf585603f229ca34d619d303e55503501000000000000002e1ad1c806e59126783cc0ffb2634c247296ae93e18381c543f97388e264c77c01000000000000005c68595ebd57820549b8e3437d02c9f4a7a59973ec568c4a018615de33e387570100000000000000aaca301bac17ded6fc8b7b18756214888367786ea19313015ef8142300803e1d0100000000000000880f8d97fcdc7021b3c31b510833a02ae0b3cfab521e858527128bb41b69737c01000000000000005c027eb5abd497132af46fa7906a2bd019482c0b9579382cf99a796f42d10e710100000000000000842bc40308e126f84e37d2548ae59e117678b301b9cbb81825f76b51eff3353c01000000000000007a2b08afc3ae8dd6142454c92f89da44b59d48b77a4cb2c1f70b802d81a5616501000000000000008eb08114174368dbcc919de7a24c9ff92fc687e7460f5081f2913de6bc55a02601000000000000001accbd204cce7d8a5d5af6bef8e16867f8c8117e90f6de50ad0a43032144c80e0100000000000000deddb694fb8d803ab5d187226dc8af702e268413e02f11b7bce61b54ba49c37c01000000000000000c1af511431545658f228841176c169c225b0fd31cb837bb6874507e2be1667e01000000000000004435a1e26a4307623315c9ae2517542e2cab93573882b9193de8f45c4a35e12501000000000000008ed8b2c7248c7145ae63f0f007f1f5d10e13bf77dd751e694c1bba068b7200520100000000000000648cba1b55dc09e1d0aafafa0a48c7d1f59914335da50c439aa76912ae33f028010000000000000064756a866505ec05ee12cbde1c62812a3f8bf74358a110116d3026b9d422500b010000000000000058c724a9c349fde3719f36a85ab9777c77804c6f136beb400597afbb340b0e4a0100000000000000187726f667f54f5f86407f33dd7ebcf5f3cd6989d513f3489c419c1742a0981f0100000000000000c0be316574806d2b59d5ea77c2b00ba9f473d7111ce31afd9ea35e1a8c2dd7360100000000000000f678ab316ca11894b3ee1955fbcc110bce84f7c673ca29b5cdd5869b880a5e68010000000000000052d3be9bb543ed32f4311acc7b3ff24b02de1b38f369169a5c70d7eb8f95935c0100000000000000d8fa2576e4329e51e2468031a612506121a7178c1085840b0d55192e72aa304c0100000000000000e0b214ed0fd7aa0f4f7eb9704584494273cbbddf5e0a7c8e822a4b41b6a0991c010000000000000046a66f7d163b96dbeb44324a9f2557091527dbf83e1d36a2740fb8f549372f5b01000000000000008e1b17c7c4d08fe7f956609521f4bffe7f941f5067bc6fbdc835a2d8ac294527010000000000000082c8f9dfb03c214735c492afd7792029e55cdcfddb6c0b4a3be1eedb4793b81a01000000000000008447eb61c24a4403d3191397bb12d7080907666a1701111f9f7be5654a52063d0100000000000000f89b7d34e30056a8fb7429462065acb315fcea481b84a9c20f8a1125eee0106001000000000000004c3b7b04d9714ccb6024a358f3e41e93d82682567a894a0fd030afe03f0b4a62010000000000000056c613fd149197bc382db5aa69ca4f1349454a9fc5e4aa22f1e9c3d8852c84650100000000000000767ac80a7d18e6862e1053d9f496b434a15bf7f7d4a68e27ea23c1bbeb5cb24d01000000000000001072db214c06b269fa0ffd213780ec3996114b0596f25d9bdb01e7a7bed9bb2c0100000000000000fa62539bb1779616fc593111aac86c4840f6da936fb4091d6b3d13e9760d9044010000000000000046694ae4b8e82c078e2580b0a38a8877c838ac3ea2144f7fb9bfefe6642fca4d0100000000000000e6594a327ac9dd82bbfd0513ed672675167c55177a17273467bbd220469ee5350100000000000000a826b82af0d922c80243de032d68f9b289e141257dbe8de0dbfec11416a4570c01000000000000007e071d68ff497493bd4fb4defec748df5a92d22d05a658d34ba041c1882ae7100100000000000000b87d6ffd38143c176320d877c3a35b13c35279a86687e154b58c0ba527932a3c01000000000000006096e853ce12c7ffd178bb26b044ae14ab0e64dd47b8c956574f0d4d2e8bfa68010000000000000038d8ae2e30bdf01914d1a945649a8ed71641507d9560694ee9816f5f3e3b201c01000000000000004a22a2403587a47cfb1f62228ca880c0dd6d075e39a3a3e95dac79f9ca91c95901000000000000006aadefd8bce06224db3c08058b697d1c5667c8b051deca1e568e98f8636e002d0100000000000000e8d8a3ab46970ffadcab32b165cefab5ae9a330fff302b215ec0d72ad73aec090100000000000000aaa07c84b5690de59dc360397351cd7a04ff14c4a7acfbf94f06a11a35b9dc500100000000000000688af6f5926b70a73d000d6de3f11a58bbcc7ed12f491a9a664a7d34b293c20b0100000000000000ce995e13776450fc419200f344645dc5ec2ccad95da141a839adcb93784912520100000000000000a0e844d2b4a21ca429f4ffbb8ce70f34851220bfdebf29554aea2d4bc5fb9b440100000000000000f08014d7ecf7f84e4cc405b920998baa444e0201faffd2b8470825c467d7235d0100000000000000640a3a2794bd7e47899cd1d7e0ac1daabe4c5a7d8860a962637a6ee193af08170100000000000000401bb8d2fe09d8b6d04e9323a673cefa4d6052bd65286919c68fe8531321a64c0100000000000000148798c41d796e1b561e1ef82c6111e143d70beb79039bbabc3709b5ff957d520100000000000000a612c3e9b9d981933f4f490a7b499383ad1ec1e31c394da8d9f50098a8cd2d6d01000000000000004a0501294b8029e24a88b4d386e7948cc0fd3a5bd099bcb4f295f15d51a5857d0100000000000000585b918383075118154616fedd07fdb5b8087b64579f79975584889c4e38e8060100000000000000e4d11597912f30bad9257a859caeadeb48a9064fcefe0ad8ce9bc5960fca0d100100000000000000d6b5a996919814cd3a438850c8655388c4c454e1d0d4c892e1a8d23a092b3a5f01000000000000002cf89e6d048e7759e19c72d50ad444b1e961b333ad2f848d2791e059c4caa73b010000000000000058ab66b2ec837ff9106bbbdcd91d42129bcd204b4f10d27432b5f52717e32c5e0100000000000000228fb2a647a6c627a6cbad1ed196b56462d1aa24d35be38f4b27c852e487c2250100000000000000d0fff1f8dd08368ef87606ea228435de58edeccf5639e88623bb6b8d6ab066610100000000000000a6adb2b070db495aaa8b42273146603fef4bb3439d89c223e7678a8730ba35070100000000000000ea05c190c48078fe0359256b9801749bcf94542dae3a300f433654b8de64231d0100000000000000268f481d9116197d487ae4955791849a1d5f9c68c853c4bd47915033277bdc7a0100000000000000089a6ace8a07d359554a589baafcc9f8b4a61abdb1e311a50242fcc7e87413520100000000000000e0955ff956b6ad5b1a8a0cc7341d9c17829bd3f79d1e523f6f76f1a01673024e0100000000000000702eb01695e26f92f0ffc885ee0103c13571a9a4cb1c383722aaf3e38f6fc8380100000000000000c6a297a97d28483000abb87cd5309b4c316dd007cf3957a0f7e6e10b3867bc6e01000000000000001233162e12deeef969f4f7a3835724e676e94a262fe52985547ccf525ba19d1a0100000000000000e01777e43c6707d113461e7f3c13931f6e743215f1a873862705fca4b8785e25010000000000000040866ee02fb29e38c2fc4e1ea1a3cbf80061e48d9653ccfe705d04292d7da62201000000000000003e859f6b7e34f035889d6a77de24bc76dddd9c21fd2943bd98b9cad3128d2e520100000000000000c86646ae19521395cdc1b68799095c29dd44a9e6659724dc5ba7347875c0696001000000000000002650bc867719e90c43b97dd6680a9c05b3dbcd5f99db413e7c8fbf3070f4975d0100000000000000b61f34196133d7c9add42fafd835e77516553cb2bc453362a68589197e6d7702010000000000000052e018ab8de9f83f117c25e55e1fafedf2658d2b13adb1287f53dbc96634335c0100000000000000903d6ae813c70e1b8a123c44bf757f5efbefb1a15a3f3cee133cfcacdb21c5490100000000000000c89f811a11e9804c1c4030b05e93e2120e46d78d52d2c9eb1bf137efd241c61c0100000000000000dccb837dedb52b997835f88a40f3323886328389004e89c9df128f4e2f99df000100000000000000bcf7e0c0ec0581a6bff67f33fb40a078ecbc4bc61567a50af211a1a9d80dfb520100000000000000a42b2f9e12f211dcd50b9a11179b0dca805943af2c934488e9a9231eea058d7401000000000000002438d2c7aefdceae24436b7c67e5e94e0f5647dbf053b99ab8836c498a7f277a01000000000000002e87e5b7c59c45c4da2c6c76f194372257bb0d942e971a6b38c0363ba935c640010000000000000030673798468c2fb61816c9588b220625d24943830de0fd9b15438b348060f4750100000000000000889d752ae5f531fd6559effa5c77a4ab30805a74a85881fb22f690215f297e050100000000000000544e1fad3dcba3a935652342e6cef05d7e5f7b67eb6a523f52dff7872683d21601000000000000004aecea2529a1e04901d14672828a7a244248546de6c7a4347f00298bd7cd58000100000000000000aeb888e3fa12ef194d187e32c4438314633987243b4be1ad75b8a4ab312e1103010000000000000076168fd4cb72943677f41c883d5950a0262e633b0695576a2e7fab85162534390100000000000000482b4426073f194d49705dffaaee9aacef27fd4fe5cb33c25855d0368fdb1a4f01000000000000008889361702f5cb1061a697985ba1fe460ab9180dc973307d36235db478b9722c01000000000000001c08ea2b20b0943a0f4af1574ceafe4211bf4f1ba3708f0ed56723f51ed6ab2b010000000000000058445556f8f65ffb279e075a9c235cb70121620eb51b0ef6779fdb1536b0bd64010000000000000030a09710ccfc8f18db0ba7b673daf98e19fedb189f844b7127eb5a348b3ae2130100000000000000ae9da6ab58910bd5842c103a816b2b483072e2f5a71c73b38f725565d9be0b0b01000000000000006a2d4ee2416aad2e14fa8322d189a9680253e40cc5fbcdd7c5ea94e9f791e635010000000000000056c6e52b6ca6350392a56c4ff42931ec2c6873ccd7964e1dbc8b5eb4c225e81c010000000000000052222f72eab42511f1bc3c5b4271cdd3bbbf84a755183abdd02f56930aa9fd10010000000000000032753e56e472dea44aecb76aa1ebf1b41859b14fb935a990977eeabf64032a360100000000000000c8c11088e72d076a8df738a24f3b63677495377c675b8e1377d3f8ce9a7fc6640100000000000000608fec4564befd5867b54a37311cc1be2292825c55eef6bf5a651fcd8191250501000000000000004a3672c6ca9fd22c28139bdd742b21dc736487994a9e8adbe1fc9d838b65476501000000000000001c61da7c83d91b12ad017159c5161de07b82cd4ce88696e2a3835c87db94b50e010000000000000072a07ea9ceecd162f8029cc58e0217169206ec69bfd4d5d45a85ac3fe64d9e580100000000000000a46dc30d881365a57c356d2b2c730449abb16e87a233d986204f03071aa13b7201000000000000009e2fcf735cc82fbbde1c14d1c6b6385a7f40b1b85717d8ae06c6ca1bb3ceaa3a010000000000000024175533fba299cc842442773f2f043f3bbaf67ed9e1ed2e44740ab4317c87550100000000000000ca3a62f97737cad7f7b0ba76e29bfc3546e99115235436427087bacd7ef6726201000000000000009a424f103bbcb2212b61205d216823dce73b807f27f5c24b8e0fcf460554696901000000000000003c62b94c8588e64334f5238f1fe39bb8665d6d2396c614fb1f0094acd94fdf1401000000000000001c01d34a3af1a67c0011564426df4aa7afe0d4f270c83ec4f1d7a54ecf057e2f0100000000000000c8d5277f99c6d34149e5e08ddca8edb2cb012b173fb96ab66ebd1e43879de62c0100000000000000fea159528386ca237ea93deb9dcb910e16272b24d351fa0d065cc29ae6a8a63801000000000000007ca2ca060d3c746afbaa98d610fc05ece6e15470506172a844138b1e34d38e04010000000000000080a96ace5196a51b6b17b7b4976282df1e9a9b62e98d18a83a0f0c71f0dad8400100000000000000f29515bedca3dfdd9f4b0dd5573fb5365ecb64afe2614beb4daed99d90ca314901000000000000008ec48eb902823704af3070777bb462785166b2b2d24e18e8b5dcd30d5d4856340100000000000000e6483b29203441853526bdf735ca30edab2fe0365fe84e29c51bd437ebee7c330100000000000000a84c137cabd458211c20c7ebd6ab69499618e090cead62991eb23c372f7a1e74010000000000000082028b177d8811df7cb1fb3b7138b58a767bf0058a36e5b949d9de08063fc96f0100000000000000902250f658fec89bf1e514f991f92fc55b29efd76b435bbff13947a8dbec1d1201000000000000005c64f23ff990859550aa77f3a86068c6da68171586efa4867cf9a4ac09be051a01000000000000008a468fee7d82ff5c7fe3799aad72d96e794a4d88b365a3280825644d25d9f92b010000000000000042dc3e1416a1f7bf0719ecac2789c08ba4595dae10d7b0f3adc5fc6435f0d5330100000000000000960830d9513a0e2518880595395503ec90dcf7c9956bb8f4f12424288f50031d0100000000000000e63332c8237e79e8147b3152db55f98ed7a2b746b1d4cf2295ddedb853abf510010000000000000058d1efe485e09865b483c403a1d47813eab134d11d8c92db7e66b7b19117505a0100000000000000aced024c01687658713b3dbfc8d41063fbf7db90fe5a47c83886bb51ca7ca2260100000000000000d86d05fff521e4d46009a2e0df9abaf5d9f63a9a2bf2e783a59b5f29a9e15c36010000000000000056b1aafb1160013fabe4577211a764230292e5cbe59fb1c7b0d3c0dcace622560100000000000000cabbd685cab2c594a2b42738968d56201e590683870027b3ad6be3617d966b750100000000000000fae9a33b3de208d85559cd533861c5c37863333bb852631ad46be3165c8ab03201000000000000004e20c133f7e99fcb94951822bbd58107022dcd50c48c03708fbfc79a54c86d510100000000000000b2e41c634e5e673a5ea21dd806717466fa2ceb279a55a8e774868399488c996a01000000000000006a3398a599f2f1cb856b772319ea571aec7e988abba58991b5fb2dffd54cde4a010000000000000034c0505fb88a037bb69fb1a4debfae9bb73817556347a624a135d72144b8b95c01000000000000004804977adb1b062fb72a1d17f8c6d42368860d1ef48fccdde0391518c3423e240100000000000000603e917a6977f283d18a0da400ec87d0f16f6a511cc68092124bf2dca9acad1301000000000000004c41aae61cdb1f2456e7faaa005e85365a5ac657d8f9b7abe6456ac3dbd49c590100000000000000ba6ef6af1bfa68003c889598792f299f79b73ed58a5c584feae0168ad4c3ff6f0100000000000000a48135caded700203e1953d41ab7418d3a0ea4be6c5ad9643a099615ff58292a01000000000000005420027f4a8035711041cd1a39230e80895e39906f34ffde192df9aa3c1c810c010000000000000032be102a24c2ecf582b9e593dac8f417924651f3bab14adab907e833b1eb8953010000000000000038ac7dbc87fc3d12c2d93343d55ac364d15df3abcf997750bd242188fba3b43601000000000000002442758774e775068c74220866b2cd27bcac9cf5f84da054a391a4f263ebf329010000000000000010295f1846d4891b952226b23b02d34e0440e78dc52b054ef2d16e963228e40301000000000000003ae275a0b203a05bc07e96ab102645a968474f2953f7c360c19939f9509bb367010000000000000014c4fd324955a25b44784231523351a170b791b927da2718a9bef9e1b84c70120100000000000000d031cc5b1aaed244e6fc31552c3da8c4828fad90ed258a07f194250f71240578010000000000000040c126a7025961a41bf05b53af9798aa2aa3c7bb3d98e82659b5f04563c1fa5f0100000000000000ac2166d5634e631b96cdf44c4676c208ab4aeeefcb6b17fe1765e184b7135b70010000000000000094ca3fe24a4e1c62e189e436bd952077091dc33046bcf9c2b1fc61077a754d2c01000000000000004040782fd0860c362c58f30bc8b4c5442794eea0d8eef07eb967494c10a2f2500100000000000000eafaed0fe1545946768a9716ff1fb375116f68a7d529df45445b3e5d69cf23580100000000000000903ea8a34a7166a64e18a80c109bbd5a0b0aed20796e0f790b833e4851a084420100000000000000f8038cb26a6700be115e9cf5d18b4e1ebe3035b423b8a3fede633ee6a76b5b0201000000000000001451c142367a76b17c8f2f3a2e43b671d165ef793033b1394f06abf4d7d4501101000000000000005a378b6b9203df494f791e4544c0bdbf257244fcd262b5dd1322bc679a2f992d01000000000000002803a214e2b60b9a1a7062115c44c03c1bafdb9595f6beab8d68f103ef65a07a0100000000000000e2a67e80c18ddbc02ed0c1dd94de28d3ae64545510bab8441de7f9619807800301000000000000009463ba6873947fd43d601bfaf816b8f2856cd8164406079b302d898300c2e51401000000000000008661b275126f190b8b03a015c482b6aea9f7a8a520669495f9ad36382a9a416a0100000000000000068796a0a56f351bb079dc8c6ee8fe05d0c22868aba30883b7c92a771b7bda5c010000000000000036a2563ac9161109d3ca6dc1943f75e2caee529b099b6ce8d38f4da089f268070100000000000000c386d5e8612b45057bae741e105afcde06af5b2248a3944fc5f8c2c1f49c4b3b0446524e4b9d7001d102be3176ba563578a27e5d499afb69c3f1272f8f33fd8fd28c800625fce8b053d50100000000000000d1cbb104eb5f010bf887b8d3ae2ecd77572bb17c74b2688bfa2cc175101f33b1010000000000000036c3ba9b05f6814e0de9a3ab1e108141416c95d784122b76ee52e3b0e8c86aba010000000000000047348abc24fc6f4797ec0ddc9c55f25a54bedc2ce7b2da9ee119d33f09f0d1e50100000000000000ddffb4a5a93a9e724db26abeecfc5fe2e89cbb9a5050d44d0c3a219cdc932ac401000000000000007af1312fa10df57db7a897af5faaa7d14a66dee1c20972c13c8e36fd6f5412270100000000000000afe37da24aa41def0065c9ed84db03f4cdd2fbc34f675b0942a135ce881fe47201000000000000001c29d7ce15b7fb3d22341fed44715d8f34c356a32251e6441825006a62b72041010000000000000090bad22c20a221271b1796a03386d2ea1e4f1707abd25a77f802d4144c3b3a2b0100000000000000b4295abb37df4a3bd35b74e4fd436154812c12f136945c6a6e745d438c22598d01000000000000008b1405c139148dfaa0a4d9cabd6b6f005dce1848df7fd30bd7eb0ae8e00fa5880100000000000000c98bacb2b0b5e88da24060475191f4ed37af1401b2bb037ce499e3243dc3c9cc0100000000000000b74f28c43f03e75b603d48d965006da8f539ed1b905e1e2460ca562a17ef8db301000000000000001797dc668e97eec57830acc88c1736599a1d1eb8175c63224fe6db9d4b43f9850100000000000000381bb2f0ef2a9384e93fe56774bd6417277dc3a5a76d15ac55bfcfdd0f907787010000000000000032056774822ed8ebdfb83d794aada5199e601b7ac327698e1550ae7a0f6758620100000000000000bd5a26ac70fa1eb1eaead625f4d29f53eeb18f45019407cc116bb1ce52d1558101000000000000007f6ceb4f00bd5c6fbeec01831f900b83fd7b4e0d13005db6c1c3e8cbddb957c10100000000000000e1f6b1b1f31e7617c76c91fa702ea40b34b20ce5669f33d66811e5f245f1da7d0100000000000000694d99913da55b6ee2b72224469ce328a615db6479826df186ac516d0435959f010000000000000023e2c6cd29c9287050211d30ebd5af88680678d40c841e7797709a6f899a26840100000000000000a130eb8f61f41ca057c7ee5ade07b74ded056b6cb7d8470805888a269cc508360100000000000000f1e59c4c3d8bbf74609dc42e2678f28d3e79bef0c057cc2c15cdc448bea17855010000000000000036cd1eebeeaea926936837b65de98409c469d2438b9cd13f9aecd9828f2320f80100000000000000a6e890135689a8333ca6c55999d8558ca4ff8ae7f2f389378d2efd98d9a7012d010000000000000017b50d72a7b3109874ce67d2ea7e22abf891b5faffa3ef4a8245d54878cb021301000000000000005e7a5c5c33ae82aa725c36f84cb900a3dc5f29fc0331554d95203a345a31a9c40100000000000000c02c56fec10f424876ceace65c21c231214c1ced594725a0d6f5abbf34716de60100000000000000e39ae6044c2a41c024df12bad4484af6b824a4d1da2ddeb99f6751a2d8f41b6b0100000000000000480eb3fcbf3c885bb57d9dc1ef1397117f69db1b7ea14483029e28106273b4850100000000000000a5ce03e4e99d5af6147bfed01871a1de10459f121b205faf52c52ec6eecc9a210100000000000000f45066c35babc6ba27b7ca056df70f04dc877dbcc0bd9ac8c9d82d56856b1500010000000000000098cd7c69c861ef3ec32eca898456b9bbca10f8b2b5589db075885416e80544ef01000000000000000bbf8f4a346f74964fc07987fe00053d72f1f132d4964e7e5012bc9fa460a5030100000000000000d5703af4abd74aea1ced12395b3557dfbddf41384d20a15e3c3d70e6f696877f010000000000000052d43d31d31eba86ce8ca4a89182bd3a2963a2e5b0224a80780cce6f2cdf6c330100000000000000a8331fd9318a6f27d2a3da1dd8b84dab2ec7dda2a464c648e8b8a7454a75579e01000000000000001bf458138d654c66325b2b7d3aff0c41109f3f874b11bed59f7c4ffd702fee210100000000000000a50f9e7484ff6a8ba136ee3842d0d5b9db4e82bf55ebb1f7e20ec903035b5596010000000000000087fa0fe1dd2374248be1aeaef7b0bde64f08e55aec77d6b0b8ddcf1615e01fe70100000000000000596d81e0039b9ef6266345878349fde921c94c20d16b7692ffc1b089807cbd9201000000000000002868883051b17c28d5cc53dde5a69a210208402bcead13086cd43d19aaf820d4010000000000000086299a4880451a9f5793f9b74d1e680e6deea0bbe84c0730dab7a164e23df8a50100000000000000c1f633c7f4c65a703def23d07e85e650e7a124383b9f4e71dadf6e938f2cc4290100000000000000182da31370a90a76902f45222b29432a72474fe3aafa7c62d368111f6214205c01000000000000008fe7f46a1b1e0dd662da1eef2413c4a05f8bbbeb03ef7c71d1a8f08bdb53ffb50100000000000000ec189cd23dfbcd51661cae853e6dc2733391c4375b75fec40b37ab56bb1a781c0100000000000000edeca0f369da1d06ca877e340a26edeb9b0ac6d55c49a3d9a39401ec27e4a8dc01000000000000008b6095191469b75cbb2c9af9c72d6b4365e53c06e928b15580a725edf2793e5301000000000000006a98fcba174345cbef21cf2390349028093f1c485c559afa25e73b4c7585dd4b010000000000000020a94668bd0318e90ef37861e37af261812f4dfee97669ea48d02871cabdcd9801000000000000007a20bf7c9973bddaad18c1e04483b18dba1574089effee6765b4aab03db6d23501000000000000009267d47d901b7bba2dfe9d9faa6a68e6ff2c217bbe1de2a4522cf94604c49651010000000000000055eb142abb22faa5c26afab0b76b180590f159b688641d9db6f89486b484d65a0100000000000000a71fffea17e3043f4e36c2004cdd7c7f8308e238efb2801cf6607f5e0c5348d20100000000000000382259ea2c100680295e728b078969f0ef7e077f858171c66059d002cad771d30100000000000000e3dd7d97be697c1066d4d3f3bd4e3ac21d935b34bb7bd4c7ddf87c7d7673881701000000000000002f0c50f427c7772dd9c4011fe2e34278efce6aeb716c173a5f4ba00b70f5a73a0100000000000000f4906ffd03b2fa8aaca2c04f6d6a489d9476c3e7ee8a161630b1a8fddcccbbc301000000000000000caa2b70176fc6f9fad111cfbf9a1a8fe7b2fbb92211c36fec2a8102d169cd9e0100000000000000f26c8b40fe23a64d3bf0fbad27ea08edfe070db5f65fb28ec26fa743ad6df6f60100000000000000a606b66b8466c1d7085e0ed1d6146fc322e257a55aae164f3a2e3812e212bd190100000000000000bbfcfcf1c9a6a576d76559e3ef69d5a13a2751f32661c0798d9b32ab017aad5201000000000000007fe53b7d4f2a836b30650cd5569af304c47fbd28d8c7fe39f002a000fd464e7101000000000000006f9e97d7e766c7f369e931933a724ee8860bdcc31dc81108c8a48dfdeb8646ca01000000000000004f1e8a61e5c40431e0d1d91cac91735a0cb2a30862d99a3214fbe0d79662cd0c01000000000000006832d483010e7c234350719bec24e9e7af9304ac3280ab24b1c6d7875d750b81010000000000000014903a1932f729eec51c64c8652f5a7bdf17dfa359799e550f1083e16457ff270100000000000000acc1556550579ad3aad1d51c84f355088bc39c586c9ff897dbea3d2ee11466d501000000000000007db403c44e5a3477c87352cf349111957a3771fc0e151aeb2dc46b3be272ab800100000000000000df8dce4bce99d3a6c7c2dca5a5a216ffd023888562026cce394e4d5e06464c9d0100000000000000f2d9e6989cc7f249cb708da9a9c4a2f460532c39985fa30c9f23d8bf30b2a0cd0100000000000000f7f82054b1489f15dfe4506d9f88f579f782d35b8379e497a3708355cc4033750100000000000000c4d1330d166175441af8f807eecc6bc9d196b7ee7199207ffeae982ff2ae8bc201000000000000005e7b52eec0c156214b2bbfba7c059cebd6db3c45835a665e0f6aa35621d7788c01000000000000004c8bd6887657bb9de51fb9d21cb73b4f85d210df5fe526f35bb6a20ef5e570b20100000000000000dd165feb16d970b3225b3854f57a1313ad6f454df614f8279a7f9670bc2e13d201000000000000003bd44e2e82498b2f4f7df63a496a79832d792d30706469b0cffdeccfe10383600100000000000000bc2bb43d3b59fe071edd53a1ff82229754099fbe4866ab14eabebea28acd121c010000000000000081975e9bce27b6c069616c3cbf163ff2a26b0002dd0f7845008c57059bfd0298010000000000000057202418c6101d0a0f8d03b611b768c4b9b8d2254b6c12c614afe86fb3879c680100000000000000f3d58d0f267316d55c08caf4638bfdf4e781a44ba0165a6589a6102370490d89010000000000000032649d7f5f5d7a5480c9988d0e10cf099674760c1799d1d9d286c65b6a089cb201000000000000002c435333bf117e42eb577e7da4817eb42b50cc5569e9636a6ec49a8231ee9c210100000000000000994df0e11b294c602e0b227c41902fe0fb7f9442e0dde56f5e5b48825b90a5fe0100000000000000d99439653859671b34738bcf6e584e65e996e9085b08b3cdd44951e17ac8f3b001000000000000008a3849c6cabe5082e90ec44d3b8d9816b01bd577d5fecfb109a547a5cb10d033010000000000000001dba1dc0533585985bc9a3a1176cd3ef1efc419eab6c7bec10917f348a467e601000000000000002e821f1558745b04cb87be3f276a7d9ecc388756d64fffc17301594f6ce88f10010000000000000081152139c7ffdbd5dcb1a57ea9cc893d3faec3e3c6a71624bc31828897bae9820100000000000000653c227d39cb3e05f2dfcd02c1efa499ba9f9c866e4116892ef3bdd27789c53b0100000000000000ed452e8c9de0a22c714eb0dad77c86d602bb19cf270c2ef0f043adfcb7b78a730100000000000000a2ce3196fc0345f86c6107b812d9a4c8bb556da96f45534977efc66c40f11e750100000000000000d28052dffc44331f1c4cab46b139a5cd74debc43e753b72076ea53d64ee0f97801000000000000002d20751aa6147a65fc4369ded68e83cd379a4d7b9bcd9b0c08cba4b70abd0c06010000000000000044c2491c0f07ed47038b9fe0715df4880efe2dc0021ab79baa16bdf8801b9e570100000000000000be81a8172aeb78bf198d11cfe9f0469ce948442f29c00f367d036750fc264faf0100000000000000e827303f26bcbbb420974d760094760e6c2be0bc4e95e6bfbaa203fc593dd2430100000000000000b877182d649bf6a3cbcf83d23cc7b36995f6bd15701ac9700b06d0aa753d6d8201000000000000006403c64325daefe5f12ddb9196053f91c80a65b233e9ebbc1750ea9da4e5762101000000000000003b08fe08e2666a1cdeeed6e06de040de1d5be68f4f7fc3f3f5544edd3bf2fe280100000000000000baff92b53651ee2bade8d835e16b9285175db0f478c73c4f15167941ba3ea635010000000000000021981f96f9177d61e09526c18868b549902fe948b3f4939cf0b6364ecfdfa3dc01000000000000004fac5246491c037b0a9e51616990f7f6a9c2e06bd6626b853f3449664aa1d9a30100000000000000ac8f78625e4fbb787873ce0f674a39a0dfd637e507430b776a2740d5cf06317901000000000000003d99caffab70949d7f61c7d413765aacdbcdb91f16f5466f19f632c1482a917b01000000000000003f1d6c381dca5539bb59d5acf90a2df4a75faa3388da0b0121681da1d2e161b20100000000000000826f98f9b44c4d5db76a7a0589bfef47bb56bace0d7da80f09b54d362a8149d30100000000000000920c6f0c681aebc27488fc9bba305c2192b4a158167ac8058ac05d4bac71766d0100000000000000d5af4d6c0ad7a8030ead1d12882e465d6b44b5fcfa381d1b516ee24e525973d901000000000000000b7d96129a974413809d38b63765381cf71a450879526ea81f85279ac62a6c61010000000000000096f8f9a5388bd2d5264095c5bf4bd5eeb2fc3cf0f3c9463d0bd2c5994025e8f00100000000000000399d0657551bc8c81d0600413d6227cb21e3fdad2c48436ecd68e4e1010a07cb01000000000000000cdd32be1023b699cd5367d0c13ed249b92f68f21e95627dd391ee06aa04c72a0100000000000000cb9a71fb5dc862cc5ae578f7a0c972417f29fd76b193e1117ced2abdcb5c5dc70100000000000000ed7ea587204974fd1254bb7dc1e7090922f2e38b152f4dc08646c00d1e6547680100000000000000fe67d8036100e86613063a48fc47ee960077c2dee57bb7a9ce55957ab4e1a1da0100000000000000f5e73f61a88d305f9f88d3efe06e964140f87505ebd5f644a4b92ae52e5797130100000000000000d6014f78638d11506b5df8d4533b2e3f0cb8ced5f174bb6673b33a5ac5a35f290100000000000000384ea72b9abb03ad899b91058b56d446e9ecdcbb7bf0c833b6553d3723bf85cc0100000000000000699b7afae579652ffa2e23a261220dec2d90d7570d2dec51b871f313574c2cc301000000000000008765598dbfe72fb93c2e1d148ad00f7266cdd4cbdffcec86711a9664501321c2010000000000000052fccafc1c664f8792f23a1bc468fd777fd9da29621f98c64e67f2794aa5e64f0100000000000000e1c4e93102228ef8281e734ebe1fcea1b05a68b69fd583cd5eefb7ac4b86dae00100000000000000feaa69db2241b65fa0e332a77f86f9dfc50c163353b4a5ccc058cf3c6f7725dd01000000000000005f8936dc6ea6bc0e914a77310d6ecd5f75baa11f981e9d6c18d78433d82899e601000000000000000acc7024c10d76dcc968a5f2482d7c5be47abe879e95a2fae7887206fd1e681f0100000000000000ee37ff73d973d47c403f77ae91d1811d82b683bd1d1b4af7d27a822121517ff201000000000000002ae51fb0c979ee950011794994e0d66d4c120d9429cc604dbcfbca94548033540100000000000000ae5ec6c8875f197907e4d3d2278e35240d1912ef2e0464baeafcf01fccb37d9b0100000000000000b16e5080eab65e0a9c24f4c056f95de843518ef3ec91b5dd4b07e2d6cfc49fac010000000000000064c99d806bec03038aa17580cf1437e43db68727821747b89ac0510a60699ba70100000000000000522314443f35f8665f15ba4e61123cc75f707470d1bd9e7a31e4c01acce249340100000000000000a2e8a586a6465f5f0f1f59397e3583e5ce4d04a72244f05031b0780501a8145701000000000000002d46ef1367dee8ac1227ef5f82d5aa6555f6ea215b1a3cc27c3414de760c842501000000000000008ac9ed615769af924b1083f091d560729519d17662c21d3182bf197319c2aa9e0100000000000000104854d7e85ff9a6825dc4de7e668d5345432b747522f73d8c2613ec8c82808d0100000000000000feecb15246516594f044600df85d1654bd36e5353d936945dc16496b77a15f93010000000000000064684cba9d4dd50d31124e12f9c476980c0ffc1f64ce9d33afa675db1bb201040100000000000000c29e560494d8b77f06eb95ec3e3f41b33dc5241be230b76c32b8b6d17c63680e010000000000000032c1db84cfdadc5790e1ec356cd4c2f5251dc2d8304c64039220ee81e13d45e501000000000000009ed7c59bcd539604aa5ddaccd8c0400cedb6d9cbde7666201dd953ec90cf877e0100000000000000f46f6a6697a3770cac385d1743a437d66ede2537c892ad6fbbc9a5145bc27ce7010000000000000025f2c599f64098c77484791ba096fdea0c041075d4bf37e2556c639f1ea52be00100000000000000fa54144438903fa56cc2f23596fd161cc127d29b933a3c6d23bd66fd15c45f4c010000000000000023b53b773dbbd1d980ba24f8931e5758484070f4b0cbc5347b9acaa0380d514d01000000000000001f81141a8c46fe442e4555cc9d2da4e61c4412c27e6938683eec4bbd43e8879c010000000000000082eb3046dd03a320330c9af754d88c238bb2e6525465b57a042b28e755492e3801000000000000004e49af86a14b8ceeb39bc7aaa567f5c4088df40476ff3e833c5fd298136ba9e20100000000000000d399d14bb9c069b89f60501e66caee61961f9466d5d45488770eea7161e527b4010000000000000035a9b95981c7cef53adcb5a45bfd6b9eac7063145c9c30bcac2998b78c68f53f01000000000000005737f7e7de716804af080f9ddb03352b5ac71cf51c65b7b87c7b3370e3d1613b01000000000000005e00e22fa583881d898d6414bb3c202aa74e3cad4ea531cb0053bb4823098f100100000000000000bfb4faa776bd4cec6ea19566475857efae2cd2afc009d3d84c9c6ce68e2d166401000000000000001bf3029a3c9caf3cb5bfcbb7dec6fa419b3cb42d85970a305e3f76c01afadd7e01000000000000007814f049c3eeb469a0241e7f91188660d954028d612d1730aae41a788861ed6f01000000000000002d670f6ddc4dbe590bc6890ff5d40eb93afece47c5d4897f8d5403e348ec261f0100000000000000fc43ea5f56363696ac1de3106b38bb7c7efbb1ced6597de27b34b64d6b7c46c20100000000000000f472db5155597dda08b4e8d82ae9e224cc7ad60798ce146eea51897b277939170100000000000000ff185b43021488634315977358d816c5da8520702359b6f1d2b6fdb8c0a754ca0100000000000000539c6a9cf39788feef973f48f54c2dc4863d3b28b5af4e08579471056a262c600100000000000000be53158c79e951df2b9ba1565182bf27cdecca36c4a28acd5576a909c989d86501000000000000009af27f2eb469d7d1db97d8836dc18d5e1414c47a9da27c6e4d21e21e33edb26201000000000000003f8640dbb084347eb3623a97c1304cff9c8f98516b990280e163a5ce35228a8d0100000000000000bdd403ef085a334fabd948e159eccdaecdbf24ecb53b984f440f3e074e96d57901000000000000002daa75813c73ca5543def4d0891dbc76e450b9fd4802a0bf80811db845f927640100000000000000d8936d6b05ca589a7400ce46c0876e028b383083c3c909e7434ab1a1228044690100000000000000a9a8c1d21714101306499e029eb6472378cb2d5ff5d42d05032a204738bc23f80100000000000000b8b23064d88162748adc202060ec705772b975e62938a3f2ca5b80f4e9db942901000000000000005d93cb9ed6b2d69d022f2e3d3d0ded29a29dba0982b6dc583664bb16408b545e0100000000000000d5e192a1d8666467c694a30fa1ba6420355bf0663983e14df991b9f97e25b91d01000000000000007be9a15295a120bd330f07e9295800e06cbc2165decc736f9d5f8ec580efad150100000000000000b13857cd183fee7fc5eac82874d253091f006d1a2863c9ba5d65b1367c68184a0100000000000000338eb7fff4c556816ca861e0c1f7111eebabc6efd4bdda9252b77ef12678bde00100000000000000663bc85589190c716bf3a13d579e703994a887136896a5c33d79469f9cc793e001000000000000002a977454e0036aeb0662937af4ca409dde3188c27b49c7d2a4e722c5dbfdd95301000000000000005d99312941c8cf08c1e4b8536ba520efe321862aea306a013f6e72df904aa99e010000000000000055017cbfe6279168629e1d902ab9368382ce6942699a646c97399b96919597300100000000000000b303f9c26f683847adba82de29bd2328a06c66c994e0737532f0978f8c64b17d0100000000000000f3b90e1e57d2ab26c6c1d0abe4ef8e306dd65f6325722da00655b160b27a5e8a010000000000000000000000054241424501011220a3a94a9199bccf411c3355d1e323ddadb33f7fde6d460f9aa0fd057fb161d41051581242634641a22de40fa9b30d6ffdfe8b1cfac5b7209a217196f25b85")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 4, len(digest))

	// kusama block 1482003, from polkadot.js
	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x587f6da1bfa71a675f10dfa0f63edfcf168e8ece97eb5f526aaf0e8a8e82db3f"),
			Number:         big.NewInt(1482003),
			StateRoot:      common.MustHexToHash("0xd2de750002f33968437bdd54912dd4f55c3bddc5a391a8e0b8332568e1efea8d"),
			ExtrinsicsRoot: common.MustHexToHash("0xdf5da95780b77e83ad0bf820d5838f07a0d5131aa95a75f8dfbd01fbccb300bd"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_KusamaRuntime_KusamaBlock4939774(t *testing.T) {
	t.Skip("skip for now as block4939773 is too large")
	ksmTrie := newTrieFromPairs(t, "../test_data/kusama/block4939773.out")
	expectedRoot := common.MustHexToHash("0xc45748e6e8632b44fc32b04cc4380098a9584cbd63ffbc59adce189574fc36fe")
	require.Equal(t, expectedRoot, ksmTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(ksmTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(ksmTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x08280402000b80eb3cd17501710984c2292bcf6f34fc2d25f7a1ebaec41c3239536f12f75417c73f7c5aca53308668016ec90c2318ee45af373755527436c4d7a257c481fdc3214634eb4b5c6711ae181827c378843da82c72191647667607ee97e0f0335f14d0876c63503b5f2b8986650304001f010200083e1f2bfd408d3b8d2266ce9b6f2d40acef27b773414537be72576ee3e6108b256eb45e26258d7ac737c3ad3af8cd1b2208d45c472ba19ebfc3e2fb834a6e904d01de574b00010000007506180228040052dac5497bbdd42583d07aa46102790d54aacdcbfac8877189e3b609117a29150b00a0724e180904001cf8853df87ca8588405e30c46a434d636c86561b955b09e2e9b27fc296bf4290b005039278c040400f49db9c8894863a7dd213be93b1c440b145cc19d4927b4c29fe5fa25e8a1667f0b005039278c040400e05f031d874257a24232076830a073a6af6851c07735de201edfc412ca8853180b005039278c0404009289e88ec986066d04f7d93d80f7a3c9794580b5e59d2a7af6b19745dd148f6f0b005039278c0404006c8aff52c496b64b476ca22e58fc54822b435abbbbcaf0c9dd7cf1ab573227790b005039278c04040044e31f7c4afa3b055696923ccb405da2ee2d9eefccf568aa3c6855dbff573e5f0b005039278c040400469ec0f872af2503a9251666fd089d0e84d3f6c8b761ee94b0e868788e0f60500b005039278c040400b41cc00e4ee2945ce9974dbb355265e39c9cf325c176147d7f6b1631af38ce590b005039278c040400d8e2f26a12d4bfc513fd32c1e5a7f14e930c3ef37997bf4e3de2fed51eed515a0b005039278c040048227b8300000000")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 2, len(exts.([][]byte)))

	digestBytes := common.MustHexToBytes("0x080642414245b50101ef0100000815f30f000000004014ed1a99f017ea2c0d879d7317f51106938f879b296ff92c64319c0c70fe453d72035395da8d53e885def26e63cf90461ee549d0864f9691a4f401b31c1801730c014bc0641b307e8a30692e7d074b4656993b40d6f08698bc49dea40c11090542414245010192ed24972a8108b9bad1a8785b443efe72d4bc2069ab40eac65519fb01ff04250f44f6202d30ca88c30fee385bc8d7f51df15dddacf4e5d53788d260ce758c89")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest))

	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0xac08290f49cb9760a3a4c5a49351af76ba9432add29178e5cc27d4451f9126c9"),
			Number:         big.NewInt(4939774),
			StateRoot:      common.MustHexToHash("0x5d66f43cdbf1740b8ca41f0cd016602f1648fb08b74fe49f5f078845071d0a54"),
			ExtrinsicsRoot: common.MustHexToHash("0x5d887e118ee6320aca38e49cbd98adc25472c6efbf77a695ab0d6c476a4ec6e9"),
			Digest:         digest,
		},
		Body: types.NewBody(body),
	}

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func TestInstance_ExecuteBlock_PolkadotBlock1089328(t *testing.T) {
	dotTrie := newTrieFromPairs(t, "../test_data/polkadot/block1089327.json")
	expectedRoot := common.MustHexToHash("0x87ed9ebe7fb645d3b5b0255cc16e78ed022d9fbb52486105436e15a74557535b")
	require.Equal(t, expectedRoot, dotTrie.MustHash())

	// set state to genesis state
	state, err := storage.NewTrieState(dotTrie)
	require.NoError(t, err)

	cfg := &Config{}
	cfg.Storage = state
	cfg.LogLvl = 4

	instance, err := NewInstanceFromTrie(dotTrie, cfg)
	require.NoError(t, err)

	body := common.MustHexToBytes("0x0c280403000be02ab6d873011004140000b90384468e34dbdcc8da24e44b0f0d34d97ccad5ce0281e465db0cc1d8e1423d50d90a018a89185c693f77b050fa35d1f80b19608b72a6e626110e835caedf949668a12b0ad7b786accf2caac0ec874941ccea9825d50b6bb5870e1400f0e56bb4c18b87a5021501001d00862e432e0cf75693899c62691ac0f48967f815add97ae85659dcde8332708551001b000cf4da8aea0e5649a8bedbc1f08e8a8c0febe50cd5b1c9ce0da2164f19aef40f01014a87a7d3673e5c80aec79973682140828a0d1c3899f4f3cc953bd02673e11a022aaa4f269e3f1a90156db29df88f780b1540b610aeb5cd347ee703c5dff48485")
	exts, err := scale.Decode(body, [][]byte{})
	require.NoError(t, err)
	require.Equal(t, 3, len(exts.([][]byte)))

	// digest from polkadot.js
	digestBytes := common.MustHexToBytes("0x080642414245b501017b000000428edd0f00000000c4fd75c7535d8eec375d70d21cc62262247b599aa67d8a9cf2f7d1b8cb93cd1f9539f04902c33d4c0fe47f723dfed8505d31de1c04d0036a9df233ff902fce0d70060908faa4b3f481e54cbd6a52dfc20c3faac82f746d84dc03c2f824a89a0d0542414245010122041949669a56c8f11b3e3e7c803e477ad24a71ed887bc81c956b59ea8f2b30122e6042494aab60a75e0db8fdff45951e456e6053bd64eb5722600e4a13038b")
	r := &bytes.Buffer{}
	_, _ = r.Write(digestBytes)
	digest, err := types.DecodeDigest(r)
	require.NoError(t, err)
	require.Equal(t, 2, len(digest))

	block := &types.Block{
		Header: &types.Header{
			ParentHash:     common.MustHexToHash("0x21dc35454805411be396debf3e1d5aad8d6e9d0d7679cce0cc632ba8a647d07c"),
			Number:         big.NewInt(1089328),
			StateRoot:      common.MustHexToHash("0x257b1a7f6bc0287fcbf50676dd29817f2f7ae193cb65b31962e351917406fa23"),
			ExtrinsicsRoot: common.MustHexToHash("0x950173af1d9fdcd0be5428fc3eaf05d5f34376bd3882d9a61b348fa2dc641012"),
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
