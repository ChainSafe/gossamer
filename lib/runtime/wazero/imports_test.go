package wazero_runtime

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/proof"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

// NewTestInstance will create a new runtime instance using the given target runtime
func NewTestInstance(t *testing.T, targetRuntime string) *Instance {
	t.Helper()
	return NewTestInstanceWithTrie(t, targetRuntime, nil)
}

func setupConfig(t *testing.T, ctrl *gomock.Controller, tt *trie.Trie, lvl log.Level,
	role common.NetworkRole, targetRuntime string) Config {
	t.Helper()

	s := storage.NewTrieState(tt)

	ns := runtime.NodeStorage{
		LocalStorage:      runtime.NewInMemoryDB(t),
		PersistentStorage: runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
		BaseDB:            runtime.NewInMemoryDB(t), // we're using a local storage here since this is a test runtime
	}

	// version := (*runtime.Version)(nil)
	// if targetRuntime == runtime.HOST_API_TEST_RUNTIME {
	// 	// Force state version to 0 since the host api test runtime
	// 	// does not implement the Core_version call so we cannot get the
	// 	// state version from it.
	// 	version = &runtime.Version{}
	// }

	return Config{
		Storage:     s,
		Keystore:    keystore.NewGlobalKeystore(),
		LogLvl:      lvl,
		NodeStorage: ns,
		Network:     new(runtime.TestRuntimeNetwork),
		Transaction: mocks.NewMockTransactionState(ctrl),
		Role:        role,
		// testVersion: version,
	}
}

// DefaultTestLogLvl is the log level used for test runtime instances
var DefaultTestLogLvl = log.Info

// NewTestInstanceWithTrie returns an instance based on the target runtime string specified,
// which can be a file path or a constant from the constants defined in `lib/runtime/constants.go`.
// The instance uses the trie given as argument for its storage.
func NewTestInstanceWithTrie(t *testing.T, targetRuntime string, tt *trie.Trie) *Instance {
	t.Helper()

	ctrl := gomock.NewController(t)

	cfg := setupConfig(t, ctrl, tt, DefaultTestLogLvl, common.NoNetworkRole, targetRuntime)
	targetRuntime, err := runtime.GetRuntime(context.Background(), targetRuntime)
	require.NoError(t, err)

	r, err := NewInstanceFromFile(targetRuntime, cfg)
	require.NoError(t, err)

	return r
}

func decompressWasm(code []byte) ([]byte, error) {
	compressionFlag := []byte{82, 188, 83, 118, 70, 219, 142, 5}
	if !bytes.HasPrefix(code, compressionFlag) {
		return code, nil
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("creating zstd reader: %s", err)
	}
	bytes, err := decoder.DecodeAll(code[len(compressionFlag):], nil)
	if err != nil {
		return nil, err
	}
	return bytes, err
}

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg Config) (*Instance, error) {
	fmt.Println(fp)

	// Reads the WebAssembly module as bytes.
	// Retrieve WASM binary
	bytes, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("Failed to read wasm file: %s", err)
	}

	if strings.Contains(fp, "compact") {
		var err error
		bytes, err = decompressWasm(bytes)
		if err != nil {
			return nil, err
		}
	}

	return NewInstance(bytes, cfg)
}

func Test_ext_crypto_ed25519_generate_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	mnemonic := "vessel track notable smile sign cloth problem unfair join orange snack fly"

	mnemonicBytes := []byte(mnemonic)
	var data = &mnemonicBytes
	seedData, err := scale.Marshal(data)
	require.NoError(t, err)

	params := append(idData, seedData...)

	pubKeyBytes, err := inst.Exec("rtm_ext_crypto_ed25519_generate_version_1", params)
	require.NoError(t, err)
	require.Equal(t,
		[]byte{128, 218, 27, 3, 63, 174, 140, 212, 114, 255, 156, 37, 221, 158, 30, 75, 187,
			49, 167, 79, 249, 228, 195, 86, 15, 10, 167, 37, 36, 126, 82, 126, 225},
		pubKeyBytes,
	)

	// this is SCALE encoded, but it should just be a 32 byte buffer. may be due to way test runtime is written.
	pubKey, err := ed25519.NewPublicKey(pubKeyBytes[1:])
	require.NoError(t, err)

	require.Equal(t, 1, ks.Size())
	kp := ks.GetKeypair(pubKey)
	require.NotNil(t, kp)
}

