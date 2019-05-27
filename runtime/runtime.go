package runtime

import (
	"bytes"
	"errors"
	//"fmt"
	"io/ioutil"
	scale "github.com/ChainSafe/gossamer/codec"
	exec "github.com/perlin-network/life/exec"
)

var (
	DEFAULT_MEMORY_PAGES = 4096
	DEFAULT_TABLE_SIZE = 655360
	DEFAULT_MAX_CALL_STACK_DEPTH = 0
)

type Runtime struct {
	vm *exec.VirtualMachine
	// TODO: memory management on top of wasm memory buffer
}

type Version struct {
	Spec_name         []byte
	Impl_name         []byte
	Authoring_version int32
	Spec_version      int32
	Impl_version      int32
}

func NewRuntime(fp string) (*Runtime, error) {
	input, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: DEFAULT_MEMORY_PAGES,
		DefaultTableSize:   DEFAULT_TABLE_SIZE,
		MaxCallStackDepth:  DEFAULT_MAX_CALL_STACK_DEPTH,
	}, &Resolver{}, nil)

	return &Runtime{
		vm: vm,
	}, err
}

func (r *Runtime) Exec(function string) (interface{}, error) {	
	entryID, ok := r.vm.GetFunctionExport(function)
	if !ok {
		return nil, errors.New("entry function not found")
	}

	ret, err := r.vm.Run(entryID, 0, 0)
	if err != nil {
		return nil, err
	}

	switch function {
	case "Core_version":	
		// ret is int64; top 4 bytes are the size of the returned data and bottom 4 bytes are 
		// the offset in the wasm memory buffer
		size := int32(ret >> 32)
		offset := int32(ret)
		returnData := r.vm.Memory[offset:offset+size]
		return decodeVersion(returnData)
	case "Core_authorities":
		return nil, nil
	case "Core_execute_block":
		return nil, nil
	case "Core_initialise_block":
		return nil, nil
	default:
		return nil, nil
	}
}

func decodeVersion(in []byte) (interface{}, error) {
	buf := &bytes.Buffer{}
	sd := scale.Decoder{buf}
	buf.Write(in)
	var v Version
	output, err := sd.DecodeTuple(&v)
	return output, err
}