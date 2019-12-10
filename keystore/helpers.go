package keystore

import (
	"errors"

	"github.com/ChainSafe/gossamer/crypto"
	"github.com/ChainSafe/gossamer/crypto/ed25519"
	"github.com/ChainSafe/gossamer/crypto/sr25519"
)

func PrivateKeyToKeypair(priv crypto.PrivateKey) (kp crypto.Keypair, err error) {
	if key, ok := priv.(*sr25519.PrivateKey); ok {
		kp, err = sr25519.NewKeypairFromPrivate(key)
	} else if key, ok := priv.(*ed25519.PrivateKey); ok {
		kp, err = ed25519.NewKeypairFromPrivate(key)
	} else {
		return nil, errors.New("cannot decode key: invalid key type")
	}

	return kp, err
}

func DecodePrivateKey(in []byte, keytype crypto.KeyType) (priv crypto.PrivateKey, err error) {
	if keytype == crypto.Ed25519Type {
		priv, err = ed25519.NewPrivateKey(in)
	} else if keytype == crypto.Sr25519Type {
		priv, err = sr25519.NewPrivateKey(in)
	} else {
		return nil, errors.New("cannot decode key: invalid key type")
	}

	return priv, err
}
