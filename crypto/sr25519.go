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
	SR25519_CHAINCODE_SIZE  = 32
	SR25519_KEYPAIR_SIZE    = 96
	SR25519_PUBLIC_SIZE     = 32
	SR25519_SECRET_SIZE     = 64
	SR25519_SEED_SIZE       = 32
	SR25519_SIGNATURE_SIZE  = 64
	SR25519_VRF_OUTPUT_SIZE = 32
	SR25519_VRF_PROOF_SIZE  = 64
)

const (
	Ok = iota + 1
	EquationFalse
	PointDecompressionError
	ScalarFormatError
	BytesLengthError
	NotMarkedSchnorrkel
	MuSigAbsent
	MuSigInconsistent
)

type VrfSignResult struct {
	result  uint
	is_less bool
}

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
	c_message_length := (C.ulong)(message_length)
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
	c_message_length := (C.ulong)(message_length)
	ver := C.sr25519_verify(c_signature_ptr, c_message_ptr, c_message_length, c_public_ptr)
	return (bool)(ver), nil
}

/**
 * Sign the provided message using a Verifiable Random Function and
 * if the result is less than \param limit provide the proof
 * @param out_and_proof_ptr pointer to output array, where the VRF out and proof will be written
 * @param keypair_ptr byte representation of the keypair that will be used during signing
 * @param message_ptr byte array to be signed
 * @param limit_ptr byte array, must be 32 bytes long
 */
func sr25519_vrf_sign_if_less(out_and_proof_ptr, keypair_ptr, message_ptr, limit_ptr []byte, message_length uint32) (C.VrfSignResult, error) {
	if len(out_and_proof_ptr) != SR25519_VRF_OUTPUT_SIZE+SR25519_VRF_PROOF_SIZE {
		return C.VrfSignResult{}, errors.New("out_and_proof_ptr length not equal to SR25519_VRF_OUTPUT_SIZE + SR25519_VRF_PROOF_SIZE")
	}

	if len(keypair_ptr) != SR25519_KEYPAIR_SIZE {
		return C.VrfSignResult{}, errors.New("keypair_ptrllength not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(limit_ptr) != 32 {
		return C.VrfSignResult{}, errors.New("limit_ptr length not equal to 32")
	}

	c_out_and_proof_ptr := (*C.uchar)(unsafe.Pointer(&out_and_proof_ptr[0]))
	c_keypair_ptr := (*C.uchar)(unsafe.Pointer(&keypair_ptr[0]))
	c_limit_ptr := (*C.uchar)(unsafe.Pointer(&limit_ptr[0]))
	c_message_ptr := (*C.uchar)(unsafe.Pointer(&message_ptr[0]))
	c_message_length := (C.ulong)(message_length)
	ret := C.sr25519_vrf_sign_if_less(c_out_and_proof_ptr, c_keypair_ptr, c_message_ptr, c_message_length, c_limit_ptr)
	return ret, nil
}

/**
 * Verify a signature produced by a VRF with its original input and the corresponding proof
 * @param public_key_ptr byte representation of the public key that was used to sign the message
 * @param message_ptr the orignal signed message
 * @param output_ptr the signature
 * @param proof_ptr the proof of the signature
 */
func sr25519_vrf_verify(public_key_ptr, message_ptr, output_ptr, proof_ptr []byte, message_length uint32) (C.Sr25519SignatureResult, error) {
	if len(public_key_ptr) != SR25519_PUBLIC_SIZE {
		return 1<<32 - 1, errors.New("public_key_ptr length not equal to SR25519_KEYPAIR_SIZE")
	}

	if len(output_ptr) != SR25519_VRF_OUTPUT_SIZE {
		return 1<<32 - 1, errors.New("output_ptr length not equal to SR25519_VRF_OUTPUT_SIZE")
	}

	if len(proof_ptr) != SR25519_VRF_PROOF_SIZE {
		return 1<<32 - 1, errors.New("proof_ptr length not equal to SR25519_VRF_PROOF_SIZE")
	}

	c_public_key_ptr := (*C.uchar)(unsafe.Pointer(&public_key_ptr[0]))
	c_output_ptr := (*C.uchar)(unsafe.Pointer(&output_ptr[0]))
	c_proof_ptr := (*C.uchar)(unsafe.Pointer(&proof_ptr[0]))
	c_message_ptr := (*C.uchar)(unsafe.Pointer(&message_ptr[0]))
	c_message_length := (C.ulong)(message_length)
	ret := C.sr25519_vrf_verify(c_public_key_ptr, c_message_ptr, c_message_length, c_output_ptr, c_proof_ptr)
	return ret, nil
}
