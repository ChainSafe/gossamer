package runtime

import (
	"bytes"
	"fmt"
	"io/ioutil"
	scale "github.com/ChainSafe/gossamer/codec"
	exec "github.com/perlin-network/life/exec"
)

type Version struct {
	Spec_name         []byte
	Impl_name         []byte
	Authoring_version int32
	Spec_version      int32
	Impl_version      int32
}

func padTo8Bytes(in []byte) []byte {
	for {
		if len(in) >= 8 {
			return in
		}
		in = append(in, 0)
	}
}

func scaleDecode(in []byte) (interface{}, error) {
	buf := &bytes.Buffer{}
	sd := scale.Decoder{buf}
	buf.Write(in)
	var v Version
	output, err := sd.DecodeTuple(&v)
	return output, err
}

func int64ToBytes(in int64) []byte {
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = byte(in & 0xff)
		in = in >> 8
	}
	return out
}

func Exec(fp string) (interface{}, error) {	
	input, err := ioutil.ReadFile(fp)
	if err != nil {
		panic(err)
	}

	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: 4096,
		DefaultTableSize:   655360,
		MaxCallStackDepth:  0,
	}, &Resolver{}, nil)
	if err != nil { 
		// if the wasm bytecode is invalid
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport("Core_version")
	if !ok {
		panic("entry function not found")
	}

	ret, err := vm.Run(entryID, 0, 0)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}

	fmt.Printf("return value = %x\n", ret)
	size := int32(ret >> 32)
	offset := int32(ret)
	fmt.Printf("size = %d offset = %x\n", size, offset)

	returnData := vm.Memory[offset:offset+size]
	fmt.Printf("version: %v\n", returnData)

	v, err := scaleDecode(returnData)
	version := v.(*Version)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	} else {
		fmt.Printf("return value decoded = %v\n", v)
		fmt.Printf("Spec_name: %s\n", version.Spec_name)
		fmt.Printf("Impl_name: %s\n", version.Impl_name)
		fmt.Printf("Authoring_version: %d\n", version.Authoring_version)
		fmt.Printf("Spec_version: %d\n", version.Spec_version)
		fmt.Printf("Impl_version: %d\n", version.Impl_version)
	}

	return version, err
}
