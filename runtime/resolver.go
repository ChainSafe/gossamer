package runtime

import (
	"fmt"
	exec "github.com/perlin-network/life/exec"
)

var offset int64 = 0

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
				fmt.Printf("[ext_print_num] local[0]: %d\n", vm.GetCurrentFrame().Locals[0])
				return 0
			}
		case "ext_malloc":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_malloc")
				size := vm.GetCurrentFrame().Locals[0]
				fmt.Printf("[ext_malloc] local[0]: %v\n", size)
				offset = offset + size
				fmt.Printf("offset: %d\n", offset)
				return offset
			}
		case "ext_free":
			return func(vm *exec.VirtualMachine) int64 {
				fmt.Printf("executing: %s\n", "ext_free")
				addr := vm.GetCurrentFrame().Locals[0]
				offset = addr
				return offset
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