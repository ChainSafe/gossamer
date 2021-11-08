// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/secp256k1"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

func TestGetSr25519PublicKeys(t *testing.T) {
	ks := NewGenericKeystore("test")

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

	for i := 0; i < numKps; i++ {
		kp, err := ed25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
	}

	pubkeys := ks.Sr25519PublicKeys()
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

func TestGetEd25519PublicKeys(t *testing.T) {
	ks := NewGenericKeystore("test")

	expectedPubkeys := []crypto.PublicKey{}
	numKps := 10

	for i := 0; i < numKps; i++ {
		kp, err := ed25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
		expectedPubkeys = append(expectedPubkeys, kp.Public())
	}

	for i := 0; i < numKps; i++ {
		kp, err := secp256k1.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
	}

	pubkeys := ks.Ed25519PublicKeys()
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

func TestGetSecp256k1PublicKeys(t *testing.T) {
	ks := NewGenericKeystore("test")

	expectedPubkeys := []crypto.PublicKey{}
	numKps := 10

	for i := 0; i < numKps; i++ {
		kp, err := secp256k1.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
		expectedPubkeys = append(expectedPubkeys, kp.Public())
	}

	for i := 0; i < numKps; i++ {
		kp, err := sr25519.GenerateKeypair()
		if err != nil {
			t.Fatal(err)
		}
		ks.Insert(kp)
	}

	pubkeys := ks.Secp256k1PublicKeys()
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
