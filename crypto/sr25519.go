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
func sr25519_keypair_from_seed(keypair_out, seed_ptr [] byte) error {
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

/**
 * Perform a derivation on a secret
 *  keypair_out: pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  pair_ptr: existing keypair - input buffer of SR25519_KEYPAIR_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
func sr25519_derive_keypair_soft(keypair_out, pair_ptr, cc_ptr []byte) error {
	if len(pair_ptr) != SR25519_KEYPAIR_SIZE {
		return errors.New("pair_ptr length not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(cc_ptr) != SR25519_CHAINCODE_SIZE {
		return errors.New("cc_ptr length not equal to SR25519_CHAINCODE_SIZE")
	}

	C.sr25519_derive_keypair_soft((*C.uchar)(unsafe.Pointer(&keypair_out[0])), (*C.uchar)(unsafe.Pointer(&pair_ptr[0])), (*C.uchar)(unsafe.Pointer(&cc_ptr[0]))) 
	return nil
}

/**
 * Perform a derivation on a publicKey
 *  pubkey_out: pre-allocated output buffer of SR25519_PUBLIC_SIZE bytes
 *  public_ptr: public key - input buffer of SR25519_PUBLIC_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
func sr25519_derive_public_soft(pubkey_out, public_ptr, cc_ptr []byte) error {
	if len(public_ptr) != SR25519_PUBLIC_SIZE {
		return errors.New("public_ptr length not equal to SR25519_PUBLIC_SIZE")
	}

	if len(cc_ptr) != SR25519_CHAINCODE_SIZE {
		return errors.New("cc_ptr length not equal to SR25519_CHAINCODE_SIZE")
	}

	C.sr25519_derive_public_soft((*C.uchar)(unsafe.Pointer(&pubkey_out[0])), (*C.uchar)(unsafe.Pointer(&public_ptr[0])), (*C.uchar)(unsafe.Pointer(&cc_ptr[0]))) 
	return nil
}

/**
 * Sign a message
 * The combination of both public and private key must be provided.
 * This is effectively equivalent to a keypair.
 *  signature_out: output buffer of SR25519_SIGNATURE_SIZE bytes
 *  public_ptr: public key - input buffer of SR25519_PUBLIC_SIZE bytes
 *  secret_ptr: private key (secret) - input buffer of SR25519_SECRET_SIZE bytes
 *  message_ptr: Arbitrary message; input buffer of size message_length
 *  message_length: Length of a message
 */
 func sr25519_sign(signature_out, public_ptr, secret_ptr, message_ptr []byte, message_length uint32) error {
	if len(public_ptr) != SR25519_PUBLIC_SIZE {
		return errors.New("public_ptr length not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(secret_ptr) != SR25519_SECRET_SIZE {
		return errors.New("secret_ptr length not equal to SR25519_SECRET_SIZE")
	}

	c_signature_out := (*C.uchar)(unsafe.Pointer(&signature_out[0]))
	c_public_ptr := (*C.uchar)(unsafe.Pointer(&public_ptr[0]))
	c_secret_ptr := (*C.uchar)(unsafe.Pointer(&secret_ptr[0]))
	c_message_ptr := (*C.uchar)(unsafe.Pointer(&secret_ptr[0]))
	c_message_length := (C.uintptr_t)(message_length)
	C.sr25519_sign(c_signature_out, c_public_ptr, c_secret_ptr, c_message_ptr, c_message_length) 
	return nil
}

/**
 * Verify a message and its corresponding against a public key;
 *  signature_ptr: verify this signature
 *  message_ptr: Arbitrary message; input buffer of message_length bytes
 *  message_length: Message size
 *  public_ptr: verify with this public key; input buffer of SR25519_PUBLIC_SIZE bytes
 *  returned true if signature is valid, false otherwise
 */
 func sr25519_verify(signature_ptr, message_ptr, public_ptr []byte, message_length uint32) (bool, error) {
	if len(public_ptr) != SR25519_PUBLIC_SIZE {
		return false, errors.New("public_ptr length not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(signature_ptr) != SR25519_SIGNATURE_SIZE {
		return false, errors.New("signature_ptr length not equal to SR25519_SIGNATURE_SIZE")
	}

	c_signature_ptr := (*C.uchar)(unsafe.Pointer(&signature_ptr[0]))
	c_public_ptr := (*C.uchar)(unsafe.Pointer(&public_ptr[0]))
	c_message_ptr := (*C.uchar)(unsafe.Pointer(&message_ptr[0]))
	c_message_length := (C.uintptr_t)(message_length)
	ver := C.sr25519_verify(c_signature_ptr, c_message_ptr, c_message_length, c_public_ptr) 
	return (bool)(ver), nil
}