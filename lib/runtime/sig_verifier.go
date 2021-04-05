package runtime

import (
	ced25519 "crypto/ed25519"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	"github.com/ChainSafe/go-schnorrkel"
	"github.com/hdevalence/ed25519consensus"
)

// Signature ...
type Signature struct {
	PubKey    []byte
	Sign      []byte
	Msg       []byte
	KeyTypeID crypto.KeyType
}

// SignatureVerifier ...
type SignatureVerifier struct {
	ed25519Batch *ed25519consensus.BatchVerifier
	sr25519Batch *schnorrkel.BatchVerifier
	init         bool // Indicates whether the batch processing is started.
	sync.RWMutex
}

// NewSignatureVerifier initializes SignatureVerifier which does background verification of signatures.
// Start() is called to start the verification process.
// Finish() is called to stop the verification process.
// Signatures can be added to the batch using Add().
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{}
}

// Start signature verification in batch.
func (sv *SignatureVerifier) Start() {
	// Update the init state.
	sv.Lock()
	sv.init = true
	sv.Unlock()

	ed25519Batch := ed25519consensus.NewBatchVerifier()
	sv.ed25519Batch = &ed25519Batch
	sv.sr25519Batch = schnorrkel.NewBatchVerifier()
}

// IsStarted ...
func (sv *SignatureVerifier) IsStarted() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.init
}

// Add ...
func (sv *SignatureVerifier) Add(s *Signature) error {
	if !sv.IsStarted() {
		return fmt.Errorf("verifier has not yet been started")
	}

	sv.Lock()
	defer sv.Unlock()

	switch s.KeyTypeID {
	case crypto.Ed25519Type:
		pubKey, err := ed25519.NewPublicKey(s.PubKey)
		if err != nil {
			return fmt.Errorf("invalid ed25519 public key: %w", err)
		}

		sv.ed25519Batch.Add(ced25519.PublicKey(*pubKey), s.Msg, s.Sign)
	case crypto.Sr25519Type:
		pubKey, err := sr25519.NewPublicKey(s.PubKey)
		if err != nil {
			return fmt.Errorf("invalid sr25519 public key: %w", err)
		}

		b := [sr25519.SignatureLength]byte{}
		copy(b[:], s.Sign)

		sig := &schnorrkel.Signature{}
		err = sig.Decode(b)
		if err != nil {
			return fmt.Errorf("invalid sr25519 signature: %w", err)
		}

		t := schnorrkel.NewSigningContext(sr25519.SigningContext, s.Msg)
		return sv.sr25519Batch.Add(t, sig, pubKey.Key())
	default:
		return fmt.Errorf("invalid signature type added to batch verifier: type=%s", s.KeyTypeID)
	}

	return nil
}

// Reset reset the signature verifier for reuse.
func (sv *SignatureVerifier) reset() {
	sv.Lock()
	defer sv.Unlock()

	sv.ed25519Batch = nil
	sv.sr25519Batch = nil
	sv.init = false
}

// Finish waits till batch is finished. Returns true if all the signatures are valid, Otherwise returns false.
func (sv *SignatureVerifier) Finish() bool {
	defer sv.reset()
	return sv.ed25519Batch.Verify() && sv.sr25519Batch.Verify()
}
