// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

var testPassword = []byte("1234")

func TestBasicKeystore(t *testing.T) {
	ks := NewBasicKeystore("test", crypto.Sr25519Type)

	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	addr := kp.Public().Address()
	ks.Insert(kp)
	kp2 := ks.GetKeypairFromAddress(addr)

	if !reflect.DeepEqual(kp, kp2) {
		t.Fatalf("Fail: got %v expected %v", kp2, kp)
	}
}

func TestBasicKeystore_PublicKeys(t *testing.T) {
	ks := NewBasicKeystore("test", crypto.Sr25519Type)

	expectedPubkeys := []crypto.PublicKey{}
	numKps := 12

	for i := 0; i < numKps; i++ {
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
		expectedPubkeys = append(expectedPubkeys, kp.Public())
	}

	pubkeys := ks.PublicKeys()
	sort.Slice(pubkeys, func(i, j int) bool {
		return strings.Compare(string(pubkeys[i].Address()), string(pubkeys[j].Address())) < 0
	})
	sort.Slice(expectedPubkeys, func(i, j int) bool {
		return strings.Compare(string(expectedPubkeys[i].Address()), string(expectedPubkeys[j].Address())) < 0
	})

	if !reflect.DeepEqual(pubkeys, expectedPubkeys) {
		t.Fatalf("Fail: got %v expected %v", pubkeys, expectedPubkeys)
	}
}
