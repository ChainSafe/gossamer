// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wazero_runtime

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/lib/trie/proof"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ext_offchain_index_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.Context.NodeStorage.BaseDB.Put(testKey, testValue)
	require.NoError(t, err)

	value, err := inst.Context.NodeStorage.BaseDB.Get(testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, value)

	encKey, err := scale.Marshal(testKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_offchain_index_clear_version_1", encKey)
	require.NoError(t, err)

	_, err = inst.Context.NodeStorage.BaseDB.Get(testKey)
	require.ErrorIs(t, err, database.ErrNotFound)
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

	params := append(idData, seedData...) //skipcq: CRT-D0001

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
	copy(value, val[:])

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

func Test_ext_crypto_ecdsa_verify_version_2_Table(t *testing.T) {
	testCases := map[string]struct {
		sig      []byte
		msg      []byte
		key      []byte
		expected []byte
		err      string
	}{
		"valid_signature": {
			sig:      []byte{5, 1, 187, 179, 88, 183, 46, 115, 242, 32, 9, 54, 141, 207, 44, 15, 238, 42, 217, 196, 111, 173, 239, 204, 128, 93, 49, 179, 137, 150, 162, 125, 226, 225, 28, 145, 122, 127, 15, 154, 185, 11, 3, 66, 27, 187, 204, 242, 107, 68, 26, 111, 245, 30, 115, 141, 85, 74, 158, 211, 161, 217, 43, 151, 120, 125, 1}, //nolint:lll
			msg:      []byte{48, 72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 33},
			key:      []byte{132, 2, 39, 206, 55, 134, 131, 142, 43, 100, 63, 134, 96, 14, 253, 15, 222, 119, 154, 110, 188, 20, 159, 62, 125, 42, 59, 127, 19, 16, 0, 161, 236, 109}, //nolint:lll
			expected: []byte{1, 0, 0, 0},
		},
		"invalid_signature": {
			sig:      []byte{5, 1, 187, 0, 0, 183, 46, 115, 242, 32, 9, 54, 141, 207, 44, 15, 238, 42, 217, 196, 111, 173, 239, 204, 128, 93, 49, 179, 137, 150, 162, 125, 226, 225, 28, 145, 122, 127, 15, 154, 185, 11, 3, 66, 27, 187, 204, 242, 107, 68, 26, 111, 245, 30, 115, 141, 85, 74, 158, 211, 161, 217, 43, 151, 120, 125, 1}, //nolint:lll
			msg:      []byte{48, 72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 33},
			key:      []byte{132, 2, 39, 206, 55, 134, 131, 142, 43, 100, 63, 134, 96, 14, 253, 15, 222, 119, 154, 110, 188, 20, 159, 62, 125, 42, 59, 127, 19, 16, 0, 161, 236, 109}, //nolint:lll
			expected: []byte{0, 0, 0, 0},
		},
		"wrong_key": {
			sig:      []byte{5, 1, 187, 0, 0, 183, 46, 115, 242, 32, 9, 54, 141, 207, 44, 15, 238, 42, 217, 196, 111, 173, 239, 204, 128, 93, 49, 179, 137, 150, 162, 125, 226, 225, 28, 145, 122, 127, 15, 154, 185, 11, 3, 66, 27, 187, 204, 242, 107, 68, 26, 111, 245, 30, 115, 141, 85, 74, 158, 211, 161, 217, 43, 151, 120, 125, 1}, //nolint:lll
			msg:      []byte{48, 72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 33},
			key:      []byte{132, 2, 39, 0, 55, 134, 131, 142, 43, 100, 63, 134, 96, 14, 253, 15, 222, 119, 154, 110, 188, 20, 159, 62, 125, 42, 59, 127, 19, 16, 0, 161, 236, 109}, //nolint:lll
			expected: []byte{0, 0, 0, 0},
		},
		"invalid_key": {
			sig: []byte{5, 1, 187, 0, 0, 183, 46, 115, 242, 32, 9, 54, 141, 207, 44, 15, 238, 42, 217, 196, 111, 173, 239, 204, 128, 93, 49, 179, 137, 150, 162, 125, 226, 225, 28, 145, 122, 127, 15, 154, 185, 11, 3, 66, 27, 187, 204, 242, 107, 68, 26, 111, 245, 30, 115, 141, 85, 74, 158, 211, 161, 217, 43, 151, 120, 125, 1}, //nolint:lll
			msg: []byte{48, 72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 33},
			key: []byte{132, 2, 39, 55, 134, 131, 142, 43, 100, 63, 134, 96, 14, 253, 15, 222, 119, 154, 110, 188, 20, 159, 62, 125, 42, 59, 127, 19, 16, 0, 161, 236, 109}, //nolint:lll
			err: "running runtime function: wasm error",
		},
		"invalid_message": {
			sig: []byte{5, 1, 187, 179, 88, 183, 46, 115, 242, 32, 9, 54, 141, 207, 44, 15, 238, 42, 217, 196, 111, 173, 239, 204, 128, 93, 49, 179, 137, 150, 162, 125, 226, 225, 28, 145, 122, 127, 15, 154, 185, 11, 3, 66, 27, 187, 204, 242, 107, 68, 26, 111, 245, 30, 115, 141, 85, 74, 158, 211, 161, 217, 43, 151, 120, 125, 1}, //nolint:lll
			msg: []byte{48, 72, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100},
			key: []byte{132, 2, 39, 206, 55, 134, 131, 142, 43, 100, 63, 134, 96, 14, 253, 15, 222, 119, 154, 110, 188, 20, 159, 62, 125, 42, 59, 127, 19, 16, 0, 161, 236, 109}, //nolint:lll
			err: "running runtime function: wasm error",
		},
	}
	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

			ret, err := inst.Exec("rtm_ext_crypto_ecdsa_verify_version_2", append(append(tc.sig, tc.msg...), tc.key...))
			assert.Equal(t, tc.expected, ret)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			assert.NoError(t, err)
		})
	}
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

	params := append(idData, seedData...) //skipcq: CRT-D0001

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
	copy(value, val[:])

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
	memdb, err := database.NewPebble(tmp, true)
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
	bytes, err := os.ReadFile(fp)
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
		panic("write overlow")
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

