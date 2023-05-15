//go:build !darwin
// +build !darwin

package wasmergo

// #include <wasmer.h>
// extern uint64_t metering_delegate(enum wasmer_parser_operator_t op);
import "C"

func getPlatformLong(v uint64) C.ulong {
	return C.ulong(v)
}
