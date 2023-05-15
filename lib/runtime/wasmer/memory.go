// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

import (
	"errors"
	"fmt"
	"math"

	"github.com/ChainSafe/gossamer/pkg/wasmergo"
)

var (
	errCantGrowMemory         = errors.New("failed to grow memory")
	errMemoryValueOutOfBounds = errors.New("memory value is out of bounds")
)

// Memory is a thin wrapper around Wasmer memory to support
// Gossamer runtime.Memory interface
type Memory struct {
	memory *wasmergo.Memory
}

func checkBounds(value uint64) (uint32, error) {
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("%w", errMemoryValueOutOfBounds)
	}
	return uint32(value), nil
}

// Data returns the memory's data
func (m Memory) Data() []byte {
	return m.memory.Data()
}

// Length returns the memory's length
func (m Memory) Length() uint32 {
	value, err := checkBounds(uint64(m.memory.DataSize()))
	if err != nil {
		panic(err)
	}
	return value
}

// Grow grows the memory by the given number of pages
func (m Memory) Grow(numPages uint32) error {
	ok := m.memory.Grow(wasmergo.Pages(numPages))
	if !ok {
		return fmt.Errorf("%w: by %d pages", errCantGrowMemory, numPages)
	}
	return nil
}