var (
	testChildKey = []byte("childKey")
	testKey      = []byte("key")
	testValue    = []byte("value")
)

func Test_ext_default_child_storage_read_version_1(t *testing.T) {
	testOffset := uint32(2)
	testBufferSize := uint32(100)

	expectedValue := testValue[testOffset:]
	cases := map[string]struct {
		setupInstance func(t *testing.T) *Instance
		expected      *[]byte
	}{
		"should_return_none_if_child_doesnt_exist": {
			setupInstance: func(t *testing.T) *Instance {
				return NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
			},
			expected: nil,
		},

		"child_exists_return_value": {
			setupInstance: func(t *testing.T) *Instance {
				inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

				err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
				require.NoError(t, err)

				err = inst.Context.Storage.SetChildStorage(testChildKey, testKey, testValue)
				require.NoError(t, err)
				return inst
			},
			expected: &expectedValue,
		},
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			inst := tt.setupInstance(t)

			encChildKey, err := scale.Marshal(testChildKey)
			require.NoError(t, err)

			encKey, err := scale.Marshal(testKey)
			require.NoError(t, err)

			encBufferSize, err := scale.Marshal(testBufferSize)
			require.NoError(t, err)

			encOffset, err := scale.Marshal(testOffset)
			require.NoError(t, err)

			args := bytes.Join([][]byte{
				encChildKey, encKey,
				encOffset, encBufferSize,
			}, nil)

			ret, err := inst.Exec("rtm_ext_default_child_storage_read_version_1", args)
			require.NoError(t, err)

			var read *[]byte
			err = scale.Unmarshal(ret, &read)
			require.NoError(t, err)

			if tt.expected == nil {
				require.Nil(t, read)
			} else {
				val := *read
				require.Equal(t, testValue[testOffset:], val[:len(testValue)-int(testOffset)])
			}
		})
	}
}

