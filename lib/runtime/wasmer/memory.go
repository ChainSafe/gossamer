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

package wasmer

import "C"

import (
	"errors"

	"github.com/wasmerio/wasmer-go/wasmer"
)

// Memory is a thin wrapper around Wasmer memory to support
// Gossamer runtime.Memory interface
type Memory struct {
	memory *wasmer.Memory
}

// Data returns the memory's data
func (m Memory) Data() []byte {
	return m.memory.Data()
}

// Length returns the memory's length
func (m Memory) Length() uint32 {
	return uint32(m.memory.DataSize())
}

// Grow grows the memory by the given number of pages
func (m Memory) Grow(numPages uint32) error {
	ok := m.memory.Grow(wasmer.Pages(numPages))
	if !ok {
		return errors.New("failed to grow memory")
	}
	return nil
}
