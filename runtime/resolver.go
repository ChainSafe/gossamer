package runtime

import (
	"fmt"
	exec "github.com/perlin-network/life/exec"
	log "github.com/inconshreveable/log15"
	trie "github.com/ChainSafe/gossamer/trie"
)

func padToLen(in []byte, l int) []byte {
	for {
		if len(in) == l {
			return in
		}
		in = append(in, 0)
	}
}

type Resolver struct{
	t *trie.Trie
}

// ResolveFunc resolves the imported functions in the runtime
func (r *Resolver) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "env":
		switch field {
		case "ext_get_storage_into":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_get_storage_into")
				keyData := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				valueData := int(uint32(vm.GetCurrentFrame().Locals[2]))
				valueLen := int(uint32(vm.GetCurrentFrame().Locals[3]))
				valueOffset := int(uint32(vm.GetCurrentFrame().Locals[4]))
				log.Debug("[ext_get_storage_into]", "local[0]", keyData, "local[1]", keyLen, "local[2]", valueData, "local[3]", valueLen, "local[4]", valueOffset)
				
				key := vm.Memory[keyData:keyData+keyLen]
				log.Debug("[ext_get_storage_into]", "key", string(key))

				value, err := r.t.Get(key)
				if err != nil {
					return 0
				}

				value = value[valueOffset:]
				paddedVal := padToLen(value, valueLen)
				log.Debug("[ext_get_storage_into]", "value", paddedVal)
				copy(vm.Memory[valueData:valueData+valueLen], value)
				//copy(vm.Memory[valueData:valueData+valueLen], []byte{0xa0, 0, 0, 0})
				return 0
			}
		case "ext_blake2_256":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_blake2_256")
				return 0
			}
		case "ext_blake2_256_enumerated_trie_root":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing ext_blake2_256_enumerated_trie_root")
				return 0
			}
		case "ext_print_utf8":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_print_utf8")
				log.Debug("[ext_print_utf8]", "local[0]", vm.GetCurrentFrame().Locals[0], "local[1]", vm.GetCurrentFrame().Locals[1])
				ptr := int(uint32(vm.GetCurrentFrame().Locals[0]))
				msgLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				msg := vm.Memory[ptr : ptr+msgLen]
				log.Debug("[ext_print_utf8]", "msg", string(msg))
				return 0
			}
		case "ext_print_num":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_print_num")
				log.Debug("[ext_print_num]", "local[0]", vm.GetCurrentFrame().Locals[0])
				return 0
			}
		case "ext_malloc":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_malloc")
				size := vm.GetCurrentFrame().Locals[0]
				log.Debug("[ext_malloc]", "local[0]", size)
				var offset int64 = 1
				return offset
			}
		case "ext_free":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_free")
				return 1
			}
		case "ext_twox_128":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_twox_128")
				return 0
			}
		case "ext_clear_storage":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_clear_storage")
				return 0
			}
		case "ext_set_storage":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_set_storage")
				return 0
			}
		case "ext_exists_storage":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_exists_storage")
				return 0
			}
		case "ext_sr25519_verify":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_sr25519_verify")
				return 0
			}
		case "ext_ed25519_verify":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_ed25519_verify")
				return 0
			}
		case "ext_storage_root":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_storage_root")
				return 0
			}
		case "ext_storage_changes_root":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_storage_changes_root")
				return 0
			}
		case "ext_print_hex":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_print_hex")
				return 0
			}
		default:
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: default")
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