func Test_ext_default_child_storage_set_version_1(t *testing.T) {
	cases := map[string]struct {
		setupInstance    func(*testing.T) *Instance
		existsBeforehand bool
	}{
		"child_trie_exists_should_not_panic": {
			existsBeforehand: true,
			setupInstance: func(t *testing.T) *Instance {
				inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

				err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
				require.NoError(t, err)

				return inst
			},
		},
		"child_trie_not_found_should_create_a_empty_one": {
			existsBeforehand: false,
			setupInstance: func(t *testing.T) *Instance {
				inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
				return inst
			},
		},
	}

	insertKeyAndValue := func(t *testing.T, inst *Instance, childKey, key, value []byte) {
		encChildKey, err := scale.Marshal(childKey)
		require.NoError(t, err)

		encKey, err := scale.Marshal(key)
		require.NoError(t, err)

		encVal, err := scale.Marshal(value)
		require.NoError(t, err)

		args := bytes.Join([][]byte{
			encChildKey, encKey, encVal,
		}, nil)

		_, err = inst.Exec("rtm_ext_default_child_storage_set_version_1", args)
		require.NoError(t, err)
	}

	getValueFromChildStorage := func(t *testing.T, inst *Instance, childKey, key []byte) *[]byte {
		encChildKey, err := scale.Marshal(childKey)
		require.NoError(t, err)

		encKey, err := scale.Marshal(key)
		require.NoError(t, err)

		ret, err := inst.Exec("rtm_ext_default_child_storage_get_version_1", append(encChildKey, encKey...))
		require.NoError(t, err)

		var retrieved *[]byte
		err = scale.Unmarshal(ret, &retrieved)
		require.NoError(t, err)

		return retrieved
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			inst := tt.setupInstance(t)

			exampleChildKey := []byte("example_child_key")
			exampleKey := []byte("key_to_account")
			exampleValue := []byte("some_acc_address")

			insertKeyAndValue(t, inst, exampleChildKey, exampleKey, exampleValue)

			anotherKey := []byte("key_to_account_2")
			anotherValue := []byte("some_acc_address_2")
			insertKeyAndValue(t, inst, exampleChildKey, anotherKey, anotherValue)

			// should be possible to retrive the first address and the new inserted one

			acc1 := getValueFromChildStorage(t, inst, exampleChildKey, exampleKey)
			require.Equal(t, &exampleValue, acc1)

			acc2 := getValueFromChildStorage(t, inst, exampleChildKey, anotherKey)
			require.Equal(t, &anotherValue, acc2)
		})
	}
}

func Test_ext_default_child_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.Context.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	// Confirm if value is set
	val, err := inst.Context.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Equal(t, testValue, val)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Marshal(testKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	val, err = inst.Context.Storage.GetChildStorage(testChildKey, testKey)
	require.NoError(t, err)
	require.Nil(t, val)
}

func Test_ext_default_child_storage_clear_prefix_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	prefix := []byte("key")

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("keyOne"), []byte("value1")},
		{[]byte("keyTwo"), []byte("value2")},
		{[]byte("keyThree"), []byte("value3")},
	}

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.Context.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	// Confirm if value is set
	keys, err := inst.Context.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	encPrefix, err := scale.Marshal(prefix)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_prefix_version_1", append(encChildKey, encPrefix...))
	require.NoError(t, err)

	keys, err = inst.Context.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 0, len(keys))
}

func Test_ext_default_child_storage_exists_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.Context.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Marshal(testKey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_exists_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
}

func Test_ext_default_child_storage_get_version_1(t *testing.T) {
	cases := map[string]struct {
		setupInstance func(t *testing.T) *Instance
		expected      *[]byte
	}{
		"value_exists_expected_value": {
			setupInstance: func(t *testing.T) *Instance {
				inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
				err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
				require.NoError(t, err)

				err = inst.Context.Storage.SetChildStorage(testChildKey, testKey, testValue)
				require.NoError(t, err)
				return inst
			},
			expected: &testValue,
		},
		"value_not_found_expected_none": {
			setupInstance: func(t *testing.T) *Instance {
				return NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)
			},
			expected: nil,
		},
	}

	for tname, tt := range cases {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			inst := tt.setupInstance(t)

			encChildKey, err := scale.Marshal(testChildKey)
			require.NoError(t, err)

			encKey, err := scale.Marshal(testKey)
			require.NoError(t, err)

			ret, err := inst.Exec("rtm_ext_default_child_storage_get_version_1", append(encChildKey, encKey...))
			require.NoError(t, err)

			var read *[]byte
			err = scale.Unmarshal(ret, &read)
			require.NoError(t, err)

			require.Equal(t, tt.expected, read)
		})
	}
}

func Test_ext_default_child_storage_next_key_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("apple"), []byte("value1")},
		{[]byte("key"), []byte("value2")},
	}

	key := testKeyValuePair[0].key

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.Context.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	encKey, err := scale.Marshal(key)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_next_key_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
	require.Equal(t, testKeyValuePair[1].key, *read)
}

func Test_ext_default_child_storage_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	err = inst.Context.Storage.SetChildStorage(testChildKey, testKey, testValue)
	require.NoError(t, err)

	child, err := inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)

	rootHash, err := child.Hash()
	require.NoError(t, err)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)
	encKey, err := scale.Marshal(testKey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_default_child_storage_root_version_1", append(encChildKey, encKey...))
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	// Convert decoded interface to common Hash
	actualValue := common.BytesToHash(hash)
	require.Equal(t, rootHash, actualValue)
}

