package crypto

import (
	"testing"
)

func TestDeriveKeypair(t *testing.T) {
	keypair_out := []byte{}
	pair_ptr := []byte{}
	cc_ptr := []byte{}

	sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
} 