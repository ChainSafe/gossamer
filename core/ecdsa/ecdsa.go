package ecdsa

// / The ECDSA compressed public key.
type Public [33]byte

// / A signature (a 512-bit value, plus 8 bits for recovery ID).
type Signature [65]byte
