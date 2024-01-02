package app

import (
	applicationcrypto "github.com/ChainSafe/gossamer/internal/primitives/application-crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

type Public struct {
	ed25519.Public
}

var (
	_ crypto.Public                              = Public{}
	_ applicationcrypto.RuntimePublic[Signature] = Public{}
)

// impl $crate::CryptoType for Public {
// type Pair = Pair;
// }
// impl $crate::AppCrypto for Public {
// type Public = Public;
// type Pair = Pair;
// type Signature = Signature;
// const ID:$crate::KeyTypeId = GRANDPA;
// const CRYPTO_ID:$crate::CryptoTypeId = (ed25519::CRYPTO_ID);
// }
// impl $crate::ByteArray for Public {
// const LEN:usize =  <ed25519::Public>::LEN;
// }
func (p Public) ToRawVec() []byte {
	return p.Public[:]
}

// impl $crate::Public for Public{}

// impl $crate::AppPublic for Public {
// type Generic = ed25519::Public;
// }
func (p Public) Verify(msg []byte, signature Signature) bool {
	// TODO: implement this!
	// reutrn p.Public
	return true
}

func (p Public) String() string {
	return string(p.Public[:])
}

type Signature struct {
	ed25519.Signature
}

func (s Signature) String() string {
	return string(s.Signature[:])
}
