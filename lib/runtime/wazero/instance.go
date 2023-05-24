package wazero_runtime

import (
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Instance backed by wazero.Runtime
type Instance struct {
	Runtime   wazero.Runtime
	Module    api.Module
	Allocator *runtime.FreeingBumpHeapAllocator
}

// // NewInstance instantiates a runtime from raw wasm bytecode
// func NewInstance(code []byte, cfg Config) (instance *Instance, err error) {
// 	return &Instance{}
// }