func Test_ext_default_child_storage_storage_kill_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_default_child_storage_storage_kill_version_1", encChildKey)
	require.NoError(t, err)

	child, _ = inst.Context.Storage.GetChild(testChildKey)
	require.Nil(t, child)
}

func Test_ext_default_child_storage_storage_kill_version_2_limit_all(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	tr := trie.NewEmptyTrie()
	tr.Put([]byte(`key2`), []byte(`value2`))
	tr.Put([]byte(`key1`), []byte(`value1`))
	err := inst.Context.Storage.SetChild(testChildKey, tr)
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	testLimit := uint32(2)
	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, testLimit)

	optLimit, err := scale.Marshal(&testLimitBytes)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_default_child_storage_storage_kill_version_2", append(encChildKey, optLimit...))
	require.NoError(t, err)
	require.Equal(t, []byte{1, 0, 0, 0}, res)

	child, err = inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.Empty(t, child.Entries())
}

func Test_ext_default_child_storage_storage_kill_version_2_limit_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	tr := trie.NewEmptyTrie()
	tr.Put([]byte(`key2`), []byte(`value2`))
	tr.Put([]byte(`key1`), []byte(`value1`))
	err := inst.Context.Storage.SetChild(testChildKey, tr)
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	testLimit := uint32(1)
	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, testLimit)

	optLimit, err := scale.Marshal(&testLimitBytes)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_default_child_storage_storage_kill_version_2", append(encChildKey, optLimit...))
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 0}, res)

	child, err = inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.Equal(t, 1, len(child.Entries()))
}

func Test_ext_default_child_storage_storage_kill_version_2_limit_none(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	tr := trie.NewEmptyTrie()
	tr.Put([]byte(`key2`), []byte(`value2`))
	tr.Put([]byte(`key1`), []byte(`value1`))
	err := inst.Context.Storage.SetChild(testChildKey, tr)
	require.NoError(t, err)

	// Confirm if value is set
	child, err := inst.Context.Storage.GetChild(testChildKey)
	require.NoError(t, err)
	require.NotNil(t, child)

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	var val *[]byte
	optLimit, err := scale.Marshal(val)
	require.NoError(t, err)

	res, err := inst.Exec("rtm_ext_default_child_storage_storage_kill_version_2", append(encChildKey, optLimit...))
	require.NoError(t, err)
	require.Equal(t, []byte{1, 0, 0, 0}, res)

	child, err = inst.Context.Storage.GetChild(testChildKey)
	require.Error(t, err)
	require.Nil(t, child)
}

func Test_ext_default_child_storage_storage_kill_version_3(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	tr := trie.NewEmptyTrie()
	tr.Put([]byte(`key2`), []byte(`value2`))
	tr.Put([]byte(`key1`), []byte(`value1`))
	tr.Put([]byte(`key3`), []byte(`value3`))
	err := inst.Context.Storage.SetChild(testChildKey, tr)
	require.NoError(t, err)

	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, uint32(2))
	optLimit2 := &testLimitBytes

	testCases := []struct {
		key      []byte
		limit    *[]byte
		expected []byte
		errMsg   string
	}{
		{
			key:      []byte(`fakekey`),
			limit:    optLimit2,
			expected: nil, // supposed to return None
		},
		{key: testChildKey, limit: optLimit2, expected: []byte{1, 2, 0, 0, 0}},
		{key: testChildKey, limit: nil, expected: []byte{0, 1, 0, 0, 0}},
	}

	for _, test := range testCases {
		encChildKey, err := scale.Marshal(test.key)
		require.NoError(t, err)
		encOptLimit, err := scale.Marshal(test.limit)
		require.NoError(t, err)
		res, err := inst.Exec("rtm_ext_default_child_storage_storage_kill_version_3", append(encChildKey, encOptLimit...))
		if test.errMsg != "" {
			require.Error(t, err)
			require.EqualError(t, err, test.errMsg)
			continue
		}
		require.NoError(t, err)

		var read *[]byte
		err = scale.Unmarshal(res, &read)
		require.NoError(t, err)
		if test.expected == nil {
			require.Nil(t, read)
		} else {
			require.Equal(t, test.expected, *read)
		}
	}
}

func Test_ext_hashing_blake2_128_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_blake2_128_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected, err := common.Blake2b128(data)
	require.NoError(t, err)
	require.Equal(t, expected, hash)
}

