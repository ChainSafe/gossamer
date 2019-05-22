package runtime

import (
	"errors"
	"log"
	"os"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"
)

func resolver(name string) (*wasm.Module, error) {
	switch name {
	case "env":
		return wasm.NewModule(), nil
		//switch field {
		// case "ext_get_storage_into":
		// 	return wasm.NewModule(), nil
		// case "ext_blake2_256":
		// 	return wasm.NewModule(), nil
		// case "ext_blake2_256_enumerated_trie_root":
		// 	return wasm.NewModule(), nil
		// case "ext_print_utf8":
		// 	return wasm.NewModule(), nil
		// case "ext_print_num":
		// 	return wasm.NewModule(), nil
		// case "ext_malloc":
		// 	return wasm.NewModule(), nil
		// case "ext_free":
		// 	return wasm.NewModule(), nil
		// case "ext_twox_128":
		// 	return wasm.NewModule(), nil
		// case "ext_clear_storage":
		// 	return wasm.NewModule(), nil
		// case "ext_set_storage":
		// 	return wasm.NewModule(), nil
		// case "ext_exists_storage":
		// 	return wasm.NewModule(), nil
		// case "ext_sr25519_verify":
		// 	return wasm.NewModule(), nil
		// case "ext_ed25519_verify":
		// 	return wasm.NewModule(), nil
		// case "ext_storage_root":
		// 	return wasm.NewModule(), nil
		// case "ext_storage_changes_root":
		// 	return wasm.NewModule(), nil
		// case "ext_print_hex":
		// 	return wasm.NewModule(), nil
	// 	}
	default:
		return nil, errors.New("invalid module")
	}

	return nil, errors.New("invalid module")
}

func Exec(filename string) (interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	module, err := wasm.ReadModule(file, resolver)
	if err != nil {
		log.Fatal(err)
	}
	if err = validate.VerifyModule(module); err != nil {
		log.Fatalf("%s: %v", filename, err)
	}

	//vm, err := exec.NewVM(module, exec.EnableAOT(true))
	vm, err := exec.NewVM(module)
	if err != nil {
		log.Fatalf("%s: %v", filename, err)
	}

	ret, err := vm.ExecCode(0)
	return ret, err
}