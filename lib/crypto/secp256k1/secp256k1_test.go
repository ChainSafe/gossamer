// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package secp256k1

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"

	"github.com/stretchr/testify/require"
)

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("borkbork")
	hash, err := common.Blake2bHash(msg)
	if err != nil {
		t.Fatal(err)
	}

	sig, err := kp.private.Sign(hash[:])
	if err != nil {
		t.Fatal(err)
	}

	ok, err := kp.public.Verify(hash[:], sig[:64])
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("did not verify :(")
	}
}

func TestPrivateKeys(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	privbytes := kp.private.Encode()

	priv, err := NewPrivateKey(privbytes)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(kp.private, priv) {
		t.Fatalf("Fail: got %x expected %x", kp.private.Encode(), priv.Encode())
	}
}

func TestPublicKeys(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	kp2 := NewKeypair(kp.private.key)

	if !reflect.DeepEqual(kp.Public(), kp2.Public()) {
		t.Fatalf("Fail: pubkeys do not match got %x expected %x", kp2.Public(), kp.Public())
	}

	pub, err := kp.private.Public()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pub, kp.Public()) {
		t.Fatalf("Fail: pubkeys do not match got %x expected %x", kp2.Public(), kp.Public())
	}
}

func TestEncodeAndDecodePriv(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	enc := kp.Private().Encode()
	res := new(PrivateKey)
	err = res.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}

	exp := kp.Private().(*PrivateKey).Encode()
	if !reflect.DeepEqual(res.Encode(), exp) {
		t.Fatalf("Fail: got %x expected %x", res.Encode(), exp)
	}
}

func TestEncodeAndDecodePub(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	enc := kp.Public().Encode()
	res := new(PublicKey)
	err = res.Decode(enc)
	if err != nil {
		t.Fatal(err)
	}

	exp := kp.Public().(*PublicKey).Encode()
	if !reflect.DeepEqual(res.Encode(), exp) {
		t.Fatalf("Fail: got %v expected %v", res, exp)
	}
}

func TestRecoverPublicKey(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	msg := []byte("borkbork")
	hash, err := common.Blake2bHash(msg)
	require.NoError(t, err)

	sig, err := kp.private.Sign(hash[:])
	require.NoError(t, err)

	recovered, err := RecoverPublicKey(hash[:], sig)
	require.NoError(t, err)

	r := new(PublicKey)
	err = r.UnmarshalPubkey(recovered)
	require.NoError(t, err)
	require.Equal(t, kp.Public(), r)
}

func TestRecoverPublicKeyCompressed(t *testing.T) {
	kp, err := GenerateKeypair()
	require.NoError(t, err)

	msg := []byte("borkbork")
	hash, err := common.Blake2bHash(msg)
	require.NoError(t, err)

	sig, err := kp.private.Sign(hash[:])
	require.NoError(t, err)

	recovered, err := RecoverPublicKeyCompressed(hash[:], sig)
	require.NoError(t, err)

	r := new(PublicKey)
	err = r.Decode(recovered)
	require.NoError(t, err)
	require.Equal(t, kp.Public(), r)
}
