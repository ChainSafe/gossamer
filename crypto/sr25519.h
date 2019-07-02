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
                  uintptr_t message_length);

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
                    uintptr_t message_length,
                    const uint8_t *public_ptr);

#endif /* __SR25519_INCLUDE_GUARD_H__ */
