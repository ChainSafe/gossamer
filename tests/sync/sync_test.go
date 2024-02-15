// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/ChainSafe/gossamer/tests/utils"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/stretchr/testify/require"
)

type testRPCCall struct {
	nodeIdx int
	method  string
	params  string
	delay   time.Duration
}

type checkDBCall struct {
	call1idx int
	call2idx int
	field    string
}

var tests = []testRPCCall{
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 1, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 2, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 0, method: "chain_getHeader", params: "[]", delay: time.Second * 10},
	{nodeIdx: 1, method: "chain_getHeader", params: "[]", delay: 0},
	{nodeIdx: 2, method: "chain_getHeader", params: "[]", delay: 0},
}

var checks = []checkDBCall{
	{call1idx: 0, call2idx: 1, field: "parentHash"},
	{call1idx: 0, call2idx: 2, field: "parentHash"},
	{call1idx: 3, call2idx: 4, field: "parentHash"},
	{call1idx: 3, call2idx: 5, field: "parentHash"},
}

// this starts nodes and runs RPC calls (which loads db)
func TestCalls(t *testing.T) {
	if utils.MODE != "sync" {
		t.Skip("MODE != 'sync', skipping stress test")
	}

	err := utils.BuildGossamer()
	require.NoError(t, err)

	ctx := context.Background()

	const qtyNodes = 3
	tomlConfig := config.Default()
	framework, err := utils.InitFramework(ctx, t, qtyNodes, tomlConfig)

	require.NoError(t, err)

	nodesCtx, nodesCancel := context.WithCancel(ctx)

	runtimeErrors, startErr := framework.StartNodes(nodesCtx, t)

	t.Cleanup(func() {
		nodesCancel()
		for _, runtimeError := range runtimeErrors {
			<-runtimeError
		}
	})

	require.NoError(t, startErr)

	for _, call := range tests {
		time.Sleep(call.delay)

		const callRPCTimeout = time.Second
		callRPCCtx, cancel := context.WithTimeout(ctx, callRPCTimeout)

		_, err := framework.CallRPC(callRPCCtx, call.nodeIdx, call.method, call.params)

		cancel()

		require.NoError(t, err)
	}

	framework.PrintDB()

	// test check
	for _, check := range checks {
		res := framework.CheckEqual(check.call1idx, check.call2idx, check.field)
		require.True(t, res)
	}
}

