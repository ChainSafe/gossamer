package crypto

import (
	sr25519 "github.com/ChainSafe/go-schnorrkel"
)

type Sr25519Keypair struct {
	public *Sr25519PublicKey
	private *Sr25519PrivateKey
}

type Sr25519PublicKey struct {
	key *sr25519.PublicKey
}

type Sr25519PrivateKey struct {
	key *sr25519.SecretKey
}

func NewSr25519Keypair(priv *sr25519.SecretKey) (*Sr25519Keypair, error) {
	pub, err := priv.Public()
	if err != nil {
		return nil, err
	}

	return &Sr25519Keypair {
		public: &Sr25519PublicKey{key: pub},
		private: &Sr25519PrivateKey{key: priv},
	}, nil
}

func GenerateSr25519Keypair() (*Sr25519Keypair, error) {
	priv, pub, err := sr25519.GenerateKeypair()
	if err != nil {
		return nil, err
	}

	return &Sr25519Keypair{
		public: &Sr25519PublicKey{key: pub},
		private: &Sr25519PrivateKey{key: priv},
	}, nil
}

func (kp *Sr25519Keypair) Sign(msg []byte) []byte {
	return nil
}

func (kp *Sr25519Keypair) Public() PublicKey {
	return nil
}

func (kp *Sr25519Keypair) Private() PrivateKey {
	return nil
}

func (k *Sr25519PrivateKey) Sign(msg []byte) []byte {
	return nil

}

func (k *Sr25519PrivateKey) Public() PublicKey {
	return nil

}

func (k *Sr25519PrivateKey) Encode() []byte {
	return nil
}

func (k *Sr25519PrivateKey) Decode(in []byte) error {
	return nil
}

func (k *Sr25519PublicKey) Verify(msg, sig []byte) bool {
	return false
}

func (k *Sr25519PublicKey) Encode() []byte {
	return nil
}

func (k *Sr25519PublicKey) Decode(in []byte) error {
	return nil
}