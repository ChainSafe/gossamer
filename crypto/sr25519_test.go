package crypto

import (
	"bytes"
	"github.com/ChainSafe/gossamer/common"
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

	empty := make([]byte, 96)
	if bytes.Equal(keypair_out, empty) {
		t.Errorf("did not derive keypair from seed")
	}
}

func TestDeriveKeypairHard(t *testing.T) {
	pair_ptr, err := common.HexToBytes("0x28b0ae221c6bb06856b287f60d7ea0d98552ea5a16db16956849aa371db3eb51fd190cce74df356432b410bd64682309d6dedb27c76845daf388557cbac3ca3446ebddef8cd9bb167dc30878d7113b7e168e6f0646beffd77d69d39bad76b47a")
	if err != nil {
		t.Fatal(err)
	}

	cc_ptr, err := common.HexToBytes("0x14416c6963650000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)

	err = sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := common.HexToBytes("0xd43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, keypair_out[64:]) {
		t.Errorf("actual pubkey does not match expected: got %x expected %x", keypair_out[64:], expected)
	}
}

func TestDeriveKeypairSoft(t *testing.T) {
	pair_ptr, err := common.HexToBytes("0x28b0ae221c6bb06856b287f60d7ea0d98552ea5a16db16956849aa371db3eb51fd190cce74df356432b410bd64682309d6dedb27c76845daf388557cbac3ca3446ebddef8cd9bb167dc30878d7113b7e168e6f0646beffd77d69d39bad76b47a")
	if err != nil {
		t.Fatal(err)
	}

	cc_ptr, err := common.HexToBytes("0x0c666f6f00000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)

	err = sr25519_derive_keypair_soft(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := common.HexToBytes("0x40b9675df90efa6069ff623b0fdfcf706cd47ca7452a5056c7ad58194d23440a")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, keypair_out[64:]) {
		t.Errorf("actual pubkey does not match expected: got %x expected %x", keypair_out[64:], expected)
	}
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

	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err = sr25519_derive_public_soft(pubkey_out, public_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%x", pubkey_out)
}

func TestSignAndVerify(t *testing.T) {
	pair_ptr, err := common.HexToBytes("0x28b0ae221c6bb06856b287f60d7ea0d98552ea5a16db16956849aa371db3eb51fd190cce74df356432b410bd64682309d6dedb27c76845daf388557cbac3ca3446ebddef8cd9bb167dc30878d7113b7e168e6f0646beffd77d69d39bad76b47a")
	if err != nil {
		t.Fatal(err)
	}

	cc_ptr, err := common.HexToBytes("0x14416c6963650000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)

	err = sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	public_ptr := keypair_out[64:]
	secret_ptr := keypair_out[:64]

	signature_out := make([]byte, 64)
	message_ptr := []byte("helloworld")
	message_length := uint32(len(message_ptr))

	err = sr25519_sign(signature_out, public_ptr, secret_ptr, message_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%x", signature_out)

	ver, err := sr25519_verify(signature_out, message_ptr, public_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	if !ver {
		t.Error("did not verify signature")
	}
}