func TestDebugWestendBlock14576855And14576856(t *testing.T) {
	wnd14576854StateTrie := newTrieFromRPC(t, "../../lib/runtime/test_data/14576854trie_state_data.json")
	expectedStorageRootHash := common.MustHexToHash("0xedd08c8c9453f56727a1b307d7458eb192b42830b03ef1314be104b4a6aefd9b")
	require.Equal(t, expectedStorageRootHash, trie.V0.MustHash(*wnd14576854StateTrie))

	db, err := database.NewPebble(t.TempDir(), true)
	require.NoError(t, err)

	storageState, err := state.NewStorageState(db, nil, state.NewTries())
	require.NoError(t, err)

	trieState := storage.NewTrieState(wnd14576854StateTrie)
	codeHash, err := trieState.LoadCodeHash()
	require.NoError(t, err)

	fmt.Printf("code hash at start: %s\n", codeHash.String())

	cfg := wazero_runtime.Config{
		Storage: trieState,
		LogLvl:  log.Critical,
	}

	instance, err := wazero_runtime.NewInstanceFromTrie(wnd14576854StateTrie, cfg)
	require.NoError(t, err)

	blockResponseBytes, err := common.HexToBytes("0x0a91140a200283b22f28a02825206b5cfd1668e117c4c1d3b49ec02a5b4731aff2c3ac1c4012a10240a65a5bad9b346fc4d86fe33eadfa0cd49d0378910b0af5e73159ed503611e35eb37903e8c4636bd5f01d9f9a18fa96949787975c9c3b78a0624ae95e3078da20c33605cf34d6de3155b72a0818608103915619964369e8599482e246a4cdf43df1a7500c0642414245b501030f0000008a22a61000000000821ded3b100e3f68e3ae8109c028e91b0d084bfa6458d5a28203e35b2d01e87bf4f8ef774929e4522b451e3abf892dbbb5aaf21ba7870db5e0efa8d481268e0159463ac6be5114cb953e7bb5d070e785b3ace95680bc6040bab4aed1575bed0708054241424501014ea332ce7e94687b6103adaf88e0fd370879926e95834ca700852fdaff0e260857572d8c7798150b0ffdf60d85d90fc6c6bf10c974606307245d4d4683b64e8e1a0b280402000b6182c93586011abb11e522042d00400c0100000000b6a695e095d905fda4aa41746c0a0752cc3559eb00687d9a13c2a01745290573f6fa11cf3569c116e125d57ec15723653e8f4cda3ec481df1e0d2938f9c3f6880c0101000000740d6868002741aad034c5ffb5dcf31db78f2bb2eb1349bb02dc17d445e9f837914ec9394d52b6d6a3301fc7597de47bc2e48f6f3965999b4ea48fa5667aa4890c01020000002a0962f0b38163dfc2ed4f18727d530d49c05b4e8dbbd1b2c3af310956651277ed247263db04acedb2076ec99b516dff7598ba1f2d328006d4eb04e4c7e7028a0c0103000000f4a8ad3422500b05b5e91f1bae5db4aaa0d576f201ef8f39104764be34b9f229e3e123aec9f98fc9a64b3c4323428bbee3fb358474e2142f273fc3218cc9248c0c0104000000a6a5ee07dafbff4e52ae279f3149d35023751e61a4174c95e3a1afd56ca3365b939b7437f87bda45bfc052a0b0a71cd731296ed6d9edbc127a8e16cdeb52ff800c0105000000b0901765a119c80051f5000b3418b88f328a425b0fbd67aaa0f48acf6347903747cd681c8c4abd4fcb04f0748de2402a3da9ba59f6da9154e07b17e0c1154f890c01060000004cf0b4ec852a062a2cc599ac764a5bd6d6afb469326cfbf736ef0399d987de06ddf842c42f03c479d683aa7c511eb5ccc8f038192ff1fb8a1e1f234446efcb8c0c01070000004aad3d5f2f24ff187a06db9b87a2affa8ac0e219b02512dfd81549bccf62b8111e9961befb215d7e05d7f9ae14f2726098dd95f971e01daac6e720fde387f08c0c01080000004263728475b9e1e48f313e9e4d10cd9a1938d3bd3a80d48ed1418657d303643fac70ba80a6cdad37746e119eb280b668455d297e3b9113edc80ac6762536d18d0c01090000002a9aef1f1b91400ba0d52126b535393ffbfa60c531265efe2bf3dee09aa6f5453a14f3d94bc7f34f3b3d931d46f4726bc87e117a7baa93dd9bb1ac8a9d41da8e0c010a0000000606a0fbaf5f6ba57bc8276aa3e4ad587a1103563d8ea806ded26c8c751fa51dc3e69b4c1f016446cc711be1172a687ee2a5c52f2cb25b9255bf95c4b0fb57810c010b00000054ef1c2e69dd06552fef2b085c4afe5ea1bede23cb912aefb3eff3a3be820f3f1852e54001c15ba05d8b038e2b72207d94c94f8550b0f02fd8c7ee44bcf4df840c010c000000a6b9492a5bc58a4ca6f975c1c50f0c8f09f6b4e842a04a767867c11cc4910f1528efe6a26579981a7d82bda20cdc346fa805703d0aff9988a61ae63443a5008a0c010d000000ca13f5d6336bf7b54351c831d51c3d3aa37c9c1b806ed5715bb72c6a3206d25c893772dac586a3b3e3a616c637947d86119aa54ee4599c4dad23d4ff2f4e7b860c010e0000000e756428f4382e4e540c46a0ea60fdb094129e0d0be80fb7494498d7e714d832c57b158de81ca0f0df736f89035e7eb7449b609adf5c80ef751fb205acac0a800c010f000000820323f879134b5ca5bdcfa4f45a6089c1f5f198633ca8f841349e0e32a6032d20612506730aa5cf0141d56e156234540548164cf1cb6837d767a14d9d5dc18b04ea03000040a65a5bad9b346fc4d86fe33eadfa0cd49d0378910b0af5e73159ed503611e3468b0e01113f2c78d4b7b20efd4bdc38627c60bb4b840c8e7476e891b505d828eb016dba1703a583c3e64688bdf97e2bc29a8b6cb499a432d237a323b1412e90e2f769655c4651e3d298ecdc2a536cb89a3a3f31a0f6441a3ae4aa51c4741d32d1cd99206627a51c448c1dbf1f50009655c98b0c1326ae38420d9515cbbcf93ec8084285c48449903c919b164453fde3c5412badebcee765a01aa1b7a8fd503519283259d37c98ecab006af925b013b57a27fe56800cda5f5feec980cb110f87030f7913bfad164bd2dcff5556e48cf06bc28bb3f1ead5200be27d95adbcac624e714e61e9b3100565ae524c326102f8bc17bd3e9e73a185b8116a9e964e665c000000e902d1eb45830486f22485d977ff49349529742a039db30cb9da5fda419c6302bfa8b6ea0b005ff08f7e128ee3e8d263926c4f682842d2eb319b09f251540a815836db74761490734e7a9fd6e677f451d30e6fe142c52655c15db683a4931659c75ac85fbf6c080661757261204411530800000000056175726101015e53dd83b63d9a65ee2906fcbeb0719f9a7a7281513441e211f448efcdfd472a80fe4cb669fa22b8499961bea7643c3586fd16d26e220cb7e5a9da8a1d07ce8b00000000d66cde001401b0b5c1ee601faedc6101d132c793fcdae39737468c9c3dcc3c57fb5b4e2f14648b002685a67f4aeca1fef16795c2c2c352b5afd91a8fe346b7f278f2fce67b8602d23eb4ca1a20f459e1557a1ccc8976dcc686873f3499262e8cd38d0e6294a4340c1fa00bc26ec9a18df5b1b14d4b90bd3c75c5d35b6dff8eeca23976de2f288301d4fa4f3d94cb63c38ea719e64ae09cd7d6eb409d8d1bc21f1a00377fb30e4118e89dd2b4fba1a538819f4dc57e5cf76e80d953ef4c04ad81918b4d68b050db8f021aba632739c7993e789b4ae276b6c933e82a644d5442443224e9fb12d499e12a41a8b4deeb4ebe503842cf8cb680e84e0e4e6898bf62a3cf722262f654cec086029c38b051968569013dc5c2fbc93db0c4bb86a3f548dc5ddff363f32fbe7704682a5dabb9e9689ba820a5f68ac31a92613e0f760d4862c41f5d97edb22e378d8a141f007d63733ead4d8c05045abe1f10fca05ca02aa44c5aafb59135a841c09afeb3965ab37903edd08c8c9453f56727a1b307d7458eb192b42830b03ef1314be104b4a6aefd9b5f6028c17d04a84ec8811461e52a44e489e67b01bfd421b1e25e002200ab433e080642414245b501030d0000008922a610000000002e1e6cadbc719c66a3a4193c2b577cce37eec5e041213ba010823a9b988da947f942254709d3786c37eebb54253f639584dd149883df229258a3ae02ee81860b5945c6ba5fb7990a0fb7ec0ff19c9d5ea8b7d613cfe7de3a8bcb4bdb8f6fed0f05424142450101266ba8cf6a2b5b86a3a1bda16be242d75b042a8dea21e57e0c265cdbc942612c9531ff2178a02871336d6e326d2a71bbb749c3dbc7e3a82ea1c2c0690a73ca8a")
	require.NoError(t, err)

	blockResponse := new(network.BlockResponseMessage)
	err = blockResponse.Decode(blockResponseBytes)
	require.NoError(t, err)

	blockData := blockResponse.BlockData[0]

	block := &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	fmt.Printf("Executing block number %d\n", block.Header.Number)
	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)

	fmt.Printf("Storing the trie to the disk, regards block %d\n", block.Header.Number)

	newTrieState := storage.NewTrieState(trieState.Trie().DeepCopy())
	err = storageState.StoreTrie(newTrieState, &block.Header)
	require.NoError(t, err)

	codeHash, err = trieState.LoadCodeHash()
	require.NoError(t, err)

	fmt.Printf("code hash from updated state: %s\n", codeHash.String())

	// wnd14576855StateTrie := newTrieFromRPC(t, "../test_data/14576855trie_state_data.json")
	// expectedStorageRootHash := common.MustHexToHash("0xe8c4636bd5f01d9f9a18fa96949787975c9c3b78a0624ae95e3078da20c33605")
	// require.Equal(t, expectedStorageRootHash, trie.V0.MustHash(*wnd14576855StateTrie))

	// state := storage.NewTrieState(wnd14576855StateTrie)
	instance, err = wazero_runtime.NewInstanceFromTrie(trieState.Trie(), wazero_runtime.Config{
		Storage: trieState,
		LogLvl:  log.Critical,
	})
	require.NoError(t, err)
	// instance.version()
	// instance.SetContextStorage(state)

	blockResponseBytes, err = common.HexToBytes("0x0ade0d0a20396e794b25deac6c25bc8b5e7b00892934fea53c87bb7128983c7921d72f758112a0020283b22f28a02825206b5cfd1668e117c4c1d3b49ec02a5b4731aff2c3ac1c4062b37903d1b4e4621243629a494fc62b91c92f2d8c378e2c3a0be552af2a05dc3d0dbd7b7ad6e998c94d604101a7e147cf401b041e0ce4996f609b3f5601c77c4922a529080642414245b50103000000008b22a6100000000054741a1a0fbb59012ce3b6c28a9e37487a49113964713cb85485ac24b0b46a5d18785ae8cca9a49ebc11657be3774bca3f93d8326af163290479a77be944370c6b3b763783d6bc086da473b2438389e0d6ae31abbc4a37441b27a6516e57920f05424142450101e2629d073bca188b537fd4715da7f0d3c0be8474a01935f7c25e1c5f374bde742e99423d97698db3975ebbd40819ea352f6d1d1f510dc761c9a623cf3a8975831a0b280402000bd099c93586011a890b1d16042d00400c040000000076b8399ace2a50d5b6fd5cbf464b048178c6fffa047702a416f3d8c8d264ad4a41a4853dc73e37dd91dfaec5d0d41dc01a542a13232cfc56b9372ffcbb83ef840c0401000000d2f839d7eaa667fc2e48a16d98807290e7c10a5ecd2583f8568fcc1bb7d5351e8656146d1365c8619f3202e966877e0f5870c90ecf910ef03f0f1a3dd87266830c0402000000fa36f6a59008ff66a4229c482e8a50315fd5251fc53cd9469b6543bc4930cd2528e827a4fcef270dc9722d37ae23ac04953c4fb08725e1e6bd5615babc42dd800c04030000009497e81b8089a8189918c58c2d5f80690d07f3e0353801e9378efc5a76912b41a8d34b1dd0d93b897441dff3b8c68544bceb9cf27c6e0c83be911ba6eb9574870c04040000006adb55ddc67509b0ddff38ce482fb437d96b0faae302e2b7cf5d5979f5ec45655fafc6045104d9b0d234f42c1ed8ec35c88fe420c8ea6123b780112cb92fa4890c0405000000b80a4c5a180e71a741110b020bacbffba9082d9b76c3e8ce79714be476f144626d6757c6e0efb878754e692c8be83bc2deacd5582f96ceb4f91d68b3e0acaa8a0c0406000000ce8b4daede74064db15bc9a924bc046bde7c9cce04cae4e543266aa691225f2dfc41119900287c9fa43965f82e0e48cc54fc31bf3a06e38a29730b996931e88c0c0407000000b68e00e032c1316faf1415e81501158eba96906b35281f1ff9862dc0b92881430c8d3ae74f16054ab24b313dd3a6e26ca0e57b5206f1a0ee46deb82946a231820c0408000000902e9429fb62ea6de61b0a0da1aae498fe28b27321540848d4ef86c52d684416d3262c3f03826a92dd26eedb5b1ccc220429277822ac0531350c31e43aae90830c04090000007cb92e3ce7438b34d56f5af43af0363520ebd4810c1b8ea8fe717ff1bbcea404528a1765ae3602d3975347ea17554d34613969cdc69718e7fdc47ea7cdc159800c040a0000005ecb6f76ac28e9c9596b46f2571a8380ad0742f7dfae673544b71e7000dbd0039cd13e10fe3e5d7ae14614c5f100f2c72373ea4d2df1f6a34be831db81f62a8a0c040b0000009ca60150ef177cb3168b28470364b611825ad2d86e3e38056b79903d4639205c2799926328d205c4cc66e0f2ca832e6ff8fa4ff9f05f1b57103d63ab465190860c040c00000020aa8c3c211ca39b7d1ae912ef2045064742eb1cb04e073bdb87814386f32a40547f8ba9029d3b830052011273dd6f6bcafdad40efd031bed27fca87747bdf8d0c040d0000006e7204ad9b80e56429c17d0de2aa231dd9b933bb7469bcd124d44aa9c0406b71bedb4a7037bebf8605afb3cf904ed60f359335d9ce2fe3884129f81f4f2e0e850c040e000000a869927414a693a1ff6794fcd975a737434556b1b6970c841657aa831bb1dc1910d9ce3c27d46e59cf4355077ce2b1e76964e0ba5907f2a57a522944e6c73b8b0c040f000000b858993e3b38f0cc7a70b226a40b9a49b11d75ddd0fbf7158d66127830ddca78f50277fe22d6f5b13bf98b82fedaca5fcee11868bbbe3afe2f25ff7d57584583000040a65a5bad9b346fc4d86fe33eadfa0cd49d0378910b0af5e73159ed503611e35eb37903e8c4636bd5f01d9f9a18fa96949787975c9c3b78a0624ae95e3078da20c33605cf34d6de3155b72a0818608103915619964369e8599482e246a4cdf43df1a7500c0642414245b501030f0000008a22a61000000000821ded3b100e3f68e3ae8109c028e91b0d084bfa6458d5a28203e35b2d01e87bf4f8ef774929e4522b451e3abf892dbbb5aaf21ba7870db5e0efa8d481268e0159463ac6be5114cb953e7bb5d070e785b3ace95680bc6040bab4aed1575bed0708054241424501014ea332ce7e94687b6103adaf88e0fd370879926e95834ca700852fdaff0e260857572d8c7798150b0ffdf60d85d90fc6c6bf10c974606307245d4d4683b64e8e")
	require.NoError(t, err)

	blockResponse = new(network.BlockResponseMessage)
	err = blockResponse.Decode(blockResponseBytes)
	require.NoError(t, err)

	blockData = blockResponse.BlockData[0]

	block = &types.Block{
		Header: *blockData.Header,
		Body:   *blockData.Body,
	}

	fmt.Printf("Executing block number %d\n", block.Header.Number)

	_, err = instance.ExecuteBlock(block)
	require.NoError(t, err)
}

func newTrieFromRPC(t *testing.T, filename string) *trie.Trie {
	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	encodedTrieEntries := make([]string, 0)
	err = json.Unmarshal(data, &encodedTrieEntries)
	require.NoError(t, err)

	entries := make(map[string]string, len(encodedTrieEntries))
	for _, encodedEntry := range encodedTrieEntries {
		bytesEncodedEntry := common.MustHexToBytes(encodedEntry)
		entry := trie.Entry{}
		err := scale.Unmarshal(bytesEncodedEntry, &entry)
		require.NoError(t, err)

		entries[common.BytesToHex(entry.Key)] = common.BytesToHex(entry.Value)
	}

	tr, err := trie.LoadFromMap(entries)
	require.NoError(t, err)
	return &tr
}
