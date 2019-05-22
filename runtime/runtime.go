package runtime

import (
	"log"
	"os"
	"github.com/go-interpreter/wagon/exec"
	"github.com/go-interpreter/wagon/validate"
	"github.com/go-interpreter/wagon/wasm"
)

func Exec(filename string) (interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	module, err := wasm.ReadModule(file, nil)
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