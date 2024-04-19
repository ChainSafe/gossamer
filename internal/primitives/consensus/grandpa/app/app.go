package app

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

type Public = ed25519.Public

var (
	_ crypto.Public[Signature] = Public{}
)

func NewPublicFromSlice(data []byte) (Public, error) {
	if len(data) != 32 {
		return Public{}, fmt.Errorf("invalid public key from data: %v", data)
	}
	pub := Public{}
	copy(pub[:], data)
	return pub, nil
}

type Signature = ed25519.Signature