func Test_ext_hashing_blake2_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_blake2_256_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected, err := common.Blake2bHash(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_keccak_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_keccak_256_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected, err := common.Keccak256(data)
	require.NoError(t, err)
	require.Equal(t, expected[:], hash)
}

func Test_ext_hashing_twox_128_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_twox_128_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected, err := common.Twox128Hash(data)
	require.NoError(t, err)
	require.Equal(t, expected, hash)
}

func Test_ext_hashing_twox_64_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_twox_64_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected, err := common.Twox64(data)
	require.NoError(t, err)
	require.Equal(t, expected, hash)
}

func Test_ext_hashing_sha2_256_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	data := []byte("helloworld")
	enc, err := scale.Marshal(data)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_hashing_sha2_256_version_1", enc)
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected := common.Sha256(data)
	require.Equal(t, expected[:], hash)
}

func Test_ext_offchain_timestamp_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	ret, err := inst.Exec("rtm_ext_offchain_timestamp_version_1", []byte{})
	require.NoError(t, err)

	var timestamp int64
	err = scale.Unmarshal(ret, &timestamp)
	require.NoError(t, err)

	expected := time.Now().Unix()
	require.GreaterOrEqual(t, expected, timestamp)
}

func Test_ext_offchain_sleep_until_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	input := time.Now().UnixMilli()
	enc, err := scale.Marshal(input)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_offchain_sleep_until_version_1", enc) // auto conversion to i64
	require.NoError(t, err)
}

func Test_ext_default_child_storage_clear_prefix_version_2(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	prefix := []byte("key")

	testKeyValuePair := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("keyOne"), []byte("value1")},
		{[]byte("keyTwo"), []byte("value2")},
		{[]byte("keyThree"), []byte("value3")},
	}

	err := inst.Context.Storage.SetChild(testChildKey, trie.NewEmptyTrie())
	require.NoError(t, err)

	for _, kv := range testKeyValuePair {
		err = inst.Context.Storage.SetChildStorage(testChildKey, kv.key, kv.value)
		require.NoError(t, err)
	}

	// Confirm if value is set
	keys, err := inst.Context.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))

	encChildKey, err := scale.Marshal(testChildKey)
	require.NoError(t, err)

	encPrefix, err := scale.Marshal(prefix)
	require.NoError(t, err)

	testLimit := uint32(1)
	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, testLimit)

	encLimit, err := scale.Marshal(&testLimitBytes)
	require.NoError(t, err)

	data := append(encChildKey, encPrefix...)
	data = append(data, encLimit...)

	_, err = inst.Exec("rtm_ext_default_child_storage_clear_prefix_version_2", data)
	require.NoError(t, err)

	keys, err = inst.Context.Storage.(*storage.TrieState).GetKeysWithPrefixFromChild(testChildKey, prefix)
	require.NoError(t, err)
	// since one key is removed, there will be two remaining.
	require.Equal(t, 2, len(keys))
}

func Test_ext_offchain_local_storage_clear_version_1_Persistent(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("key1")
	err := inst.Context.NodeStorage.PersistentStorage.Put(testkey, []byte{1})
	require.NoError(t, err)

	kind := int32(1)
	encKind, err := scale.Marshal(kind)
	require.NoError(t, err)

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_offchain_local_storage_clear_version_1", append(encKind, encKey...))
	require.NoError(t, err)

	val, err := inst.Context.NodeStorage.PersistentStorage.Get(testkey)
	require.EqualError(t, err, "pebble: not found")
	require.Nil(t, val)
}

func Test_ext_offchain_local_storage_clear_version_1_Local(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("key1")
	err := inst.Context.NodeStorage.LocalStorage.Put(testkey, []byte{1})
	require.NoError(t, err)

	kind := int32(2)
	encKind, err := scale.Marshal(kind)
	require.NoError(t, err)

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_offchain_local_storage_clear_version_1", append(encKind, encKey...))
	require.NoError(t, err)

	val, err := inst.Context.NodeStorage.LocalStorage.Get(testkey)
	require.EqualError(t, err, "pebble: not found")
	require.Nil(t, val)
}