func Test_ext_crypto_ed25519_public_keys_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	idData := []byte(keystore.DumyName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	size := 5
	pubKeys := make([][32]byte, size)
	for i := range pubKeys {
		kp, err := ed25519.GenerateKeypair()
		require.NoError(t, err)

		ks.Insert(kp)
		copy(pubKeys[i][:], kp.Public().Encode())
	}

	sort.Slice(pubKeys, func(i int, j int) bool {
		return bytes.Compare(pubKeys[i][:], pubKeys[j][:]) < 0
	})

	res, err := inst.Exec("rtm_ext_crypto_ed25519_public_keys_version_1", idData)
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(res, &out)
	require.NoError(t, err)

	var ret [][32]byte
	err = scale.Unmarshal(out, &ret)
	require.NoError(t, err)

	sort.Slice(ret, func(i int, j int) bool {
		return bytes.Compare(ret[i][:], ret[j][:]) < 0
	})

	require.Equal(t, pubKeys, ret)
}

func Test_ext_crypto_ed25519_sign_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	ks.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_crypto_ed25519_sign_version_1", append(append(idData, encPubKey...), encMsg...))
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(res, &out)
	require.NoError(t, err)

	var val *[64]byte
	err = scale.Unmarshal(out, &val)
	require.NoError(t, err)
	require.NotNil(t, val)

	value := make([]byte, 64)
	copy(value[:], val[:])

	ok, err := kp.Public().Verify(msgData, value)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_ext_crypto_ed25519_verify_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	ks.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	sign, err := kp.Private().Sign(msgData)
	require.NoError(t, err)
	encSign, err := scale.Marshal(sign)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_ed25519_verify_version_1", append(append(encSign, encMsg...), encPubKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
}

func Test_ext_crypto_secp256k1_ecdsa_recover_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	msgData := []byte("Hello world!")
	blakeHash, err := common.Blake2bHash(msgData)
	require.NoError(t, err)

	kp, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)

	sigData, err := kp.Private().Sign(blakeHash.ToBytes())
	require.NoError(t, err)

	expectedPubKey := kp.Public().Encode()

	encSign, err := scale.Marshal(sigData)
	require.NoError(t, err)
	encMsg, err := scale.Marshal(blakeHash.ToBytes())
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_secp256k1_ecdsa_recover_version_1", append(encSign, encMsg...))
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(ret, &out)
	require.NoError(t, err)

	result := scale.NewResult([64]byte{}, nil)

	err = scale.Unmarshal(out, &result)
	require.NoError(t, err)

	rawPub, err := result.Unwrap()
	require.NoError(t, err)
	rawPubBytes := rawPub.([64]byte)
	require.Equal(t, 64, len(rawPubBytes))

	publicKey := new(secp256k1.PublicKey)

	// Generates [33]byte compressed key from uncompressed [65]byte public key.
	err = publicKey.UnmarshalPubkey(append([]byte{4}, rawPubBytes[:]...))
	require.NoError(t, err)
	require.Equal(t, expectedPubKey, publicKey.Encode())
}

func Test_ext_crypto_ecdsa_verify_version_2(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	msgHash, err := common.Blake2bHash(msgData)
	require.NoError(t, err)

	sig, err := kp.Private().Sign(msgHash[:])
	require.NoError(t, err)

	encSig, err := scale.Marshal(sig)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_ecdsa_verify_version_2", append(append(encSig, encMsg...), encPubKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
}

func Test_ext_crypto_secp256k1_ecdsa_recover_compressed_version_1(t *testing.T) {
	// TODO: fix this
	t.Skip("host API tester does not yet contain rtm_ext_crypto_secp256k1_ecdsa_recover_compressed_version_1")
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	msgData := []byte("Hello world!")
	blakeHash, err := common.Blake2bHash(msgData)
	require.NoError(t, err)

	kp, err := secp256k1.GenerateKeypair()
	require.NoError(t, err)

	sigData, err := kp.Private().Sign(blakeHash.ToBytes())
	require.NoError(t, err)

	expectedPubKey := kp.Public().Encode()

	encSign, err := scale.Marshal(sigData)
	require.NoError(t, err)
	encMsg, err := scale.Marshal(blakeHash.ToBytes())
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_secp256k1_ecdsa_recover_compressed_version_1", append(encSign, encMsg...))
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(ret, &out)
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.Write(out)

	uncomPubKey, err := new(types.Result).Decode(buf)
	require.NoError(t, err)
	rawPub := uncomPubKey.Value()
	require.Equal(t, 33, len(rawPub))

	publicKey := new(secp256k1.PublicKey)

	err = publicKey.Decode(rawPub)
	require.NoError(t, err)
	require.Equal(t, expectedPubKey, publicKey.Encode())
}

