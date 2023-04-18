// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/tests/utils/config"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/utils"

	"github.com/stretchr/testify/require"
)

func TestLoadKeystore(t *testing.T) {
	ks := NewBasicKeystore("test", crypto.Sr25519Type)

	sr25519KeyRing, err := NewSr25519Keyring()
	require.NoError(t, err)

	err = LoadKeystore(config.AliceKey, ks, sr25519KeyRing)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Size())

	ed25519KeyRing, err := NewEd25519Keyring()
	require.NoError(t, err)

	ks = NewBasicKeystore("test", crypto.Ed25519Type)
	err = LoadKeystore("bob", ks, ed25519KeyRing)
	require.NoError(t, err)
	require.Equal(t, 1, ks.Size())
}

var testKeyTypes = []struct {
	testType     string
	expectedType string
}{
	{testType: "babe", expectedType: crypto.Sr25519Type},
	{testType: "gran", expectedType: crypto.Ed25519Type},
	{testType: "acco", expectedType: crypto.Sr25519Type},
	{testType: "aura", expectedType: crypto.Sr25519Type},
	{testType: "imon", expectedType: crypto.Sr25519Type},
	{testType: "audi", expectedType: crypto.Sr25519Type},
	{testType: "dumy", expectedType: crypto.Sr25519Type},
	{testType: "xxxx", expectedType: crypto.UnknownType},
}

func TestDetermineKeyType(t *testing.T) {
	for _, test := range testKeyTypes {
		output := DetermineKeyType(test.testType)
		require.Equal(t, test.expectedType, output)
	}
}

