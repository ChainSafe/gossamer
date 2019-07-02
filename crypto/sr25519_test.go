package crypto

import (
	"math/rand"
	"testing"
)

func TestKeypairFromSeed(t *testing.T) {
	keypair_out := make([]byte, 96)
	seed_ptr := []byte{}

	buf := make([]byte, 32)
	rand.Read(buf)
	seed_ptr = buf

	err := sr25519_keypair_from_seed(&keypair_out, &seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keypair_out)
} 

func TestDeriveKeypair(t *testing.T) {
	keypair_out := make([]byte, 96)
	pair_ptr := []byte{}
	cc_ptr := []byte{}

	buf := make([]byte, 96)
	rand.Read(buf)
	pair_ptr = buf

	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err := sr25519_derive_keypair_hard(&keypair_out, &pair_ptr, &cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keypair_out)
} 