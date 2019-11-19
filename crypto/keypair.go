package crypto

import (
	"github.com/ChainSafe/gossamer/cmd/utils"
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
	Hex() string
}

type PrivateKey interface {
	Sign(msg []byte) ([]byte, error)
	Public() (PublicKey, error)
	Encode() []byte
	Decode([]byte) error
}

func DecodePrivateKey(in []byte, keytype string) (priv PrivateKey, err error) {
	if keytype == utils.Ed25519KeyType {
		priv, err = NewEd25519PrivateKey(in)
	} else {
		priv, err = NewSr25519PrivateKey(in)
	}

	return priv, err
}
