package secp256k1

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/crypto"
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
)

const SignatureLength = 65

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
		public:  &PublicKey{key: *pub.(*ecdsa.PublicKey)},
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
	hash, err := common.Blake2bHash(msg)
	if err != nil {
		return nil, err
	}
	return secp256k1.Sign(hash[:], &kp.private.key)
}

func (kp *Keypair) Public() crypto.PublicKey {
	return kp.public
}
func (kp *Keypair) Private() crypto.PrivateKey {
	return kp.private
}

func (k *PublicKey) Verify(msg, sig []byte) bool {
	if len(sig) != SignatureLength {
		fmt.Println("wrong sig length")
		return false
	}

	hash, err := common.Blake2bHash(msg)
	if err != nil {
		fmt.Println("could not hash")
		return false
	}
	return secp256k1.VerifySignature(k.Encode(), hash[:], sig)
}

func (k *PublicKey) Encode() []byte {
	return secp256k1.CompressPubkey(&k.key)
}

func (k *PublicKey) Decode(in []byte) error {
	pub, err := secp256k1.DecompressPubkey(in)
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
	hash, err := common.Blake2bHash(msg)
	if err != nil {
		return nil, err
	}
	return secp256k1.Sign(hash[:], &pk.key)
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
