package crypto

/*
#cgo LDFLAGS:  -Wl,-rpath,${SRCDIR}/sr25519 -L${SRCDIR}/sr25519 -lsr25519crust
#include "./sr25519.h"
*/
import "C"

import (
	"unsafe"
)

func sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr []byte) {
	C.sr25519_derive_keypair_hard((*C.uchar)(unsafe.Pointer(&keypair_out)), (*C.uchar)(unsafe.Pointer(&pair_ptr)), (*C.uchar)(unsafe.Pointer(&cc_ptr))) 
}