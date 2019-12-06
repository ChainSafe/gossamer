package secp256k1

import (
	"crypto/ecdsa"

	"github.com/ChainSafe/gossamer/crypto"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

type Keypair struct {
	public  *PublicKey
	private *PrivateKey
}

type PublicKey *ecdsa.PublicKey
type PrivateKey *ecdsa.PrivateKey

func NewKeypair(pk PrivateKey) (*Keypair, error) {
	pub, err := pk.Public()
	if err != nil {
		return nil, err
	}

	return &Keypair{
		public: &pub,
		private: &pk,
	}, nil
}

func GenerateKeypair() (*Keypair, error) {
	priv, err := secp256k1.GenerateKey()
	if err != nil {
		return nil, err
	}

	return NewKeypair(PrivateKey(priv))
}

func (kp *Keypair) Sign(msg []byte) ([]byte, error) {
	return kp.private.Sign(msg)
}

func (kp *Keypair) Public() crypto.PublicKey {
	return kp.public
}
func (kp *Keypair) Private() crypto.PrivateKey {
	return kp.private
}

// PublicKey methods
func (k *PublicKey) Verify(msg, sig []byte) bool {
	return secp.VerifySignature(k, msg, sig)
}

func (k *PublicKey) Encode() []byte {}
func (k *PublicKey) Decode([]byte) error {}
func (k *PublicKey) Address() common.Address {}
func (k *PublicKey) Hex() string {}

func (pk *PrivateKey) Sign(msg []byte) ([]byte, error) {
	return secp.Sign(msg, pk.private)
}

func (pk *PrivateKey) Public() (PublicKey, error) {
	kp := NewSecpKeypair(pk)
	return kp.private, nil
}

func (pk *PrivateKey) Encode() []byte {}
func (pk *PrivateKey) Decode([]byte) error {} 