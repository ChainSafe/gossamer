package crypto

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/ChainSafe/gossamer/common"
)

type Keypair interface {
	Sign(msg []byte) ([]byte, error)
	Public() PublicKey
	Private() PrivateKey
}

type PublicKey interface {
	Verify(msg, sig []byte) bool
	Encode() []byte
	Decode([]byte) error
	Address() common.Address
}

type PrivateKey interface {
	Sign(msg []byte) ([]byte, error)
	Public() (PublicKey, error)
	Encode() []byte
	Decode([]byte) error
}

func DecodePrivateKey(in []byte) (PrivateKey, error) {
	priv, err := NewEd25519PrivateKey(in)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func PublicKeyToAddress(pub PublicKey) common.Address {
	enc := pub.Encode()
	return common.Address(base58.Encode(enc))
}