func Test_ext_crypto_sr25519_generate_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	mnemonic, err := crypto.NewBIP39Mnemonic()
	require.NoError(t, err)

	mnemonicBytes := []byte(mnemonic)
	var data = &mnemonicBytes
	seedData, err := scale.Marshal(data)
	require.NoError(t, err)

	params := append(idData, seedData...)

	ret, err := inst.Exec("rtm_ext_crypto_sr25519_generate_version_1", params)
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(ret, &out)
	require.NoError(t, err)

	pubKey, err := ed25519.NewPublicKey(out)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Size())

	kp := ks.GetKeypair(pubKey)
	require.NotNil(t, kp)
}

func Test_ext_crypto_sr25519_public_keys_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	idData := []byte(keystore.DumyName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	const size = 5
	pubKeys := make([][32]byte, size)
	for i := range pubKeys {
		kp, err := sr25519.GenerateKeypair()
		require.NoError(t, err)

		ks.Insert(kp)
		copy(pubKeys[i][:], kp.Public().Encode())
	}

	sort.Slice(pubKeys, func(i int, j int) bool {
		return bytes.Compare(pubKeys[i][:], pubKeys[j][:]) < 0
	})

	res, err := inst.Exec("rtm_ext_crypto_sr25519_public_keys_version_1", idData)
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(res, &out)
	require.NoError(t, err)

	var ret [][32]byte
	err = scale.Unmarshal(out, &ret)
	require.NoError(t, err)

	sort.Slice(ret, func(i int, j int) bool {
		return bytes.Compare(ret[i][:], ret[j][:]) < 0
	})

	require.Equal(t, pubKeys, ret)
}

func Test_ext_crypto_sr25519_sign_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	ks.Insert(kp)

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_crypto_sr25519_sign_version_1", append(append(idData, encPubKey...), encMsg...))
	require.NoError(t, err)

	var out []byte
	err = scale.Unmarshal(res, &out)
	require.NoError(t, err)

	var val *[64]byte
	err = scale.Unmarshal(out, &val)
	require.NoError(t, err)
	require.NotNil(t, val)

	value := make([]byte, 64)
	copy(value[:], val[:])

	ok, err := kp.Public().Verify(msgData, value)
	require.NoError(t, err)
	require.True(t, ok)
}

func Test_ext_crypto_sr25519_verify_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	sign, err := kp.Private().Sign(msgData)
	require.NoError(t, err)
	encSign, err := scale.Marshal(sign)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_sr25519_verify_version_1", append(append(encSign, encMsg...), encPubKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
}

func Test_ext_crypto_sr25519_verify_version_2(t *testing.T) {
	// TODO: add to test runtime since this is required for Westend
	t.Skip("host API tester does not yet contain rtm_ext_crypto_sr25519_verify_version_2")
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	kp, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	idData := []byte(keystore.AccoName)
	ks, _ := inst.Context.Keystore.GetKeystore(idData)
	require.Equal(t, 0, ks.Size())

	pubKeyData := kp.Public().Encode()
	encPubKey, err := scale.Marshal(pubKeyData)
	require.NoError(t, err)

	msgData := []byte("Hello world!")
	encMsg, err := scale.Marshal(msgData)
	require.NoError(t, err)

	sign, err := kp.Private().Sign(msgData)
	require.NoError(t, err)
	encSign, err := scale.Marshal(sign)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_crypto_sr25519_verify_version_1", append(append(encSign, encMsg...), encPubKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
}

func Test_ext_trie_blake2_256_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testinput := []string{"noot", "was", "here", "??"}
	encInput, err := scale.Marshal(testinput)
	require.NoError(t, err)
	encInput[0] = encInput[0] >> 1

	res, err := inst.Exec("rtm_ext_trie_blake2_256_root_version_1", encInput)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(res, &hash)
	require.NoError(t, err)

	tt := trie.NewEmptyTrie()
	tt.Put([]byte("noot"), []byte("was"))
	tt.Put([]byte("here"), []byte("??"))

	expected := tt.MustHash()
	require.Equal(t, expected[:], hash)
}

func Test_ext_trie_blake2_256_ordered_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testvalues := []string{"static", "even-keeled", "Future-proofed"}
	encValues, err := scale.Marshal(testvalues)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_trie_blake2_256_ordered_root_version_1", encValues)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(res, &hash)
	require.NoError(t, err)

	expected := common.MustHexToHash("0xd847b86d0219a384d11458e829e9f4f4cce7e3cc2e6dcd0e8a6ad6f12c64a737")
	require.Equal(t, expected[:], hash)
}

