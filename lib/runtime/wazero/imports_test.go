package wazero_runtime

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/mocks"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/golang/mock/gomock"
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

// NewInstanceFromFile instantiates a runtime from a .wasm file
func NewInstanceFromFile(fp string, cfg Config) (*Instance, error) {
	// Reads the WebAssembly module as bytes.
	// Retrieve WASM binary
	bytes, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("Failed to read wasm file: %s", err)
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
