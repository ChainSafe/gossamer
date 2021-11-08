// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keystore

import (
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/stretchr/testify/require"
)

func TestNewSr25519Keyring(t *testing.T) {
	kr, err := NewSr25519Keyring()
	if err != nil {
		t.Fatal(err)
	}

	v := reflect.ValueOf(kr).Elem()
	for i := 0; i < v.NumField()-1; i++ {
		pub := v.Field(i).Interface().(*sr25519.Keypair).Public().Hex()

		switch i {
		case 0:
			require.Equal(t, "0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d", pub)
		case 1:
			require.Equal(t, "0x8eaf04151687736326c9fea17e25fc5287613693c912909cb226aa4794f26a48", pub)
		case 2:
			require.Equal(t, "0x90b5ab205c6974c9ea841be688864633dc9ca8a357843eeacf2314649965fe22", pub)
		case 3:
			require.Equal(t, "0x306721211d5404bd9da88e0204360a1a9ab8b87c66c1bc2fcdd37f3c2222cc20", pub)
		case 4:
			require.Equal(t, "0xe659a7a1628cdd93febc04a4e0646ea20e9f5f0ce097d9a05290d4a9e054df4e", pub)
		case 5:
			require.Equal(t, "0x1cbd2d43530a44705ad088af313e18f80b53ef16b36177cd4b77b846f2a5f07c", pub)
		case 6:
			require.Equal(t, "0x4603307f855321776922daeea21ee31720388d097cdaac66f05a6f8462b31757", pub)
		case 7:
			require.Equal(t, "0xbe1d9d59de1283380100550a7b024501cb62d6cc40e3db35fcc5cf341814986e", pub)
		case 8:
			require.Equal(t, "0x1206960f920a23f7f4c43cc9081ec2ed0721f31a9bef2c10fd7602e16e08a32c", pub)
		}
	}
}

func TestNewEd25519Keyring(t *testing.T) {
	kr, err := NewEd25519Keyring()
	if err != nil {
		t.Fatal(err)
	}

	v := reflect.ValueOf(kr).Elem()
	for i := 0; i < v.NumField()-1; i++ {
		key := v.Field(i).Interface().(*ed25519.Keypair).Private().Hex()
		// ed25519 private keys are stored in uncompressed format
		if key[:66] != privateKeys[i] {
			t.Fatalf("Fail: got %s expected %s", key[:66], privateKeys[i])
		}
	}
}
