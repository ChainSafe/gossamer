package crypto

import (
	"github.com/ChainSafe/gossamer/common"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/blake2b"
)

// KeyType str
type KeyType = string

// Ed25519Type ed25519
const Ed25519Type KeyType = "ed25519"

//Sr25519Type sr25519
const Sr25519Type KeyType = "sr25519"

//Secp256k1Type secp256k1
const Secp256k1Type KeyType = "secp256k1"

// Keypair interface
type Keypair interface {
	Sign(msg []byte) ([]byte, error)
	Public() PublicKey
	Private() PrivateKey
}

// PublicKey interface
type PublicKey interface {
	Verify(msg, sig []byte) (bool, error)
	Encode() []byte
	Decode([]byte) error
	Address() common.Address
	Hex() string
}

// PrivateKey interface
type PrivateKey interface {
	Sign(msg []byte) ([]byte, error)
	Public() (PublicKey, error)
	Encode() []byte
	Decode([]byte) error
}

var ss58Prefix = []byte("SS58PRE")

// PublicKeyToAddress returns an ss58 address given a PublicKey
// see: https://github.com/paritytech/substrate/wiki/External-Address-Format-(SS58)
// also see: https://github.com/paritytech/substrate/blob/master/primitives/core/src/crypto.rs#L275
func PublicKeyToAddress(pub PublicKey) common.Address {
	enc := pub.Encode()
	hasher, err := blake2b.New(64, nil)
	if err != nil {
		return ""
	}
	_, err = hasher.Write(append(ss58Prefix, enc...))
	if err != nil {
		return ""
	}
	checksum := hasher.Sum(nil)
	return common.Address(base58.Encode(append(enc, checksum[:2]...)))
}
