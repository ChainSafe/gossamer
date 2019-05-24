package runtime

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	//"unsafe"
	scale "github.com/ChainSafe/gossamer/codec"
	exec "github.com/perlin-network/life/exec"
)

func padTo8Bytes(in []byte) []byte {
	for {
		if len(in) >= 8 {
			return in
		}
		in = append(in, 0)
	}
}

type Resolver struct{}

func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "env":
		switch field {
		case "ext_get_storage_into":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_get_storage_into")
				return 0
			}
		case "ext_blake2_256":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_blake2_256")
				return 0
			}
		case "ext_blake2_256_enumerated_trie_root":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_blake2_256_enumerated_trie_root")
				return 0
			}
		case "ext_print_utf8":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_print_utf8")
				fmt.Printf("[ext_print_utf8] local[0]: %v local[1]: %v\n", vm.GetCurrentFrame().Locals[0], vm.GetCurrentFrame().Locals[1])
				ptr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				msgLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				msg := vm.Memory[ptr : ptr+msgLen]
				fmt.Printf("[ext_print_utf8] %s\n", string(msg))
				return 0
			}
		case "ext_print_num":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_print_num")
				fmt.Printf("[ext_print_num] local[0]: %v\n", vm.GetCurrentFrame().Locals[0])
				return 0
			}
		case "ext_malloc":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_malloc")
				size := vm.GetCurrentFrame().Locals[0]
				fmt.Printf("[ext_malloc] local[0]: %v\n", size)
				//res := make([]byte, int(size))

				// buffer := bytes.Buffer{}
				// se := scale.Encoder{&buffer}
				// se.Encode(int64(uintptr(unsafe.Pointer(&res))))
				// vm.ReturnValue = int64(binary.LittleEndian.Uint64(padTo8Bytes(buffer.Bytes())))

				vm.ReturnValue = 1049235
				//fmt.Printf("[ext_malloc] Returned value unencoded: %x\n", *(*int64)(unsafe.Pointer(&res)))
				fmt.Printf("[ext_malloc] Returned value: %d\n", vm.ReturnValue)
				return 1049235
			}
		case "ext_free":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_free")
				return 0
			}
		case "ext_twox_128":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_twox_128")
				return 0
			}
		case "ext_clear_storage":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_clear_storage")
				return 0
			}
		case "ext_set_storage":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_set_storage")
				return 0
			}
		case "ext_exists_storage":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_exists_storage")
				return 0
			}
		case "ext_sr25519_verify":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_sr25519_verify")
				return 0
			}
		case "ext_ed25519_verify":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_ed25519_verify")
				return 0
			}
		case "ext_storage_root":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_storage_root")
				return 0
			}
		case "ext_storage_changes_root":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_storage_changes_root")
				return 0
			}
		case "ext_print_hex":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_print_hex")
				return 0
			}
		default:
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "default")
				return 0
			}
		}
	default:
		panic(fmt.Errorf("unknown module: %s\n", module))
	}
}

func (r *Resolver) ResolveGlobal(module, field string) int64 {
	panic("we're not resolving global variables for now")
}

type Version struct {
	//Spec_name         []byte
	// Impl_name         []byte
	Authoring_version int8
	Spec_version      int8
	Impl_version      int8
}

func scaleDecode(in int64) (interface{}, error) {
	buf := &bytes.Buffer{}
	sd := scale.Decoder{buf}
	binary.Write(buf, binary.BigEndian, in)
	//buf.Write(int64ToBytes(in))
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
		fmt.Print(err)
	}
	vm, err := exec.NewVirtualMachine(input, exec.VMConfig{
		DefaultMemoryPages: 4096,
		DefaultTableSize:   655360,
		MaxCallStackDepth:  0,
	}, &Resolver{}, nil)
	if err != nil { // if the wasm bytecode is invalid
		panic(err)
	}

	entryID, ok := vm.GetFunctionExport("Core_version")
	if !ok {
		panic("entry function not found")
	}
	// memory := make([]byte, 1<<8)
	// memAddr := int64(uintptr(unsafe.Pointer(&memory)))
	ret, err := vm.Run(entryID, 0, 0)
	if err != nil {
		vm.PrintStackTrace()
		panic(err)
	}

	fmt.Printf("return value = %v\n", ret)
	output, err := scaleDecode(ret)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	} else {
		fmt.Printf("return value decoded = %v\n", output)
	}

	return output, err
	// entryID, ok = vm.GetFunctionExport("Metadata_metadata")
	// if !ok {
	// 	panic("entry function not found")
	// }

	// ret, err = vm.Run(entryID, 0, 0)
	// if err != nil {
	// 	vm.PrintStackTrace()
	// 	panic(err)
	// }
	// fmt.Printf("return value = %v\n", ret)
	// output, err = scaleDecode(ret)
	// if err != nil {
	// 	fmt.Printf("err: %s\n", err)
	// } else {
	// 	fmt.Printf("return value decoded = %v\n", output)
	// }
}
