package keyring

import (
	"crypto/rand"
	"fmt"

	ed25519 "golang.org/x/crypto/ed25519"
)

type Ed25519Keypair struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

func NewEd25519Keypair(priv ed25519.PrivateKey) *Ed25519Keypair {
	return &Ed25519Keypair{
		Public:  priv.Public().(ed25519.PublicKey),
		Private: priv,
	}
}

func GenerateKeypair() (*Ed25519Keypair, error) {
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}

	priv := ed25519.NewKeyFromSeed(buf)

	return NewEd25519Keypair(priv), nil
}

func NewEd25519PublicKey(in []byte) (ed25519.PublicKey, error) {
	if len(in) != 32 {
		return nil, fmt.Errorf("cannot create public key: input is not 32 bytes")
	}

	return ed25519.PublicKey(in), nil
}

func NewEd25519PrivateKey(in []byte) (ed25519.PrivateKey, error) {
	if len(in) != 64 {
		return nil, fmt.Errorf("cannot create private key: input is not 64 bytes")
	}

	return ed25519.PrivateKey(in), nil
}

func (kp *Ed25519Keypair) Sign(msg []byte) []byte {
	return ed25519.Sign(kp.Private, msg)
}

func Verify(pub ed25519.PublicKey, msg, sig []byte) bool {
	return ed25519.Verify(pub, msg, sig)
}
