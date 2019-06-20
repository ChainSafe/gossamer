package runtime

// #include <stdlib.h>
//
// extern int64_t ext_malloc(void *context, int32_t x);
// extern int64_t ext_print_utf8(void *context, int32_t offset, int32_t size);
import (
	"C"
	"fmt"
	"unsafe"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

//export ext_malloc
func ext_malloc(context unsafe.Pointer, x int32) int64 {
	return 100
}

//export ext_print_utf8
func ext_print_utf8(context unsafe.Pointer, offset int32, size int32) int64 {
	mem := (*[]byte)(context)
	fmt.Println(mem)
	return 1
}

func Exec() ([]byte, error) {
	// Reads the WebAssembly module as bytes.
	bytes, _ := wasm.ReadBytes("polkadot_runtime.compact.wasm")
	
	imports, _ := wasm.NewImports().Append("ext_malloc", ext_malloc, C.ext_malloc)

	// Instantiates the WebAssembly module.
	instance, _ := wasm.NewInstanceWithImports(bytes, imports)
	defer instance.Close()

	version := instance.Exports["Core_version"]

	fmt.Printf("%T", version)
	res, err := version()
	if err != nil {
		return nil, err
	}
	resi := res.ToI64()

	offset := int32(resi >> 32)
	length :=  int32(resi)
	fmt.Printf("offset %d length %d", offset, length)
	return instance.Memory.Data()[offset:offset+length], err
}