func Test_ext_offchain_http_request_start_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	encMethod, err := scale.Marshal([]byte("GET"))
	require.NoError(t, err)

	encURI, err := scale.Marshal([]byte("https://chainsafe.io"))
	require.NoError(t, err)

	var optMeta *[]byte
	encMeta, err := scale.Marshal(optMeta)
	require.NoError(t, err)

	params := append([]byte{}, encMethod...)
	params = append(params, encURI...)
	params = append(params, encMeta...)

	resReqID := scale.NewResult(int16(0), nil)

	// start request number 0
	ret, err := inst.Exec("rtm_ext_offchain_http_request_start_version_1", params)
	require.NoError(t, err)

	err = scale.Unmarshal(ret, &resReqID)
	require.NoError(t, err)

	requestNumber, err := resReqID.Unwrap()
	require.NoError(t, err)
	require.Equal(t, int16(1), requestNumber)

	// start request number 1
	ret, err = inst.Exec("rtm_ext_offchain_http_request_start_version_1", params)
	require.NoError(t, err)

	resReqID = scale.NewResult(int16(0), nil)

	err = scale.Unmarshal(ret, &resReqID)
	require.NoError(t, err)

	requestNumber, err = resReqID.Unwrap()
	require.NoError(t, err)
	require.Equal(t, int16(2), requestNumber)

	// start request number 2
	resReqID = scale.NewResult(int16(0), nil)
	ret, err = inst.Exec("rtm_ext_offchain_http_request_start_version_1", params)
	require.NoError(t, err)

	err = scale.Unmarshal(ret, &resReqID)
	require.NoError(t, err)

	requestNumber, err = resReqID.Unwrap()
	require.NoError(t, err)
	require.Equal(t, int16(3), requestNumber)
}

func Test_ext_offchain_http_request_add_header(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	cases := map[string]struct {
		key, value  string
		expectedErr bool
	}{
		"should_add_headers_without_problems": {
			key:         "SOME_HEADER_KEY",
			value:       "SOME_HEADER_VALUE",
			expectedErr: false,
		},

		"should_return_a_result_error": {
			key:         "",
			value:       "",
			expectedErr: true,
		},
	}

	for tname, tcase := range cases {
		tcase := tcase
		t.Run(tname, func(t *testing.T) {
			reqID, err := inst.Context.OffchainHTTPSet.StartRequest(http.MethodGet, "http://uri.example")
			require.NoError(t, err)

			encID, err := scale.Marshal(uint32(reqID))
			require.NoError(t, err)

			encHeaderKey, err := scale.Marshal(tcase.key)
			require.NoError(t, err)

			encHeaderValue, err := scale.Marshal(tcase.value)
			require.NoError(t, err)

			params := append([]byte{}, encID...)
			params = append(params, encHeaderKey...)
			params = append(params, encHeaderValue...)

			ret, err := inst.Exec("rtm_ext_offchain_http_request_add_header_version_1", params)
			require.NoError(t, err)

			gotResult := scale.NewResult(nil, nil)
			err = scale.Unmarshal(ret, &gotResult)
			require.NoError(t, err)

			ok, err := gotResult.Unwrap()
			if tcase.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			offchainReq := inst.Context.OffchainHTTPSet.Get(reqID)
			gotValue := offchainReq.Request.Header.Get(tcase.key)
			require.Equal(t, tcase.value, gotValue)

			require.Nil(t, ok)
		})
	}
}

func Test_ext_storage_clear_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	inst.Context.Storage.Put(testkey, []byte{1})

	enc, err := scale.Marshal(testkey)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_version_1", enc)
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Nil(t, val)
}

func Test_ext_storage_clear_prefix_version_1_hostAPI(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("static")
	inst.Context.Storage.Put(testkey, []byte("Inverse"))

	testkey2 := []byte("even-keeled")
	inst.Context.Storage.Put(testkey2, []byte("Future-proofed"))

	enc, err := scale.Marshal(testkey[:3])
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_prefix_version_1", enc)
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Nil(t, val)

	val = inst.Context.Storage.Get(testkey2)
	require.NotNil(t, val)
}

func Test_ext_storage_clear_prefix_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	inst.Context.Storage.Put(testkey, []byte{1})

	testkey2 := []byte("spaghet")
	inst.Context.Storage.Put(testkey2, []byte{2})

	enc, err := scale.Marshal(testkey[:3])
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_clear_prefix_version_1", enc)
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Nil(t, val)

	val = inst.Context.Storage.Get(testkey2)
	require.NotNil(t, val)
}

