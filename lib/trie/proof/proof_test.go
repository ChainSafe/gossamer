// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/db"
	"github.com/stretchr/testify/require"
)

func Test_Generate_Verify(t *testing.T) {
	t.Parallel()
	stateVersion := trie.V0

	keys := []string{
		"cat",
		"catapulta",
		"catapora",
		"dog",
		"doguinho",
	}

	trie := trie.NewEmptyTrie()

	for i, key := range keys {
		value := fmt.Sprintf("%x-%d", key, i)
		trie.Put([]byte(key), []byte(value), stateVersion)
	}

	rootHash, err := trie.Hash()
	require.NoError(t, err)

	db, err := database.NewPebble("", true)
	require.NoError(t, err)
	err = trie.WriteDirty(db)
	require.NoError(t, err)

	for i, key := range keys {
		fullKeys := [][]byte{[]byte(key)}
		proof, err := Generate(rootHash.ToBytes(), fullKeys, db)
		require.NoError(t, err)

		expectedValue := fmt.Sprintf("%x-%d", key, i)
		err = Verify(proof, rootHash.ToBytes(), []byte(key), []byte(expectedValue))
		require.NoError(t, err)
	}
}

func TestParachainHeaderStateProof(t *testing.T) {
	stateRoot, err := hex.DecodeString("3b903e9947f26c4455f213b648661d0ef9b30018da7fa7be76bb5af2f5f75735")
	require.NoError(t, err)

	encodeStorageKey, err := hex.DecodeString("cd710b30bd2eab0352ddcc26417aa1941b3c252fcb29d88eff4f3de5de4476c3b6ff6f7d467b87a9e8030000") //nolint:lll
	require.NoError(t, err)

	proof1, err := hex.DecodeString("36ff6f7d467b87a9e803000021590f48b11891aee1f281f856256f37a20f8abc5d027434f89dd2decab922fe") //nolint:lll
	require.NoError(t, err)

	proof2, err := hex.DecodeString("800464801861be085002d2b0498ea992b13cfb1ca6b5e05a7ca54f6180dcc1bcd10a9f0680f6f6801e4b41e2e6d8ec194dba122bfb9eb33feb2545ef5144cea79551f7cc5280f6370779a48f025599265f348f955ee0b12eeb99238950c07f5562091f2186d48043e819b824d89dc6e744b5342c963829d44a93a1bdad2405615856f67945c9e0") //nolint:lll
	require.NoError(t, err)

	proof3, err := hex.DecodeString("80cf93807b212eaf64882b542230cc1fa87d9505181a516c0dfd67ac55d3158cd753f8ad80862c9aecf51563f0b4f54f6d2a325bec9afdf62a66f595e150203fd9a144b1e580c2fb34bc8b88011ab509fd52c25b3469bbc9353f472a05decd83449af1e3677d80ecbe9453c51b405848014efabda8a0cde4b9458e7c26a4d9eeb589c52bdb5eb1809c43b10cb7509edfd059982f30f20ba7368bbd82786184cf0a5be813cd07490a8088d755e63972295bd4772b7322e27adb3358090fc2f16c66e65341de0d9bd22980891eac33e3ee82a64283ab12370710911e866576869040634657bcc78a1385a180a4c9385e359a9977574174c4d31beab6206a569ad15ef435bf784f16623e1d21802e89324e6a5e0be929b37bb44bdbf6619e6af80cbdebc9bd67a44c8aa072ef3980d8eabbfa85a6309a2ff6dac1a06e6a6d214faba34887e6f8e14c0d0d1858711e") //nolint:lll
	require.NoError(t, err)

	proof4, err := hex.DecodeString("80ffff80fe86a6cb12b2233729f7834ff614d56c968207d9ae09266cd3835e32fc7bbdba80ee3fa56aef90d79d5a7c8e5f6e85252288631533a5a8ebf405846bdc3cedcaf38019c7f105c5c4278d8f4b5a67adb644c0d4b056b6affbe8721df8ce54865e8fe8800b223e5d94298635855f517e49a2e925d4e39de3e27ff1af06b658de5a2e8280804165dde7158903211dc880ebc441e6fc3ef9f8a1c5f99fd261178c4e97206805808dcd7a042792b39ad7fcbd97e77273b3d0b250c3203398f7290a7d3e0d7cc20c807c3fac76e1315865f8e8fcda6748711a415fc87187794acba9584b2c151b956080091b6db629efcf3857b17a2df2fa8b296c5aedc80db088f4e6a560053c7ce890803e5026745b944d7fa9a39c6f08292b88efbaec1e1041ccd78348053881c1bf86800451959b47e46ecdb0edd2df37445db0a629898058bae12a73ad88379a130fe080f52d9d5dfa99ec1762f86ca9b229e11e8f7910633e1032ece9c89e28892397a4805e6def858a456048697cbdb9af83fc447c671b0cf283f1409400f3a6c506321f80f8093e29566bd8ebec39521f87c10156d1c424a767aadecd231adf5c55f5f538806075e1c36fcba711bb56da63b85b31fdf481041a0acb93b035a6ebb9987734f4808f0cfeb4e4785d0fcb162883963e55e8cbc49c2398f1f275cfe5d2484bac2f9780b063dd4a5cd6b883c54ff93751d114e20e99e2e05eff766ddcca317b67d4f08a") //nolint:lll
	require.NoError(t, err)

	proof5, err := hex.DecodeString("9e710b30bd2eab0352ddcc26417aa1945fcb801998fc2315e4329c3d3c59ff787fef52f1707abcf997f8114a016594b6716ce8803a5b05f6d48162e04748dce0050d00025c0d51a4845ea2119f66952522b2cd2b80549fd5090d980b3ae9b1b61196d5f617c57b2f4e5eb5f2e51e4c5c857429363180196a38280fc3af7f724552363e4833e604127b44cb46271dd28151765bb91cf0505f0e7b9012096b41c4eb3aaf947f6ea4290800004c5f03c716fb8fff3de61a883bb76adb34a2040080ae1c868ee54941861f121640db72e895211b6748da302cc4ddf39f715c76e7528052e248e38ba2e7f604c09c090bfb8abc6bc68c2f92f34f454606b4a63102e58a8026a2dd112b0ca67351d4abb723dc41978d5865dd4b208b37eed5200bbc0ba0f4807bd51e23ee41e85d99c3985aa8b0f859f70f20fa783b7055b5161adfc69e2d5180a6a96fae992961a174ea36d7e23e69c08d45ecacc82e14fbc3546b8d60ceae48") //nolint:lll
	require.NoError(t, err)

	proof6, err := hex.DecodeString("9f0b3c252fcb29d88eff4f3de5de4476c3ffff8076bed1a9045e1937ab7ad7cff6e667c66351022b28103771310fa09e0a07708f808ff91cb4e274aa25177bbea2d77d5693f3da34820ecb82d6a06529de8bc0beb580b51c90d98a3cc501566107ce9b89e91609de184f72c521efe2e2486beb095dc280af5427f678c5055f4039369c53aaa785a3767ba10cf2e42b5cc9b625d8021bca8044ea5b04397b504579d34a01419b6f0fb0c4f3003b3e6e0b99687cc88f398670803d46b9972edc81cd44df296d227eafde0abd880a53ea37632ddb558e913315e68010a7cfabb7bf234b6efd0fba3d30758e762ec52d14d329e0b9ebd5c84ec7752680c9b8e0f77483284d53f3ccb7ca3a217faa9b50a819cfa557438cfa9813306910809bb471046c73d5edf5683b4e3408714f428ecdf8c447e80f8335b4049e555c2e80fd2a0bea95ba513ddd672bdd9d9fbfd1c9588731d06e9afa5004332250054ea180263061f7d953b0fba1d98b9c6529ce6c1d78af0012180caccb388c4921216de1803543b7e854863de08e6ce77ac171ecec6b64d419e58d6171fe654ee279b5f8c28061cd0a2e641fbce3fd78bad7f2b298918a187aca625491c1a1898763705840fa807aa5071686a8d5d83f8db3531aaaa181ea3843746bfc7917193b1dfcfbb0c49b8065ad311a5eb95c25f400fd199f1005a4ba6f62a7049117e9466dba91c1df949d80aa704996ec32908132b67245030b4d8456c46415837150ef58898df6b9b0ce5e") //nolint:lll
	require.NoError(t, err)

	proof7, err := hex.DecodeString("e902116a2811eaaa372fcd8c769b5f433d3995872b21c468dcfc6270e1f9fa07167eaa4c7c00f5f981c0b4dafe3c1029e70fb290294fc21f040197ac00f209dbf659a97bd83f7dc3fc42e985905ea2313b2551b72692510e9744493bd525055e27e295948a110806617572612092d55c08000000000561757261010116fd4fedb8ecd8eba0d907b7bd534b260bc0b86a0e9a1fd8f18cb85e9073f442a6a9f5460bfb2443bce67b8fdba17bbd2927bdad8fc6ae021c03e2c8b3e33e89") //nolint:lll
	require.NoError(t, err)

	proof := [][]byte{proof1, proof2, proof3, proof4, proof5, proof6, proof7}

	expectedValue := proof7

	proofDB, err := db.NewInMemoryDBFromProof(proof)
	require.NoError(t, err)

	trie, err := buildTrie(proof, stateRoot, proofDB)
	require.NoError(t, err)
	value := trie.Get(encodeStorageKey)
	require.Equal(t, expectedValue, value)

	// Also check that we can verify the proof
	err = Verify(proof, stateRoot, encodeStorageKey, expectedValue)

	require.NoError(t, err)
}

