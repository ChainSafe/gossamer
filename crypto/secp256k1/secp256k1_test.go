package secp256k1

import (
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("borkbork")
	sig, err := kp.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}

	ok := kp.public.Verify(msg, sig)
	if !ok {
		t.Fatal("did not verify :(")
	}
}

func TestPublicKeys(t *testing.T) {

}

func TestEncodeAndDecodePriv(t *testing.T) {

}

func TestEncodeAndDecodePub(t *testing.T) {

}