func Test_ext_trie_blake2_256_verify_proof_version_1(t *testing.T) {
	tmp := t.TempDir()

	memdb, err := chaindb.NewBadgerDB(&chaindb.Config{
		InMemory: true,
		DataDir:  tmp,
	})
	require.NoError(t, err)

	otherTrie := trie.NewEmptyTrie()
	otherTrie.Put([]byte("simple"), []byte("cat"))

	otherHash, err := otherTrie.Hash()
	require.NoError(t, err)

	tr := trie.NewEmptyTrie()
	tr.Put([]byte("do"), []byte("verb"))
	tr.Put([]byte("domain"), []byte("website"))
	tr.Put([]byte("other"), []byte("random"))
	tr.Put([]byte("otherwise"), []byte("randomstuff"))
	tr.Put([]byte("cat"), []byte("another animal"))

	err = tr.WriteDirty(memdb)
	require.NoError(t, err)

	hash, err := tr.Hash()
	require.NoError(t, err)

	keys := [][]byte{
		[]byte("do"),
		[]byte("domain"),
		[]byte("other"),
		[]byte("otherwise"),
		[]byte("cat"),
	}

	root := hash.ToBytes()
	otherRoot := otherHash.ToBytes()

	allProofs, err := proof.Generate(root, keys, memdb)
	require.NoError(t, err)

	testcases := map[string]struct {
		root, key, value []byte
		proof            [][]byte
		expect           bool
	}{
		"Proof_should_be_true": {
			root: root, key: []byte("do"), proof: allProofs, value: []byte("verb"), expect: true},
		"Root_empty,_proof_should_be_false": {
			root: []byte{}, key: []byte("do"), proof: allProofs, value: []byte("verb"), expect: false},
		"Other_root,_proof_should_be_false": {
			root: otherRoot, key: []byte("do"), proof: allProofs, value: []byte("verb"), expect: false},
		"Value_empty,_proof_should_be_true": {
			root: root, key: []byte("do"), proof: allProofs, value: nil, expect: true},
		"Unknow_key,_proof_should_be_false": {
			root: root, key: []byte("unknow"), proof: allProofs, value: nil, expect: false},
		"Key_and_value_unknow,_proof_should_be_false": {
			root: root, key: []byte("unknow"), proof: allProofs, value: []byte("unknow"), expect: false},
		"Empty_proof,_should_be_false": {
			root: root, key: []byte("do"), proof: [][]byte{}, value: nil, expect: false},
	}

	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	for name, testcase := range testcases {
		testcase := testcase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			hashEnc, err := scale.Marshal(testcase.root)
			require.NoError(t, err)

			args := hashEnc

			encProof, err := scale.Marshal(testcase.proof)
			require.NoError(t, err)
			args = append(args, encProof...)

			keyEnc, err := scale.Marshal(testcase.key)
			require.NoError(t, err)
			args = append(args, keyEnc...)

			valueEnc, err := scale.Marshal(testcase.value)
			require.NoError(t, err)
			args = append(args, valueEnc...)

			res, err := inst.Exec("rtm_ext_trie_blake2_256_verify_proof_version_1", args)
			require.NoError(t, err)

			var got bool
			err = scale.Unmarshal(res, &got)
			require.NoError(t, err)
			require.Equal(t, testcase.expect, got)
		})
	}
}

func Test_ext_misc_runtime_version_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	fp, err := runtime.GetRuntime(context.Background(), runtime.WESTEND_RUNTIME_v0929)
	require.NoError(t, err)

	// Reads the WebAssembly module as bytes.
	// Retrieve WASM binary
	bytes, err := ioutil.ReadFile(fp)
	if err != nil {
		t.Errorf("Failed to read wasm file: %s", err)
	}

	if strings.Contains(fp, "compact") {
		var err error
		bytes, err = decompressWasm(bytes)
		if err != nil {
			t.Errorf("%v", err)
		}
	}

	data := bytes

	dataLength := uint32(len(data))
	inputPtr, err := inst.Context.Allocator.Allocate(dataLength)
	if err != nil {
		t.Errorf("allocating input memory: %v", err)
	}

	defer inst.Context.Allocator.Clear()

	// Store the data into memory
	mem := inst.Module.Memory()
	ok := mem.Write(inputPtr, data)
	if !ok {
		panic("wtf?")
	}

	dataSpan := newPointerSize(inputPtr, dataLength)
	ctx := context.WithValue(context.Background(), runtimeContextKey, inst.Context)
	versionPtr := ext_misc_runtime_version_version_1(ctx, inst.Module, dataSpan)

	var option *[]byte
	versionData := read(inst.Module, versionPtr)
	err = scale.Unmarshal(versionData, &option)
	require.NoError(t, err)
	require.NotNil(t, option)

	version, err := runtime.DecodeVersion(*option)
	require.NoError(t, err)

	require.Equal(t, "parity-westend", string(version.ImplName))
	require.Equal(t, "westend", string(version.SpecName))
}

func TestWestendInstance(t *testing.T) {
	NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929)
}
