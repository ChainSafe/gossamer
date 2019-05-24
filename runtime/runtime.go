package runtime

import (
	"bytes"
	"fmt"
	"io/ioutil"
	scale "github.com/ChainSafe/gossamer/codec"
	exec "github.com/perlin-network/life/exec"
)

type Runtime struct {
	vm *exec.VirtualMachine
	offset int64
}

type Version struct {
	Spec_name         []byte
	Impl_name         []byte
	Authoring_version int32
	Spec_version      int32
	Impl_version      int32
}

func decodeVersion(in []byte) (interface{}, error) {
	buf := &bytes.Buffer{}
	sd := scale.Decoder{buf}
	buf.Write(in)
	var v Version
	output, err := sd.DecodeTuple(&v)
	return output, err
}

func NewRuntime(fp string) (*Runtime, error) {
	input, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: 4096,
		DefaultTableSize:   655360,
		MaxCallStackDepth:  0,
	}, &Resolver{}, nil)

	return &Runtime{
		vm: vm,
		offset: 0,
	}, err
}

func (r *Runtime) Exec() (interface{}, error) {	
	entryID, ok := r.vm.GetFunctionExport("Core_version")
	if !ok {
		panic("entry function not found")
	}

	ret, err := r.vm.Run(entryID, 0, 0)
	if err != nil {
		r.vm.PrintStackTrace()
		return nil, err
	}

	fmt.Printf("return value = %x\n", ret)
	size := int32(ret >> 32)
	offset := int32(ret)
	fmt.Printf("size = %d offset = %x\n", size, offset)

	returnData := r.vm.Memory[offset:offset+size]
	fmt.Printf("version: %v\n", returnData)

	v, err := decodeVersion(returnData)
	version := v.(*Version)
	if err != nil {
		return nil, err
	} else {
		fmt.Printf("return value decoded = %v\n", v)
		fmt.Printf("Spec_name: %s\n", version.Spec_name)
		fmt.Printf("Impl_name: %s\n", version.Impl_name)
		fmt.Printf("Authoring_version: %d\n", version.Authoring_version)
		fmt.Printf("Spec_version: %d\n", version.Spec_version)
		fmt.Printf("Impl_version: %d\n", version.Impl_version)
	}

	return version, nil
}
