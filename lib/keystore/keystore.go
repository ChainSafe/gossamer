// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: Apache-2.0

package keystore

import (
	"errors"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
)

// Name represents a defined keystore name
type Name string

//nolint
var (
	BabeName Name = "babe"
	GranName Name = "gran"
	AccoName Name = "acco"
	AuraName Name = "aura"
	ImonName Name = "imon"
	AudiName Name = "audi"
	DumyName Name = "dumy"
)

// Keystore provides key management functionality
type Keystore interface {
	Name() Name
	Type() crypto.KeyType
	Insert(kp crypto.Keypair)
	GetKeypairFromAddress(pub common.Address) crypto.Keypair
	GetKeypair(pub crypto.PublicKey) crypto.Keypair
	PublicKeys() []crypto.PublicKey
	Keypairs() []crypto.Keypair
	Size() int
}

// GlobalKeystore defines the various keystores used by the node
type GlobalKeystore struct {
	Babe Keystore
	Gran Keystore
	Acco Keystore
	Aura Keystore
	Imon Keystore
	Audi Keystore
	Dumy Keystore
}

// NewGlobalKeystore returns a new GlobalKeystore
func NewGlobalKeystore() *GlobalKeystore {
	return &GlobalKeystore{
		Babe: NewBasicKeystore(BabeName, crypto.Sr25519Type),
		Gran: NewBasicKeystore(GranName, crypto.Ed25519Type),
		Acco: NewGenericKeystore(AccoName), // TODO: which type is used? can an account be either type? (#1872)
		Aura: NewBasicKeystore(AuraName, crypto.Sr25519Type),
		Imon: NewBasicKeystore(ImonName, crypto.Sr25519Type),
		Audi: NewBasicKeystore(AudiName, crypto.Sr25519Type),
		Dumy: NewGenericKeystore(DumyName),
	}
}

// GetKeystore returns a keystore given its name
func (k *GlobalKeystore) GetKeystore(name []byte) (Keystore, error) {
	nameStr := Name(name)
	switch nameStr {
	case BabeName:
		return k.Babe, nil
	case GranName:
		return k.Gran, nil
	case AccoName:
		return k.Acco, nil
	case AuraName:
		return k.Aura, nil
	case ImonName:
		return k.Imon, nil
	case AudiName:
		return k.Audi, nil
	case DumyName:
		return k.Dumy, nil
	default:
		return nil, errors.New("invalid keystore name")
	}
}
