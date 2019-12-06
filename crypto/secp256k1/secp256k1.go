package secp256k1

import (
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/crypto"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

type Keypair struct {
	public  *PublicKey
	private *PrivateKey
}

type PublicKey struct {
	key ecdsa.PublicKey
}

type PrivateKey struct {
	key ecdsa.PrivateKey
}

func NewKeypair(pk ecdsa.PrivateKey) *Keypair {
	pub := pk.Public()

	return &Keypair{
		public:  &PublicKey{key: pub.(ecdsa.PublicKey)},
		private: &PrivateKey{key: pk},
	}
}

func GenerateKeypair() (*Keypair, error) {
	priv, err := secp256k1.GenerateKey()
	if err != nil {
		return nil, err
	}

	return NewKeypair(*priv), nil
}

func (kp *Keypair) Sign(msg []byte) ([]byte, error) {
	// TODO: hash input before signing
	return secp256k1.Sign(msg, &kp.private.key)
}

func (kp *Keypair) Public() crypto.PublicKey {
	return kp.public
}
func (kp *Keypair) Private() crypto.PrivateKey {
	return kp.private
}

func (k *PublicKey) Verify(msg, sig []byte) bool {
	// TODO: hash input before verifying
	return secp256k1.VerifySignature(k.Encode(), msg, sig)
}

func (k *PublicKey) Encode() []byte {
	return secp256k1.FromECDSAPub(&k.key)
}

func (k *PublicKey) Decode(in []byte) error {
	pub, err := secp256k1.UnmarshalPubkey(in)
	if err != nil {
		return err
	}
	k.key = *pub
	return nil
}

func (k *PublicKey) Address() common.Address {
	return crypto.PublicKeyToAddress(k)
}

func (k *PublicKey) Hex() string {
	enc := k.Encode()
	h := hex.EncodeToString(enc)
	return "0x" + h
}

func (pk *PrivateKey) Sign(msg []byte) ([]byte, error) {
	// TODO: hash input before signing
	return secp256k1.Sign(msg, &pk.key)
}

func (pk *PrivateKey) Public() (crypto.PublicKey, error) {
	return pk.Public()
}

func (pk *PrivateKey) Encode() []byte {
	return secp256k1.FromECDSA(&pk.key)
}

func (pk *PrivateKey) Decode(in []byte) error {
	key := secp256k1.ToECDSAUnsafe(in)
	pk.key = *key
	return nil
}
