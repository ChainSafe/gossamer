package keystore

import (
	"github.com/ChainSafe/gossamer/crypto"
)

type Keystore struct {
	keys map[crypto.PublicKey]crypto.Keypair
}

func NewKeystore() *Keystore {
	return &Keystore{
		keys: make(map[crypto.PublicKey]crypto.Keypair),
	}
}

func (ks *Keystore) Insert(kp crypto.Keypair) {
	pub := kp.Public()
	ks.keys[pub] = kp
}

func (ks *Keystore) Get(pub crypto.PublicKey) crypto.Keypair {
	return ks.keys[pub]
}
