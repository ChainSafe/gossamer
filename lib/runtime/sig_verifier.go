package runtime

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	log "github.com/ChainSafe/log15"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
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
	batch   []*Signature
	init    bool // Indicates whether the batch processing is started.
	invalid bool // Set to true if any signature verification fails.
	closeCh chan struct{}
	sync.RWMutex
	sync.Once
	sync.WaitGroup
}

// NewSignatureVerifier initializes SignatureVerifier which does background verification of signatures.
// Start() is called to start the verification process.
// Finish() is called to stop the verification process.
// Signatures can be added to the batch using Add().
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{
		batch:   make([]*Signature, 0),
		init:    false,
		invalid: false,
		RWMutex: sync.RWMutex{},
		closeCh: make(chan struct{}),
	}
}

// Start signature verification in batch.
func (sv *SignatureVerifier) Start() {
	// Update the init state.
	sv.Lock()
	sv.init = true
	sv.Unlock()

	sv.WaitGroup.Add(1)

	go func() {
		defer sv.Done()
		for {
			select {
			case <-sv.closeCh:
				return
			default:
				sign := sv.Remove()
				if sign == nil {
					continue
				}
				err := sign.verify()
				if err != nil {
					log.Error("[ext_crypto_start_batch_verify_version_1]", "error", err)
					sv.Invalid()
					return
				}
			}
		}
	}()
}

// IsStarted ...
func (sv *SignatureVerifier) IsStarted() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.init
}

// IsInvalid ...
func (sv *SignatureVerifier) IsInvalid() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.invalid
}

// Invalid ...
func (sv *SignatureVerifier) Invalid() {
	sv.RLock()
	defer sv.RUnlock()
	sv.invalid = true
}

// Add ...
func (sv *SignatureVerifier) Add(s *Signature) {
	if sv.IsInvalid() {
		return
	}

	sv.Lock()
	defer sv.Unlock()
	sv.batch = append(sv.batch, s)
}

// Remove returns the first signature from the batch. Returns nil if batch is empty.
func (sv *SignatureVerifier) Remove() *Signature {
	sv.Lock()
	defer sv.Unlock()
	if len(sv.batch) == 0 {
		return nil
	}
	sign := sv.batch[0]
	sv.batch = sv.batch[1:len(sv.batch)]
	return sign
}

// Reset reset the signature verifier for reuse.
func (sv *SignatureVerifier) Reset() {
	sv.Lock()
	defer sv.Unlock()
	sv.init = false
	sv.batch = make([]*Signature, 0)
	sv.invalid = false
	sv.closeCh = make(chan struct{})
}

// Finish waits till batch is finished. Returns true if all the signatures are valid, Otherwise returns false.
func (sv *SignatureVerifier) Finish() bool {
	for {
		time.Sleep(100 * time.Millisecond)
		sv.Lock()
		if sv.invalid || len(sv.batch) == 0 {
			close(sv.closeCh)
			sv.Unlock()
			break
		}
		sv.RUnlock()
	}
	// Wait till start function to finish and then reset it.
	sv.Wait()
	isInvalid := sv.IsInvalid()
	sv.Reset()
	return !isInvalid
}

func (sig *Signature) verify() error {
	switch sig.KeyTypeID {
	case crypto.Ed25519Type:
		pubKey, err := ed25519.NewPublicKey(sig.PubKey)
		if err != nil {
			return fmt.Errorf("failed to fetch ed25519 public key: %s", err)
		}
		ok, err := pubKey.Verify(sig.Msg, sig.Sign)
		if err != nil || !ok {
			return fmt.Errorf("failed to verify ed25519 signature: %s", err)
		}
	case crypto.Sr25519Type:
		pubKey, err := sr25519.NewPublicKey(sig.PubKey)
		if err != nil {
			return fmt.Errorf("failed to fetch sr25519 public key: %s", err)
		}
		ok, err := pubKey.Verify(sig.Msg, sig.Sign)
		if err != nil || !ok {
			return fmt.Errorf("failed to verify sr25519 signature: %s", err)
		}
	case crypto.Secp256k1Type:
		ok := secp256k1.VerifySignature(sig.PubKey, sig.Msg, sig.Sign)
		if !ok {
			return fmt.Errorf("failed to verify secp256k1 signature")
		}
	}
	return nil
}
