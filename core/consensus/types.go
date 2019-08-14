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

package consensus

import (
	"github.com/ChainSafe/gossamer/common"
	"math/big"
)

//type used to store an authority Id
type authorityId [32]byte

//length of vrf output
type VRFOutput [32]byte

//signature size
type Signature [32]byte

type Hash [32]byte


//Justified Header
type JustifiedHeader struct {
	BlockHeader		common.BlockHeader
	Justification	[64]byte
	AuthorityIds	[]authorityId

}

type Block struct {
	SlotNumber		*big.Int
	PreviousHash	Hash
	VrfOutput		VRFOutput
	Transactions 	[]Transaction
	Signature		Signature
	BlockNumber		*big.Int
	Hash			Hash			

}

//generalize this into extrinsic interface later
type Transaction struct {
	
}