func TestTrieProof(t *testing.T) {
	key, err := hex.DecodeString("f0c365c3cf59d671eb72da0e7a4113c49f1f0515f462cdcf84e0f1d6045dfcbb")
	if err != nil {
		panic(err)
	}
	root, err := hex.DecodeString("dc4887669c2a6b3462e9557aa3105a66a02b6ec3b21784613de78c95dc3cbbe0")
	if err != nil {
		panic(err)
	}
	proof1, err := hex.DecodeString("80fffd8028b54b9a0a90d41b7941c43e6a0597d5914e3b62bdcb244851b9fc806c28ea2480d5ba6d50586692888b0c2f5b3c3fc345eb3a2405996f025ed37982ca396f5ed580bd281c12f20f06077bffd56b2f8b6431ee6c9fd11fed9c22db86cea849aeff2280afa1e1b5ce72ea1675e5e69be85e98fbfb660691a76fee9229f758a75315f2bc80aafc60caa3519d4b861e6b8da226266a15060e2071bba4184e194da61dfb208e809d3f6ae8f655009551de95ae1ef863f6771522fd5c0475a50ff53c5c8169b5888024a760a8f6c27928ae9e2fed9968bc5f6e17c3ae647398d8a615e5b2bb4b425f8085a0da830399f25fca4b653de654ffd3c92be39f3ae4f54e7c504961b5bd00cf80c2d44d371e5fc1f50227d7491ad65ad049630361cefb4ab1844831237609f08380c644938921d14ae611f3a90991af8b7f5bdb8fa361ee2c646c849bca90f491e6806e729ad43a591cd1321762582782bbe4ed193c6f583ec76013126f7f786e376280509bb016f2887d12137e73d26d7ddcd7f9c8ff458147cb9d309494655fe68de180009f8697d760fbe020564b07f407e6aad58ba9451b3d2d88b3ee03e12db7c47480952dcc0804e1120508a1753f1de4aa5b7481026a3320df8b48e918f0cecbaed3803360bf948fddc403d345064082e8393d7a1aad7a19081f6d02d94358f242b86c") //nolint:lll
	if err != nil {
		panic(err)
	}
	proof2, err := hex.DecodeString("9ec365c3cf59d671eb72da0e7a4113c41002505f0e7b9012096b41c4eb3aaf947f6ea429080000685f0f1f0515f462cdcf84e0f1d6045dfcbb20865c4a2b7f010000") //nolint:lll
	if err != nil {
		panic(err)
	}
	proof3, err := hex.DecodeString("8005088076c66e2871b4fe037d112ebffb3bfc8bd83a4ec26047f58ee2df7be4e9ebe3d680c1638f702aaa71e4b78cc8538ecae03e827bb494cc54279606b201ec071a5e24806d2a1e6d5236e1e13c5a5c84831f5f5383f97eba32df6f9faf80e32cf2f129bc") //nolint:lll
	if err != nil {
		panic(err)
	}

	proof := [][]byte{proof1, proof2, proof3}
	proofDB, err := db.NewInMemoryDBFromProof(proof)

	require.NoError(t, err)

	trie, err := buildTrie(proof, root, proofDB)
	require.NoError(t, err)
	value := trie.Get(key)

	require.Equal(t, []byte{0x86, 0x5c, 0x4a, 0x2b, 0x7f, 0x1, 0x0, 0x0}, value)
}
