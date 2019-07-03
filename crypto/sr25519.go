package crypto

/*
#cgo LDFLAGS:  -Wl,-rpath,${SRCDIR}/libsr25519 -L${SRCDIR}/libsr25519 -lsr25519crust
#include "./sr25519.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

const (
	SR25519_CHAINCODE_SIZE = 32
	SR25519_KEYPAIR_SIZE = 96
	SR25519_PUBLIC_SIZE = 32
	SR25519_SECRET_SIZE = 64
	SR25519_SEED_SIZE = 32
	SR25519_SIGNATURE_SIZE = 64
)

/**
 * Generate a key pair.
 *  keypair_out: keypair [32b key | 32b nonce | 32b public], pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  seed: generation seed - input buffer of SR25519_SEED_SIZE bytes
 */
func sr25519_keypair_from_seed(keypair_out, seed_ptr []byte) error {
	if len(seed_ptr) != SR25519_SEED_SIZE {
		return errors.New("seed_ptr length not equal to SR25519_SEED_SIZE")
	}
 	C.sr25519_keypair_from_seed((*C.uchar)(unsafe.Pointer(&keypair_out[0])), (*C.uchar)(unsafe.Pointer(&seed_ptr[0]))) 
	return nil
}

/**
 * Perform a derivation on a secret
 *  keypair_out: pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  pair_ptr: existing keypair - input buffer of SR25519_KEYPAIR_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
func sr25519_derive_keypair_hard(keypair_out, pair_ptr, cc_ptr []byte) error {
	if len(pair_ptr) != SR25519_KEYPAIR_SIZE {
		return errors.New("pair_ptr length not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(cc_ptr) != SR25519_CHAINCODE_SIZE {
		return errors.New("cc_ptr length not equal to SR25519_CHAINCODE_SIZE")
	}

	C.sr25519_derive_keypair_hard((*C.uchar)(unsafe.Pointer(&keypair_out[0])), (*C.uchar)(unsafe.Pointer(&pair_ptr[0])), (*C.uchar)(unsafe.Pointer(&cc_ptr[0]))) 
	return nil
}