func Test_ext_storage_clear_prefix_version_2(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	inst.Context.Storage.Put(testkey, []byte{1})

	testkey2 := []byte("noot1")
	inst.Context.Storage.Put(testkey2, []byte{1})

	testkey3 := []byte("noot2")
	inst.Context.Storage.Put(testkey3, []byte{1})

	testkey4 := []byte("noot3")
	inst.Context.Storage.Put(testkey4, []byte{1})

	testkey5 := []byte("spaghet")
	testValue5 := []byte{2}
	inst.Context.Storage.Put(testkey5, testValue5)

	enc, err := scale.Marshal(testkey[:3])
	require.NoError(t, err)

	testLimit := uint32(2)
	testLimitBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(testLimitBytes, testLimit)

	optLimit, err := scale.Marshal(&testLimitBytes)
	require.NoError(t, err)

	// clearing prefix for "noo" prefix with limit 2
	encValue, err := inst.Exec("rtm_ext_storage_clear_prefix_version_2", append(enc, optLimit...))
	require.NoError(t, err)

	var decVal []byte
	scale.Unmarshal(encValue, &decVal)

	var numDeleted uint32
	// numDeleted represents no. of actual keys deleted
	scale.Unmarshal(decVal[1:], &numDeleted)
	require.Equal(t, uint32(2), numDeleted)

	var expectedAllDeleted byte
	// expectedAllDeleted value 0 represents all keys deleted, 1 represents keys are pending with prefix in trie
	expectedAllDeleted = 1
	require.Equal(t, expectedAllDeleted, decVal[0])

	val := inst.Context.Storage.Get(testkey)
	require.NotNil(t, val)

	val = inst.Context.Storage.Get(testkey5)
	require.NotNil(t, val)
	require.Equal(t, testValue5, val)

	// clearing prefix again for "noo" prefix with limit 2
	encValue, err = inst.Exec("rtm_ext_storage_clear_prefix_version_2", append(enc, optLimit...))
	require.NoError(t, err)

	scale.Unmarshal(encValue, &decVal)
	scale.Unmarshal(decVal[1:], &numDeleted)
	require.Equal(t, uint32(2), numDeleted)

	expectedAllDeleted = 0
	require.Equal(t, expectedAllDeleted, decVal[0])

	val = inst.Context.Storage.Get(testkey)
	require.Nil(t, val)

	val = inst.Context.Storage.Get(testkey5)
	require.NotNil(t, val)
	require.Equal(t, testValue5, val)
}

func Test_ext_storage_get_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte{1, 2}
	inst.Context.Storage.Put(testkey, testvalue)

	enc, err := scale.Marshal(testkey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_get_version_1", enc)
	require.NoError(t, err)

	var value *[]byte
	err = scale.Unmarshal(ret, &value)
	require.NoError(t, err)
	require.NotNil(t, value)
	require.Equal(t, testvalue, *value)
}

func Test_ext_storage_exists_version_1(t *testing.T) {
	testCases := map[string]struct {
		key    []byte
		value  []byte // leave to nil to not insert pair
		result byte
	}{
		"value_does_not_exist": {
			key:    []byte{1},
			result: 0,
		},
		"empty_value_exists": {
			key:    []byte{1},
			value:  []byte{},
			result: 1,
		},
		"value_exist": {
			key:    []byte{1},
			value:  []byte{2},
			result: 1,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			instance := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

			if testCase.value != nil {
				instance.Context.Storage.Put(testCase.key, testCase.value)
			}

			encodedKey, err := scale.Marshal(testCase.key)
			require.NoError(t, err)

			encodedResult, err := instance.Exec("rtm_ext_storage_exists_version_1", encodedKey)
			require.NoError(t, err)

			var result byte
			err = scale.Unmarshal(encodedResult, &result)
			require.NoError(t, err)

			require.Equal(t, testCase.result, result)
		})
	}
}

func Test_ext_storage_next_key_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	inst.Context.Storage.Put(testkey, []byte{1})

	nextkey := []byte("oot")
	inst.Context.Storage.Put(nextkey, []byte{1})

	enc, err := scale.Marshal(testkey)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_next_key_version_1", enc)
	require.NoError(t, err)

	var next *[]byte
	err = scale.Unmarshal(ret, &next)
	require.NoError(t, err)
	require.NotNil(t, next)
	require.Equal(t, nextkey, *next)
}

