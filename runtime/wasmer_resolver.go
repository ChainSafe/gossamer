package runtime

// // #include <stdlib.h>
// //
// // extern int64_t ext_malloc(void *context, int32_t x);
// // extern int64_t ext_print_utf8(void *context, int32_t offset, int32_t size);
// import (
// 	"C"
// 	"fmt"
// 	"unsafe"
// )

// //export ext_malloc
// func ext_malloc(context unsafe.Pointer, x int32) int64 {
// 	return 100
// }

// //export ext_print_utf8
// func ext_print_utf8(context unsafe.Pointer, offset int32, size int32) int64 {
// 	mem := (*[]byte)(context)
// 	fmt.Println(mem)
// 	return 1
// }