func TestGenerateKey_Sr25519(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := GenerateKeypair("sr25519", nil, testdir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	keys, err := utils.KeystoreFilepaths(testdir)
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 1 {
		t.Fatal("Fail: expected 1 key in keystore")
	}

	if strings.Compare(keys[0], filepath.Base(keyfile)) != 0 {
		t.Fatalf("Fail: got %s expected %s", keys[0], keyfile)
	}
}

func TestGenerateKey_Ed25519(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := GenerateKeypair("ed25519", nil, testdir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	keys, err := utils.KeystoreFilepaths(testdir)
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 1 {
		t.Fatal("Fail: expected 1 key in keystore")
	}

	if strings.Compare(keys[0], filepath.Base(keyfile)) != 0 {
		t.Fatalf("Fail: got %s expected %s", keys[0], keyfile)
	}

	contents, err := os.ReadFile(keyfile)
	if err != nil {
		t.Fatal(err)
	}

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	if err != nil {
		t.Fatal(err)
	}

	if kscontents.Type != "ed25519" {
		t.Fatalf("Fail: got %s expected %s", kscontents.Type, "ed25519")
	}
}

func TestGenerateKey_Secp256k1(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := GenerateKeypair("secp256k1", nil, testdir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	keys, err := utils.KeystoreFilepaths(testdir)
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 1 {
		t.Fatal("Fail: expected 1 key in keystore")
	}

	if strings.Compare(keys[0], filepath.Base(keyfile)) != 0 {
		t.Fatalf("Fail: got %s expected %s", keys[0], keyfile)
	}

	contents, err := os.ReadFile(keyfile)
	if err != nil {
		t.Fatal(err)
	}

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	if err != nil {
		t.Fatal(err)
	}

	if kscontents.Type != "secp256k1" {
		t.Fatalf("Fail: got %s expected %s", kscontents.Type, "secp256k1")
	}
}

func TestGenerateKey_NoType(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := GenerateKeypair("", nil, testdir, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(keyfile)
	if err != nil {
		t.Fatal(err)
	}

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	if err != nil {
		t.Fatal(err)
	}

	if kscontents.Type != "sr25519" {
		t.Fatalf("Fail: got %s expected %s", kscontents.Type, "sr25519")
	}
}

func TestImportKey_ShouldFail(t *testing.T) {
	testdir := t.TempDir()

	_, err := ImportKeypair("./notakey.key", testdir)
	if err == nil {
		t.Fatal("did not err")
	}
}

func TestImportKey(t *testing.T) {
	basePath := t.TempDir()

	keypath := basePath

	importkeyfile, err := GenerateKeypair("sr25519", nil, keypath, testPassword)
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(importkeyfile)

	keyfile, err := ImportKeypair(importkeyfile, basePath)
	if err != nil {
		t.Fatal(err)
	}

	keys, err := utils.KeystoreFilepaths(basePath)
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != 1 {
		t.Fatal("fail")
	}

	if strings.Compare(keys[0], filepath.Base(keyfile)) != 0 {
		t.Fatalf("Fail: got %s expected %s", keys[0], keyfile)
	}
}

func TestListKeys(t *testing.T) {
	testdir := t.TempDir()

	var expected []string

	for i := 0; i < 5; i++ {
		var err error
		var keyfile string
		if i%2 == 0 {
			keyfile, err = GenerateKeypair("sr25519", nil, testdir, testPassword)
			if err != nil {
				t.Fatal(err)
			}
		} else {
			keyfile, err = GenerateKeypair("ed25519", nil, testdir, testPassword)
			if err != nil {
				t.Fatal(err)
			}
		}

		expected = append(expected, keyfile)
	}

	keys, err := utils.KeystoreFilepaths(testdir)
	if err != nil {
		t.Fatal(err)
	}

	if len(keys) != len(expected) {
		t.Fatalf("Fail: expected %d keys in keystore, got %d", len(expected), len(keys))
	}

	sort.Slice(expected, func(i, j int) bool { return strings.Compare(expected[i], expected[j]) < 0 })

	for i, key := range keys {
		if strings.Compare(key, filepath.Base(expected[i])) != 0 {
			t.Fatalf("Fail: got %s expected %s", key, filepath.Base(expected[i]))
		}
	}
}

func TestUnlockKeys(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := GenerateKeypair("sr25519", nil, testdir, testPassword)
	require.NoError(t, err)

	ks := NewBasicKeystore("test", crypto.Sr25519Type)

	err = UnlockKeys(ks, testdir, "0", string(testPassword))
	require.NoError(t, err)

	priv, err := ReadFromFileAndDecrypt(keyfile, testPassword)
	require.NoError(t, err)

	pub, err := priv.Public()
	require.NoError(t, err)

	kp, err := PrivateKeyToKeypair(priv)
	require.NoError(t, err)
	_ = kp.Public().Hex()

	expected := ks.GetKeypair(pub)
	if !reflect.DeepEqual(expected, kp) {
		t.Fatalf("Fail: got %v expected %v", expected, kp)
	}
}

func TestImportRawPrivateKey_NoType(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := ImportRawPrivateKey(
		"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"", testdir, testPassword)
	require.NoError(t, err)

	contents, err := os.ReadFile(keyfile)
	require.NoError(t, err)

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	require.NoError(t, err)
	require.Equal(t, "sr25519", kscontents.Type)
	require.Equal(t,
		"0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d",
		kscontents.PublicKey)
}

func TestImportRawPrivateKey_Sr25519(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := ImportRawPrivateKey(
		"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"sr25519", testdir, testPassword)
	require.NoError(t, err)

	contents, err := os.ReadFile(keyfile)
	require.NoError(t, err)

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	require.NoError(t, err)
	require.Equal(t, "sr25519", kscontents.Type)
	require.Equal(t,
		"0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d",
		kscontents.PublicKey)
}

func TestImportRawPrivateKey_Ed25519(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := ImportRawPrivateKey(
		"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"ed25519", testdir, testPassword)
	require.NoError(t, err)

	contents, err := os.ReadFile(keyfile)
	require.NoError(t, err)

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	require.NoError(t, err)
	require.Equal(t, "ed25519", kscontents.Type)
	require.Equal(t,
		"0x6dfb362eb332449782b7260bcff6d8777242acdea3293508b22d33ce7336a8b3",
		kscontents.PublicKey)
}

func TestImportRawPrivateKey_Secp256k1(t *testing.T) {
	testdir := t.TempDir()

	keyfile, err := ImportRawPrivateKey(
		"0x33a6f3093f158a7109f679410bef1a0c54168145e0cecb4df006c1c2fffb1f09",
		"secp256k1", testdir, testPassword)
	require.NoError(t, err)

	contents, err := os.ReadFile(keyfile)
	require.NoError(t, err)

	kscontents := new(EncryptedKeystore)
	err = json.Unmarshal(contents, kscontents)
	require.NoError(t, err)
	require.Equal(t, "secp256k1", kscontents.Type)
	require.Equal(t,
		"0x03409094a319b2961660c3ebcc7d206266182c1b3e60d341b5fb17e6851865825c",
		kscontents.PublicKey)
}

func TestDecodeKeyPairFromHex(t *testing.T) {
	keytype := DetermineKeyType("babe")
	seed := "0xfec0f475b818470af5caf1f3c1b1558729961161946d581d2755f9fb566534f8"

	keyBytes, err := common.HexToBytes(seed)
	require.NoError(t, err)

	kp, err := DecodeKeyPairFromHex(keyBytes, keytype)
	require.NoError(t, err)
	require.IsType(t, &sr25519.Keypair{}, kp)

	expectedPublic := "0x34309a9d2a24213896ff06895db16aade8b6502f3a71cf56374cc38520426026"
	require.Equal(t, kp.Public().Hex(), expectedPublic)

	keytype = DetermineKeyType("gran")
	keyBytes, err = common.HexToBytes("0x9d96ebdb66b7b6851d529f0c366393782baeba71732a59ce201ea80760d4d66c")
	require.NoError(t, err)

	kp, err = DecodeKeyPairFromHex(keyBytes, keytype)
	require.NoError(t, err)
	require.IsType(t, &ed25519.Keypair{}, kp)

	expectedPublic = "0xd3db685ed1f94c195dc3e72803fa3d8549df45381388e313fa8170f0b397895c"
	require.Equal(t, kp.Public().Hex(), expectedPublic)

	_, err = DecodeKeyPairFromHex(nil, "")
	require.Error(t, err, "cannot decode key: invalid key type")
}
