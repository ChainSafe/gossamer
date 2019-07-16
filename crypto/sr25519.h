#ifndef __SR25519_INCLUDE_GUARD_H__
#define __SR25519_INCLUDE_GUARD_H__

/* Generated with cbindgen:0.9.0 */

/* THIS FILE WAS AUTOMATICALLY GENERATED. DO NOT EDIT. Ref: https://github.com/Warchant/sr25519-crust */

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#define SR25519_CHAINCODE_SIZE 32

#define SR25519_KEYPAIR_SIZE 96

#define SR25519_PUBLIC_SIZE 32

#define SR25519_SECRET_SIZE 64

#define SR25519_SEED_SIZE 32

#define SR25519_SIGNATURE_SIZE 64

#define SR25519_VRF_OUTPUT_SIZE 32

#define SR25519_VRF_PROOF_SIZE 64

typedef enum Sr25519SignatureResult {
  Ok,
  EquationFalse,
  PointDecompressionError,
  ScalarFormatError,
  BytesLengthError,
  NotMarkedSchnorrkel,
  MuSigAbsent,
  MuSigInconsistent,
} Sr25519SignatureResult;

typedef struct VrfSignResult {
  Sr25519SignatureResult result;
  bool is_less;
} VrfSignResult;

/**
 * Perform a derivation on a secret
 *  keypair_out: pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  pair_ptr: existing keypair - input buffer of SR25519_KEYPAIR_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
void sr25519_derive_keypair_hard(uint8_t *keypair_out,
                                 const uint8_t *pair_ptr,
                                 const uint8_t *cc_ptr);

/**
 * Perform a derivation on a secret
 *  keypair_out: pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  pair_ptr: existing keypair - input buffer of SR25519_KEYPAIR_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
void sr25519_derive_keypair_soft(uint8_t *keypair_out,
                                 const uint8_t *pair_ptr,
                                 const uint8_t *cc_ptr);

/**
 * Perform a derivation on a publicKey
 *  pubkey_out: pre-allocated output buffer of SR25519_PUBLIC_SIZE bytes
 *  public_ptr: public key - input buffer of SR25519_PUBLIC_SIZE bytes
 *  cc_ptr: chaincode - input buffer of SR25519_CHAINCODE_SIZE bytes
 */
void sr25519_derive_public_soft(uint8_t *pubkey_out,
                                const uint8_t *public_ptr,
                                const uint8_t *cc_ptr);

/**
 * Generate a key pair.
 *  keypair_out: keypair [32b key | 32b nonce | 32b public], pre-allocated output buffer of SR25519_KEYPAIR_SIZE bytes
 *  seed: generation seed - input buffer of SR25519_SEED_SIZE bytes
 */
void sr25519_keypair_from_seed(uint8_t *keypair_out,
                               const uint8_t *seed_ptr);

/**
 * Sign a message
 * The combination of both public and private key must be provided.
 * This is effectively equivalent to a keypair.
 *  signature_out: output buffer of ED25519_SIGNATURE_SIZE bytes
 *  public_ptr: public key - input buffer of SR25519_PUBLIC_SIZE bytes
 *  secret_ptr: private key (secret) - input buffer of SR25519_SECRET_SIZE bytes
 *  message_ptr: Arbitrary message; input buffer of size message_length
 *  message_length: Length of a message
 */
void sr25519_sign(uint8_t *signature_out,
                  const uint8_t *public_ptr,
                  const uint8_t *secret_ptr,
                  const uint8_t *message_ptr,
                  unsigned long message_length);

/**
 * Verify a message and its corresponding against a public key;
 *  signature_ptr: verify this signature
 *  message_ptr: Arbitrary message; input buffer of message_length bytes
 *  message_length: Message size
 *  public_ptr: verify with this public key; input buffer of SR25519_PUBLIC_SIZE bytes
 *  returned true if signature is valid, false otherwise
 */
bool sr25519_verify(const uint8_t *signature_ptr,
                    const uint8_t *message_ptr,
                    unsigned long message_length,
                    const uint8_t *public_ptr);

/**
 * Sign the provided message using a Verifiable Random Function and
 * if the result is less than \param limit provide the proof
 * @param out_and_proof_ptr pointer to output array, where the VRF out and proof will be written
 * @param keypair_ptr byte representation of the keypair that will be used during signing
 * @param message_ptr byte array to be signed
 * @param limit_ptr byte array, must be 32 bytes long
 */
VrfSignResult sr25519_vrf_sign_if_less(uint8_t *out_and_proof_ptr,
                                       const uint8_t *keypair_ptr,
                                       const uint8_t *message_ptr,
                                       unsigned long message_length,
                                       const uint8_t *limit_ptr);

/**
 * Verify a signature produced by a VRF with its original input and the corresponding proof
 * @param public_key_ptr byte representation of the public key that was used to sign the message
 * @param message_ptr the orignal signed message
 * @param output_ptr the signature
 * @param proof_ptr the proof of the signature
 */
Sr25519SignatureResult sr25519_vrf_verify(const uint8_t *public_key_ptr,
                                          const uint8_t *message_ptr,
                                          unsigned long message_length,
                                          const uint8_t *output_ptr,
                                          const uint8_t *proof_ptr);

#endif /* __SR25519_INCLUDE_GUARD_H__ */