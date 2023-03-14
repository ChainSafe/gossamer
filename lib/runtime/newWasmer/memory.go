package newWasmer

import (
	"errors"
	"fmt"

	wasmgo "github.com/wasmerio/wasmer-go/wasmer"
)

var errGrowMemory = errors.New("failed to grow memory")

// Memory is a thin wrapper around Wasmer memory to support
// Gossamer runtime.Memory interface
type Memory struct {
	memory *wasmgo.Memory
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
	ok := m.memory.Grow(wasmgo.Pages(numPages))
	if !ok {
		return fmt.Errorf("%w", errGrowMemory)
	}
	return nil
}
