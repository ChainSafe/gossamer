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

package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Header is a state block header
type Header struct {
	ParentHash     common.Hash                `json:"parentHash"`
	Number         *big.Int                   `json:"number"`
	StateRoot      common.Hash                `json:"stateRoot"`
	ExtrinsicsRoot common.Hash                `json:"extrinsicsRoot"`
	Digest         scale.VaryingDataTypeSlice `json:"digest"`
	hash           common.Hash
}

// NewHeader creates a new block header and sets its hash field
func NewHeader(parentHash, stateRoot, extrinsicsRoot common.Hash, number *big.Int, digest scale.VaryingDataTypeSlice) (*Header, error) {
	if number == nil {
		// Hash() will panic if number is nil
		return nil, errors.New("cannot have nil block number")
	}

	bh := &Header{
		ParentHash:     parentHash,
		Number:         number,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         digest,
	}

	bh.Hash()
	return bh, nil
}

// NewEmptyHeader returns a new header with all zero values
func NewEmptyHeader() *Header {
	return &Header{
		Number: big.NewInt(0),
		Digest: NewDigest(),
	}
}

// Exists returns a boolean indicating if the header exists
func (bh *Header) Exists() bool {
	exists := bh != nil
	return exists
}

// Empty returns a boolean indicating is the header is empty
func (bh *Header) Empty() bool {
	if !bh.StateRoot.IsEmpty() || !bh.ExtrinsicsRoot.IsEmpty() || !bh.ParentHash.IsEmpty() {
		return false
	}
	return (bh.Number.Cmp(big.NewInt(0)) == 0 || bh.Number == nil) && len(bh.Digest.Types) == 0
}

// DeepCopy returns a deep copy of the header to prevent side effects down the road
func (bh *Header) DeepCopy() (*Header, error) {
	cp := NewEmptyHeader()
	copy(cp.ParentHash[:], bh.ParentHash[:])
	copy(cp.StateRoot[:], bh.StateRoot[:])
	copy(cp.ExtrinsicsRoot[:], bh.ExtrinsicsRoot[:])

	if bh.Number != nil {
		cp.Number = new(big.Int).Set(bh.Number)
	}

	if len(bh.Digest.Types) > 0 {
		cp.Digest = NewDigest()
		for _, d := range bh.Digest.Types {
			err := cp.Digest.Add(d.Value())
			if err != nil {
				return nil, err
			}
		}
	}

	return cp, nil
}

// String returns the formatted header as a string
func (bh *Header) String() string {
	return fmt.Sprintf("ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v Hash=%s",
		bh.ParentHash, bh.Number, bh.StateRoot, bh.ExtrinsicsRoot, bh.Digest, bh.Hash())
}

// Hash returns the hash of the block header
// If the internal hash field is nil, it hashes the block and sets the hash field.
// If hashing the header errors, this will panic.
func (bh *Header) Hash() common.Hash {
	if bh.hash == [32]byte{} {
		enc, err := scale.Marshal(*bh)
		if err != nil {
			panic(err)
		}

		hash, err := common.Blake2bHash(enc)
		if err != nil {
			panic(err)
		}

		bh.hash = hash
	}

	return bh.hash
}
