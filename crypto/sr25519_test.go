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

	err := Sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	empty := make([]byte, SR25519_KEYPAIR_SIZE)
	if bytes.Equal(keypair_out, empty) {
		t.Errorf("did not derive keypair from seed")
	}
}

func deriveKeypair() ([]byte, error) {
	pair_ptr, err := common.HexToBytes("0x28b0ae221c6bb06856b287f60d7ea0d98552ea5a16db16956849aa371db3eb51fd190cce74df356432b410bd64682309d6dedb27c76845daf388557cbac3ca3446ebddef8cd9bb167dc30878d7113b7e168e6f0646beffd77d69d39bad76b47a")
	if err != nil {
		t.Fatal(err)
	}

	cc_ptr, err := common.HexToBytes("0x14416c6963650000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)

	err = Sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		return nil, err
	}

	return keypair_out, nil
}

func TestDeriveKeypairHard(t *testing.T) {
	keypair_out, err := deriveKeypair()
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

	err = Sr25519_derive_keypair_soft(keypair_out, pair_ptr, cc_ptr)
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

	err := Sr25519_keypair_from_seed(keypair_out, seed_ptr)
	if err != nil {
		t.Fatal(err)
	}

	public_ptr := keypair_out[64:]
	pubkey_out := make([]byte, 32)
	cc_ptr := []byte{}

	buf = make([]byte, 32)
	rand.Read(buf)
	cc_ptr = buf

	err = Sr25519_derive_public_soft(pubkey_out, public_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	empty := make([]byte, SR25519_PUBLIC_SIZE)
	if bytes.Equal(pubkey_out, empty) {
		t.Errorf("did not derive keypair from seed")
	}
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

	err = Sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	public_ptr := keypair_out[64:]
	secret_ptr := keypair_out[:64]

	signature_out := make([]byte, 64)
	message_ptr := []byte("this is a message")
	message_length := uint32(len(message_ptr))

	err = Sr25519_sign(signature_out, public_ptr, secret_ptr, message_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	ver, err := Sr25519_verify(signature_out, message_ptr, public_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	if ver != true {
		t.Error("did not verify signature")
	}
}

func TestVerify(t *testing.T) {
	signature_out, err := common.HexToBytes("0xdecef12cf20443e7c7a9d406c237e90bcfcf145860722622f92ebfd5eb4b5b3990b6443934b5cba8f925a0ae75b3a77d35b8490cbb358dd850806e58eaf72904")
	if err != nil {
		t.Fatal(err)
	}

	public_ptr, err := common.HexToBytes("0x741c08a06f41c596608f6774259bd9043304adfa5d3eea62760bd9be97634d63")
	if err != nil {
		t.Fatal(err)
	}

	message_ptr := []byte("this is a message")
	message_length := uint32(len(message_ptr))

	ver, err := Sr25519_verify(signature_out, message_ptr, public_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	if ver != true {
		t.Error("did not verify signature")
	}
}

func TestVrfSignAndVerify(t *testing.T) {
	pair_ptr, err := common.HexToBytes("0x28b0ae221c6bb06856b287f60d7ea0d98552ea5a16db16956849aa371db3eb51fd190cce74df356432b410bd64682309d6dedb27c76845daf388557cbac3ca3446ebddef8cd9bb167dc30878d7113b7e168e6f0646beffd77d69d39bad76b47a")
	if err != nil {
		t.Fatal(err)
	}

	cc_ptr, err := common.HexToBytes("0x14416c6963650000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatal(err)
	}

	keypair_out := make([]byte, SR25519_KEYPAIR_SIZE)

	err = Sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr)
	if err != nil {
		t.Fatal(err)
	}

	keypair_ptr := keypair_out
	out_and_proof_ptr := make([]byte, SR25519_VRF_OUTPUT_SIZE+SR25519_VRF_PROOF_SIZE)
	message_ptr := []byte("helloworld")
	message_length := uint32(len(message_ptr))
	limit_ptr := make([]byte, 32)
	for i, _ := range limit_ptr {
		limit_ptr[i] = 0xff
	}

	ret, err := Sr25519_vrf_sign_if_less(out_and_proof_ptr, keypair_ptr, message_ptr, limit_ptr, message_length)
	if err != nil {
		t.Fatal(err)
	}

	if ret.result != 0 {
		t.Error("result not equal to Sr25519SignatureResult::Ok")
	}
	if !ret.is_less {
		t.Error("is_less not true")
	}

	public_ptr := keypair_out[64:]
	output_ptr := out_and_proof_ptr[:32]
	proof_ptr := out_and_proof_ptr[32:]

	ret2, err := Sr25519_vrf_verify(public_ptr, message_ptr, output_ptr, proof_ptr, message_length)
	if ret2 != 0 {
		t.Errorf("return value not equal to Sr25519SignatureResult::Ok")
	}
}
