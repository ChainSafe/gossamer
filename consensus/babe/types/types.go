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
	"encoding/binary"

	"github.com/ChainSafe/gossamer/crypto/sr25519"
)

// BabeHeader as defined in Polkadot RE Spec, definition 5.10 in section 5.1.4
type BabeHeader struct {
	VrfOutput          [sr25519.VrfOutputLength]byte
	VrfProof           [sr25519.VrfProofLength]byte
	BlockProducerIndex uint64
	SlotNumber         uint64
}

func (bh *BabeHeader) Encode() []byte {
	enc := []byte{}
	enc = append(enc, bh.VrfOutput[:]...)
	enc = append(enc, bh.VrfProof[:]...)
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, bh.BlockProducerIndex)
	enc = append(enc, buf...)
	binary.LittleEndian.PutUint64(buf, bh.SlotNumber)
	enc = append(enc, buf...)
	return enc
}
