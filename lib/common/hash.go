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

package common

import (
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

const (
	// HashLength is the expected length of the common.Hash type
	HashLength = 32
)

// HexToHash turns a 0x prefixed hex string into type Hash
func HexToHash(in string) (Hash, error) {
	if strings.Compare(in[:2], "0x") != 0 {
		return [32]byte{}, errors.New("could not byteify non 0x prefixed string")
	}
	in = in[2:]
	out, err := hex.DecodeString(in)
	if err != nil {
		return [32]byte{}, err
	}
	var buf = [32]byte{}
	copy(buf[:], out)
	return buf, err
}

// ReadHash reads a 32-byte hash from the reader and returns it
func ReadHash(r io.Reader) (Hash, error) {
	buf := make([]byte, 32)
	_, err := r.Read(buf)
	if err != nil {
		return Hash{}, err
	}
	h := [32]byte{}
	copy(h[:], buf)
	return Hash(h), nil
}
