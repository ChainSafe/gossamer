// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
)

// Keystore provides key management functionality
type Keystore interface {
	Name() string // TODO: change to typed vars
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

func NewGlobalKeystore() *GlobalKeystore {
	return &GlobalKeystore{}
}
