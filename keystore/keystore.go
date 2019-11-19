package keystore

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/crypto"
)

type Keystore struct {
	// map of public key encodings to keypairs
	keys map[common.Address]crypto.Keypair
}

func NewKeystore() *Keystore {
	return &Keystore{
		keys: make(map[common.Address]crypto.Keypair),
	}
}

func (ks *Keystore) Insert(kp crypto.Keypair) {
	pub := kp.Public()
	addr := crypto.PublicKeyToAddress(pub)
	ks.keys[addr] = kp
}

func (ks *Keystore) Get(pub common.Address) crypto.Keypair {
	return ks.keys[pub]
}
