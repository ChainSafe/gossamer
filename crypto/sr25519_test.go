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

	err := sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keypair_out)
} 

func TestDeriveKeypairHard(t *testing.T) {
	keypair_out := make([]byte, 96)
	seed_ptr := []byte{}

	buf := make([]byte, 32)
	rand.Read(buf)
	seed_ptr = buf

	err := sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	pair_ptr := keypair_out

	keypair_out = make([]byte, 96)
	cc_ptr := []byte{}


	// todo: what is cc_ptr (chaincode) actually? hash of chaincode?
	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err = sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keypair_out)
} 

func TestDeriveKeypairSoft(t *testing.T) {
	keypair_out := make([]byte, 96)
	seed_ptr := []byte{}

	buf := make([]byte, 32)
	rand.Read(buf)
	seed_ptr = buf

	err := sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	pair_ptr := keypair_out

	keypair_out = make([]byte, 96)
	cc_ptr := []byte{}


	// todo: what is cc_ptr (chaincode) actually? hash of chaincode?
	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err = sr25519_derive_keypair_soft(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(keypair_out)
} 


func TestDerivePublicSoft(t *testing.T) {
	keypair_out := make([]byte, 96)
	seed_ptr := []byte{}

	buf := make([]byte, 32)
	rand.Read(buf)
	seed_ptr = buf

	err := sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	public_ptr := keypair_out[64:]
	pubkey_out := make([]byte, 32)
	cc_ptr := []byte{}


	// todo: what is cc_ptr (chaincode) actually? hash of chaincode?
	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err = sr25519_derive_public_soft(pubkey_out, public_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(pubkey_out)
} 