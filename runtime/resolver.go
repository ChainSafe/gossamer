package runtime

import (
	"encoding/binary"
	"fmt"

	common "github.com/ChainSafe/gossamer/common"
	trie "github.com/ChainSafe/gossamer/trie"
	log "github.com/inconshreveable/log15"
	exec "github.com/perlin-network/life/exec"
)

type Resolver struct {
	trie *trie.Trie
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
				log.Debug("[ext_get_storage_into]", "keyData", keyData, "keyLen", keyLen, "valueData", valueData, "valueLen", valueLen, "valueOffset", valueOffset)

				key := vm.Memory[keyData : keyData+keyLen]
				log.Debug("[ext_get_storage_into]", "key", string(key), "byteskey", key)

				value, err := r.trie.Get(key)
				if err != nil {
					log.Error("[ext_get_storage_into]", "error", err)
					return 0
				}

				if valueLen == 0 {
					return 0
				}

				value = value[valueOffset:]
				copy(vm.Memory[valueData:valueData+valueLen], value)

				log.Debug("[ext_get_storage_into]", "value", vm.Memory[valueData:valueData+valueLen])
				ret := int64(binary.LittleEndian.Uint64(common.AppendZeroes(vm.Memory[valueData:valueData+valueLen], 8)))
				log.Debug("[ext_get_storage_into]", "returnvalue", ret)
				return ret
			}
		case "ext_blake2_256":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_blake2_256")
				data := int(uint32(vm.GetCurrentFrame().Locals[0]))
				length := int(uint32(vm.GetCurrentFrame().Locals[1]))
				out := int(uint32(vm.GetCurrentFrame().Locals[2]))
				hash, err := common.Blake2bHash(vm.Memory[data:data+length])
				if err != nil {
					log.Error("[ext_blake2_256]", "error", err)
					return 0
				}
				copy(vm.Memory[out:out+32], hash[:])
				return 1
			}
		case "ext_blake2_256_enumerated_trie_root":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_blake2_256_enumerated_trie_root")
				t := &trie.Trie{}

				valuesData := int(uint32(vm.GetCurrentFrame().Locals[0]))
				lensData := int(uint32(vm.GetCurrentFrame().Locals[1]))
				lensLen := int(uint32(vm.GetCurrentFrame().Locals[2]))
				result := int(uint32(vm.GetCurrentFrame().Locals[3]))

				var pos int32 = 0
				for i := 0; i < lensLen; i++ {
					valueLenBytes := vm.Memory[lensData + i*32 : lensData + (i+1)*32]
					valueLen := int32(binary.LittleEndian.Uint32(valueLenBytes))
					value := vm.Memory[valuesData + int(pos) : valuesData + int(pos) + int(valueLen)]
					pos += valueLen

					// todo: do we get key/value pairs or just values??
					err := t.Put(value, nil)
					if err != nil {
						log.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
						return 0
					}
				}

				root, err := t.Hash()
				if err != nil {
					log.Error("[ext_blake2_256_enumerated_trie_root]", "error", err)
					return 0
				}

				copy(vm.Memory[result:result+32], root[:])
				return 1
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
				keyData := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))

				key := vm.Memory[keyData:keyData+keyLen]
				err := r.trie.Delete(key)
				if err != nil {
					log.Error("[ext_storage_root]", "error", err)
					return 0
				}

				return 1
			}
		case "ext_set_storage":
			return func(vm *exec.VirtualMachine) int64 {
				log.Debug("executing: ext_set_storage")

				keyData := int(uint32(vm.GetCurrentFrame().Locals[0]))
				keyLen := int(uint32(vm.GetCurrentFrame().Locals[1]))
				valueData := int(uint32(vm.GetCurrentFrame().Locals[2]))
				valueLen := int(uint32(vm.GetCurrentFrame().Locals[3]))

				key := vm.Memory[keyData:keyData+keyLen]
				val := vm.Memory[valueData:valueData+valueLen]
				err := r.trie.Put(key, val)
				if err != nil {
					log.Error("[ext_set_storage]", "error", err)
					return 0
				}

				return 1
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
				resultPtr := int(uint32(vm.GetCurrentFrame().Locals[0]))

				root, err := r.trie.Hash()
				if err != nil {
					log.Error("[ext_storage_root]", "error", err)
				}

				copy(vm.Memory[resultPtr:resultPtr+32], root[:])
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
				offset := int(uint32(vm.GetCurrentFrame().Locals[0]))
				size := int(uint32(vm.GetCurrentFrame().Locals[1]))
				log.Debug("[ext_print_hex]", "message", fmt.Sprintf("%x", vm.Memory[offset:offset+size]))
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
