package crypto

import (
	sr25519 "github.com/ChainSafe/go-schnorrkel"
)

var SigningContext = []byte("substrate")

type Sr25519Keypair struct {
	public  *Sr25519PublicKey
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

	return &Sr25519Keypair{
		public:  &Sr25519PublicKey{key: pub},
		private: &Sr25519PrivateKey{key: priv},
	}, nil
}

func GenerateSr25519Keypair() (*Sr25519Keypair, error) {
	priv, pub, err := sr25519.GenerateKeypair()
	if err != nil {
		return nil, err
	}

	return &Sr25519Keypair{
		public:  &Sr25519PublicKey{key: pub},
		private: &Sr25519PrivateKey{key: priv},
	}, nil
}

func (kp *Sr25519Keypair) Sign(msg []byte) ([]byte, error) {
	return kp.private.Sign(msg)
}

func (kp *Sr25519Keypair) Public() PublicKey {
	return kp.public
}

func (kp *Sr25519Keypair) Private() PrivateKey {
	return kp.private
}

func (k *Sr25519PrivateKey) Sign(msg []byte) ([]byte, error) {
	t := sr25519.NewSigningContext(SigningContext, msg)
	sig, err := k.key.Sign(t)
	if err != nil {
		return nil, err
	}
	enc := sig.Encode()
	return enc[:], nil
}

func (k *Sr25519PrivateKey) Public() PublicKey {
	pub, _ := k.key.Public()
	return &Sr25519PublicKey{key: pub}
}

func (k *Sr25519PrivateKey) Encode() []byte {
	enc := k.key.Encode()
	return enc[:]
}

func (k *Sr25519PrivateKey) Decode(in []byte) error {
	b := [32]byte{}
	copy(b[:], in)
	return k.key.Decode(b)
}

func (k *Sr25519PublicKey) Verify(msg, sig []byte) bool {
	b := [64]byte{}
	copy(b[:], sig)

	s := &sr25519.Signature{}
	err := s.Decode(b)
	if err != nil {
		return false
	}

	t := sr25519.NewSigningContext(SigningContext, msg)
	return k.key.Verify(s, t)
}

func (k *Sr25519PublicKey) Encode() []byte {
	enc := k.key.Encode()
	return enc[:]
}

func (k *Sr25519PublicKey) Decode(in []byte) error {
	b := [32]byte{}
	copy(b[:], in)
	return k.key.Decode(b)
}