func Test_ext_storage_read_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")
	inst.Context.Storage.Put(testkey, testvalue)

	testoffset := uint32(2)
	testBufferSize := uint32(100)

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Marshal(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Marshal(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
	val := *read
	require.Equal(t, testvalue[testoffset:], val[:len(testvalue)-int(testoffset)])
}

func Test_ext_storage_read_version_1_again(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("_was_here_")
	inst.Context.Storage.Put(testkey, testvalue)

	testoffset := uint32(8)
	testBufferSize := uint32(5)

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Marshal(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Marshal(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)

	val := *read
	require.Equal(t, len(testvalue)-int(testoffset), len(val))
	require.Equal(t, testvalue[testoffset:], val[:len(testvalue)-int(testoffset)])
}

func Test_ext_storage_read_version_1_OffsetLargerThanValue(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")
	inst.Context.Storage.Put(testkey, testvalue)

	testoffset := uint32(len(testvalue))
	testBufferSize := uint32(8)

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encOffset, err := scale.Marshal(testoffset)
	require.NoError(t, err)
	encBufferSize, err := scale.Marshal(testBufferSize)
	require.NoError(t, err)

	ret, err := inst.Exec("rtm_ext_storage_read_version_1", append(append(encKey, encOffset...), encBufferSize...))
	require.NoError(t, err)

	var read *[]byte
	err = scale.Unmarshal(ret, &read)
	require.NoError(t, err)
	require.NotNil(t, read)
	val := *read
	require.Equal(t, []byte{}, val)
}

func Test_ext_storage_root_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	ret, err := inst.Exec("rtm_ext_storage_root_version_1", []byte{})
	require.NoError(t, err)

	var hash []byte
	err = scale.Unmarshal(ret, &hash)
	require.NoError(t, err)

	expected := trie.EmptyHash
	require.Equal(t, expected[:], hash)
}

func Test_ext_storage_set_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("washere")

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encValue, err := scale.Marshal(testvalue)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_set_version_1", append(encKey, encValue...))
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Equal(t, testvalue, val)
}

func Test_ext_storage_append_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("was")
	testvalueAppend := []byte("here")

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encVal, err := scale.Marshal(testvalue)
	require.NoError(t, err)
	doubleEncVal, err := scale.Marshal(encVal)
	require.NoError(t, err)

	encArr, err := scale.Marshal([][]byte{testvalue})
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncVal...))
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Equal(t, encArr, val)

	encValueAppend, err := scale.Marshal(testvalueAppend)
	require.NoError(t, err)
	doubleEncValueAppend, err := scale.Marshal(encValueAppend)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncValueAppend...))
	require.NoError(t, err)

	ret := inst.Context.Storage.Get(testkey)
	require.NotNil(t, ret)

	var res [][]byte
	err = scale.Unmarshal(ret, &res)
	require.NoError(t, err)

	require.Equal(t, 2, len(res))
	require.Equal(t, testvalue, res[0])
	require.Equal(t, testvalueAppend, res[1])

	expected, err := scale.Marshal([][]byte{testvalue, testvalueAppend})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func Test_ext_storage_append_version_1_again(t *testing.T) {
	DefaultTestLogLvl = 5
	inst := NewTestInstance(t, runtime.HOST_API_TEST_RUNTIME)

	testkey := []byte("noot")
	testvalue := []byte("abc")
	testvalueAppend := []byte("def")

	encKey, err := scale.Marshal(testkey)
	require.NoError(t, err)
	encVal, err := scale.Marshal(testvalue)
	require.NoError(t, err)
	doubleEncVal, err := scale.Marshal(encVal)
	require.NoError(t, err)

	encArr, err := scale.Marshal([][]byte{testvalue})
	require.NoError(t, err)

	// place SCALE encoded value in storage
	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncVal...))
	require.NoError(t, err)

	val := inst.Context.Storage.Get(testkey)
	require.Equal(t, encArr, val)

	encValueAppend, err := scale.Marshal(testvalueAppend)
	require.NoError(t, err)
	doubleEncValueAppend, err := scale.Marshal(encValueAppend)
	require.NoError(t, err)

	_, err = inst.Exec("rtm_ext_storage_append_version_1", append(encKey, doubleEncValueAppend...))
	require.NoError(t, err)

	ret := inst.Context.Storage.Get(testkey)
	require.NotNil(t, ret)

	var res [][]byte
	err = scale.Unmarshal(ret, &res)
	require.NoError(t, err)

	require.Equal(t, 2, len(res))
	require.Equal(t, testvalue, res[0])
	require.Equal(t, testvalueAppend, res[1])

	expected, err := scale.Marshal([][]byte{testvalue, testvalueAppend})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func TestWestendInstance(t *testing.T) {
	NewTestInstance(t, runtime.WESTEND_RUNTIME_v0929)
}
