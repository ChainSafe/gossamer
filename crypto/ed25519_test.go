package crypto

import (
	"reflect"
	"testing"

	ed25519 "golang.org/x/crypto/ed25519"
)

func TestEd25519SignAndVerify(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}

	msg := []byte("helloworld")
	sig := kp.Sign(msg)

	ok := Verify(kp.Public().(*Ed25519PublicKey), msg, sig)
	if !ok {
		t.Fatal("Fail: did not verify ed25519 sig")
	}
}

func TestPublicKeys(t *testing.T) {
	kp, err := GenerateEd25519Keypair()
	if err != nil {
		t.Fatal(err)
	}

	kp2 := NewEd25519Keypair(ed25519.PrivateKey(*(kp.Private().(*Ed25519PrivateKey))))
	if !reflect.DeepEqual(kp.Public(), kp2.Public()) {
		t.Fatal("Fail: pubkeys do not match")
	}
}
